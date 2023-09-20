package cluster

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/localmsp"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"time"

	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

const (
	// RetryTimeout is the time the block puller retries.
	RetryTimeout = time.Second * 10
)

// ChannelPredicate accepts channels according to their names.
type ChannelPredicate func(channelName string) bool

var TheReplicator *Replicator

// PullerConfigFromTopLevelConfig creates a PullerConfig from a TopLevel config,
// and from a signer and TLS key cert pair.
// The PullerConfig's channel is initialized to be the system channel.
func PullerConfigFromTopLevelConfig(
	systemChannel string,
	signer crypto.LocalSigner,
) PullerConfig {
	return PullerConfig{
		Channel:             systemChannel,
		MaxTotalBufferBytes: conf.V.Sealer.Raft.ReplicationBufferSize,
		Timeout:             conf.V.Sealer.Raft.RPCTimeout,
		Signer:              signer,
	}
}

// LedgerWriter allows the caller to write blocks and inspect the height
type LedgerWriter interface {
	// Append a new block to the ledger
	Append(block *common.Block) error

	// Height returns the number of blocks on the ledger
	Height() uint64
}

// LedgerFactory retrieves or creates new ledgers by chainID
type LedgerFactory interface {
	// GetOrCreate gets an existing ledger (if it exists)
	// or creates it if it does not
	GetOrCreate(chainID string) (LedgerWriter, error)
}

// ChannelLister returns channels with genesis block
type ChannelLister interface {
	// Channels returns channels with genesis block
	Channels(systemChainId string) map[string]ChannelGenesisBlock
}

// Replicator replicates chains
type Replicator struct {
	DoNotPanicIfClusterNotReachable bool
	SystemChannel                   string
	ChannelLister                   ChannelLister
	Logger                          *log.RaftLogger
	Puller                          *BlockPuller
	BootBlock                       *common.Block
}

func UpdateReplicator(systemChannelName string) (*Replicator, error) {
	pullerConfig := PullerConfigFromTopLevelConfig(systemChannelName, localmsp.NewSigner())
	puller, err := BlockPullerFromConfig(pullerConfig)
	if err != nil {
		log.Logger.Errorf("Failed creating puller config from bootstrap block: %v", err)
		return nil, err
	}
	puller.MaxPullBlockRetries = conf.V.Sealer.Raft.ReplicationMaxRetries
	puller.RetryTimeout = conf.V.Sealer.Raft.ReplicationRetryTimeout
	lastConfigBlock, err := PullLastBlockFromSystemChain(puller)
	if err != nil {
		log.Logger.Errorf("cannot pull last block from system chain, err is: %v", err)
		return nil, err
	}
	replicator := &Replicator{
		SystemChannel: systemChannelName,
		BootBlock:     lastConfigBlock,
		Logger:        log.NewRaftLogger(log.Logger),
		Puller:        puller,
		ChannelLister: &ChainInspector{
			Logger:          log.NewRaftLogger(log.Logger),
			Puller:          puller,
			LastConfigBlock: lastConfigBlock,
		},
	}
	TheReplicator = replicator

	return replicator, nil
}

func NewReplicator(systemChannelName string) (*Replicator, error) {
	pullerConfig := PullerConfigFromTopLevelConfig(systemChannelName, localmsp.NewSigner())
	puller, err := BlockPullerFromConfig(pullerConfig)
	if err != nil {
		log.Logger.Errorf("Failed creating puller config from bootstrap block: %v", err)
		return nil, err
	}
	puller.MaxPullBlockRetries = conf.V.Sealer.Raft.ReplicationMaxRetries
	puller.RetryTimeout = conf.V.Sealer.Raft.ReplicationRetryTimeout
	lastConfigBlock, err := PullLastBlockFromSystemChain(puller)
	if err != nil {
		log.Logger.Errorf("cannot pull last block from system chain, err is: %v", err)
		return nil, err
	}
	replicator := &Replicator{
		SystemChannel: systemChannelName,
		BootBlock:     lastConfigBlock,
		Logger:        log.NewRaftLogger(log.Logger),
		Puller:        puller,
		ChannelLister: &ChainInspector{
			Logger:          log.NewRaftLogger(log.Logger),
			Puller:          puller,
			LastConfigBlock: lastConfigBlock,
		},
	}
	TheReplicator = replicator

	// 定时更新TheReplicator
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				UpdateReplicator(systemChannelName)
			}
		}
	}()

	return replicator, nil
}

func (r *Replicator) ReplicateSystemChain() error {
	if err := r.PullChannel(r.SystemChannel); err != nil && err != ErrSkipped {
		r.Logger.Errorf("Failed pulling system channel: %v", err)
		return err
	}
	return nil
}

func isInMap(key string, m map[string]ChannelGenesisBlock) bool {
	for channel := range m {
		if key == channel {
			return true
		}
	}
	return false
}

// ReplicateSomeChain在joinChain的逻辑中调用
func (r *Replicator) ReplicateSomeChain(chainId string) error {
	existedChannels := r.findChannels()
	// 先获取该链的创世块
	channel := existedChannels[chainId]
	ledger, err := ledgermgmt.GetSignaledLedger(channel.ChannelName)
	if err != nil {
		r.Logger.Errorf("cannot get or create ledger for channel: %v in replicator", channel)
		return err
	}
	// appendBlock方法中已经做了高度判断，即不会出现同一个创世块被加入2次的情况
	r.appendBlock(channel.GenesisBlock, ledger, channel.ChannelName)
	// 拉取剩余区块
	err = r.PullChannel(channel.ChannelName)
	if err != nil {
		r.Logger.Errorf("Failed pulling channel %s: %v", channel.ChannelName, err)
		return err
	}
	return nil
}

func (r *Replicator) findChannels() map[string]ChannelGenesisBlock {
	r.Logger.Debug("Entering")
	defer r.Logger.Debug("Exiting")
	channels := r.ChannelLister.Channels(r.SystemChannel)
	r.Logger.Info("found", len(channels), "channels in local ledger of system channel, not including systemChain")
	return channels
}

// PullChannel pulls the given channel from some orderer,
// and commits it to the ledger.
func (r *Replicator) PullChannel(channel string) error {
	r.Logger.Infof("Pulling channel: %v", channel)
	puller := r.Puller.Clone()
	// 更换channel名称对应的logger
	puller.Logger = log.NewRaftLogger(log.Logger).With("channel", channel)
	defer puller.Close()
	puller.Channel = channel

	ledger, err := ledgermgmt.GetSignaledLedger(channel)
	if err != nil {
		r.Logger.Errorf("Failed to get the ledger for channel %s when pullChannel in replicator", channel)
		return fmt.Errorf("pull channel failed because ledger not ready")
	}

	endpoint, latestHeight, _ := latestHeightAndEndpoint(puller)
	if endpoint == "" {
		return errors.Errorf("failed obtaining the latest block for channel %s", channel)
	}
	r.Logger.Info("Latest block height for channel", channel, "is", latestHeight)
	// Ensure that if we pull the system channel, the latestHeight is bigger or equal to the
	// bootstrap block of the system channel.
	// Otherwise, we'd be left with a block gap.
	if channel == r.SystemChannel && latestHeight-1 < r.BootBlock.Header.Number {
		return errors.Errorf("latest height found among system channel(%s) orderers is %d, but the boot block's "+
			"sequence is %d", r.SystemChannel, latestHeight, r.BootBlock.Header.Number)
	}
	return r.pullChannelBlocks(channel, puller, latestHeight, ledger)
}

func (r *Replicator) pullChannelBlocks(channel string, puller *BlockPuller, latestHeight uint64, ledger LedgerWriter) error {
	nextBlockToPull := ledger.Height()
	if nextBlockToPull == latestHeight {
		r.Logger.Infof("Latest height found (%d) is equal to our height, skipping pulling channel %s", latestHeight, channel)
		return nil
	}
	// Pull the next block and remember its hash.
	nextBlock := puller.PullBlock(nextBlockToPull)
	if nextBlock == nil {
		return ErrRetryCountExhausted
	}
	r.appendBlock(nextBlock, ledger, channel)
	actualPrevHash := utils.BlockHeaderHash(nextBlock.Header)

	for seq := uint64(nextBlockToPull + 1); seq < latestHeight; seq++ {
		block := puller.PullBlock(seq)
		if block == nil {
			return ErrRetryCountExhausted
		}
		reportedPrevHash := block.Header.PreviousHash
		if !bytes.Equal(reportedPrevHash, actualPrevHash) {
			return errors.Errorf("block header mismatch on sequence %d, expected %x, got %x",
				block.Header.Number, actualPrevHash, reportedPrevHash)
		}
		actualPrevHash = utils.BlockHeaderHash(block.Header)
		if channel == r.SystemChannel && block.Header.Number == r.BootBlock.Header.Number {
			r.compareBootBlockWithSystemChannelLastConfigBlock(block)
			r.appendBlock(block, ledger, channel)
			// No need to pull further blocks from the system channel
			return nil
		}
		r.appendBlock(block, ledger, channel)
	}
	return nil
}

func (r *Replicator) appendBlock(block *common.Block, ledger LedgerWriter, channel string) {
	height := ledger.Height()
	if height > block.Header.Number {
		r.Logger.Infof("Skipping commit of block [%d] for channel %s because height is at %d", block.Header.Number, channel, height)
		return
	}
	if err := ledger.Append(block); err != nil {
		r.Logger.Panicf("Failed to write block [%d]: %v", block.Header.Number, err)
	}
	r.Logger.Infof("Committed block [%d] for channel %s", block.Header.Number, channel)
}

func (r *Replicator) compareBootBlockWithSystemChannelLastConfigBlock(block *common.Block) {
	// Overwrite the received block's data hash
	block.Header.DataHash = utils.BlockDataHash(block.Data)

	bootBlockHash := utils.BlockHeaderHash(r.BootBlock.Header)
	retrievedBlockHash := utils.BlockHeaderHash(block.Header)
	if bytes.Equal(bootBlockHash, retrievedBlockHash) {
		return
	}
	r.Logger.Panicf("Block header mismatch on last system channel block, expected %s, got %s",
		hex.EncodeToString(bootBlockHash), hex.EncodeToString(retrievedBlockHash))
}

// PullerConfig configures a BlockPuller.
type PullerConfig struct {
	Timeout             time.Duration
	Signer              crypto.LocalSigner
	Channel             string
	MaxTotalBufferBytes int
}

// BlockPullerFromConfigBlock returns a BlockPuller that doesn't verify signatures on blocks.
func BlockPullerFromConfig(config PullerConfig) (*BlockPuller, error) {
	var endpoints []EndpointCriteria
	lastBlock, err := FindLastBlockFromSystemChain()
	// 优先从账本中读取lastBlock，若为空，则从BootStrapEndPoint拉取
	if lastBlock == nil || err != nil {
		ec := EndpointCriteria{Endpoint: conf.V.Sealer.Raft.BootStrapEndPoint}
		endpoints = []EndpointCriteria{ec}
	} else {
		// 抽取元数据
		metadata, err := utils.GetMetadataFromBlock(lastBlock, common.BlockMetadataIndex_ORDERER)
		if err != nil {
			return nil, fmt.Errorf("cannot extract orderer metadata from last block of system chain")
		}
		// 从metadata中读取*etcdraft.BlockMetadata
		if metadata == nil || len(metadata.Value) == 0 {
			return nil, fmt.Errorf("there is no block metadata")
		}
		m := &etcdraft.BlockMetadata{}
		if err := proto.Unmarshal(metadata.Value, m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block's metadata into *etcdraft.BlockMetadata")
		}
		for _, addr := range m.ConsenterEndpoints {
			endpoints = append(endpoints, EndpointCriteria{Endpoint: addr})
		}
		// 最后再添加BootStrapEndPoint，以防元数据中的所有raft集群节点都连不上
		// 不用担心重复添加，因为会在endpointInfoBucket的byEndpoints()方法中去除重复的节点
		endpoints = append(endpoints, EndpointCriteria{conf.V.Sealer.Raft.BootStrapEndPoint})
	}

	clientConf := comm.ClientConfig{
		Timeout:      config.Timeout,
		AsyncConnect: true,
		KaOpts:       comm.DefaultKeepaliveOptions(),
	}

	dialer := &StandardDialer{
		Config: clientConf.Clone(),
	}

	return &BlockPuller{
		Logger:              log.NewRaftLogger(log.Logger).With("channel", config.Channel),
		Dialer:              dialer,
		MaxTotalBufferBytes: config.MaxTotalBufferBytes,
		Endpoints:           endpoints,
		RetryTimeout:        RetryTimeout,
		FetchTimeout:        config.Timeout,
		Channel:             config.Channel,
		Signer:              config.Signer,
	}, nil
}

// ChainPuller pulls blocks from a chain
type ChainPuller interface {
	// PullBlock pulls the given block from some orderer node
	PullBlock(seq uint64) *common.Block

	// HeightsByEndpoints returns the block heights by endpoints of orderers
	HeightsByEndpoints() (map[string]uint64, error)

	// Close closes the ChainPuller
	Close()
}

// ChainInspector walks over a chain
type ChainInspector struct {
	Logger          *log.RaftLogger
	Puller          ChainPuller
	LastConfigBlock *common.Block
}

// ErrSkipped denotes that replicating a chain was skipped
var ErrSkipped = errors.New("skipped")

// ErrForbidden denotes that an ordering node refuses sending blocks due to access control.
var ErrForbidden = errors.New("forbidden pulling the channel")

// ErrServiceUnavailable denotes that an ordering node is not servicing at the moment.
var ErrServiceUnavailable = errors.New("service unavailable")

// ErrNotInChannel denotes that an ordering node is not in the channel
var ErrNotInChannel = errors.New("not in the channel")

var ErrRetryCountExhausted = errors.New("retry attempts exhausted")

// SelfMembershipPredicate determines whether the caller is found in the given config block
type SelfMembershipPredicate func(configBlock *common.Block) error

// PullLastConfigBlock pulls the last configuration block, or returns an error on failure.
func PullLastBlockFromSystemChain(puller *BlockPuller) (*common.Block, error) {
	ledger, _ := ledgermgmt.GetSignaledLedger(conf.V.Sealer.SystemChainId)
	if ledger.Height() >= 1 {
		block, err := ledger.GetBlockByNumber(ledger.Height() - 1)
		if block == nil || err != nil {
			return nil, fmt.Errorf("cannot get last block from ledger, err is: %v", err)
		}
		return block, nil
	}

	endpoint, latestHeight, err := latestHeightAndEndpoint(puller)
	if err != nil {
		return nil, err
	}
	if endpoint == "" {
		return nil, fmt.Errorf("no endpoint discovered from cluster")
	}
	lastBlock := puller.PullBlock(latestHeight - 1)
	if lastBlock == nil {
		return nil, ErrRetryCountExhausted
	}
	return lastBlock, nil
}

func FindLastBlockFromSystemChain() (*common.Block, error) {
	ledger, _ := ledgermgmt.GetSignaledLedger(conf.V.Sealer.SystemChainId)
	if ledger.Height() == 0 {
		return nil, fmt.Errorf("no block in system chain's ledger")
	}
	block, err := ledger.GetBlockByNumber(ledger.Height() - 1)
	if block == nil || err != nil {
		return nil, fmt.Errorf("cannot find last block in system chain's ledger")
	}
	return block, nil
}

func latestHeightAndEndpoint(puller ChainPuller) (string, uint64, error) {
	var maxHeight uint64
	var mostUpToDateEndpoint string
	heightsByEndpoints, err := puller.HeightsByEndpoints()
	if err != nil {
		return "", 0, err
	}
	for endpoint, height := range heightsByEndpoints {
		if height >= maxHeight {
			maxHeight = height
			mostUpToDateEndpoint = endpoint
		}
	}
	return mostUpToDateEndpoint, maxHeight, nil
}

// Close closes the ChainInspector
func (ci *ChainInspector) Close() {
	ci.Puller.Close()
}

// ChannelGenesisBlock wraps a Block and its channel name
type ChannelGenesisBlock struct {
	ChannelName  string
	GenesisBlock *common.Block
}

// GenesisBlocks aggregates several ChannelGenesisBlocks
type GenesisBlocks []ChannelGenesisBlock

// Names returns the channel names all ChannelGenesisBlocks
func (gbs GenesisBlocks) Names() []string {
	var res []string
	for _, gb := range gbs {
		res = append(res, gb.ChannelName)
	}
	return res
}

// GetChannel returns GenesisBlock of some channel
func (ci *ChainInspector) GetChannel(chainId, systemChainId string) *ChannelGenesisBlock {
	lastConfigBlockNum := ci.LastConfigBlock.Header.Number
	var block *common.Block
	var prevHash []byte
	var err error
	ledger, _ := ledgermgmt.GetSignaledLedger(systemChainId)

	for seq := uint64(0); seq < lastConfigBlockNum+1; seq++ {
		block, err = ledger.GetBlockByNumber(seq)
		if block == nil || err != nil {
			ci.Logger.Errorf("Failed getting block [%d] from local system channel", seq)
			continue
		}
		ci.validateHashPointer(block, prevHash)
		// Set the previous hash for the next iteration
		prevHash = utils.BlockHeaderHash(block.Header)

		channel, gb, err := ExtractGenesisBlock(ci.Logger, block)
		if err != nil {
			// If we failed to inspect a block, something is wrong in the system chain
			// we're trying to pull, so abort.
			ci.Logger.Panicf("Failed extracting channel genesis block from config block: %v", err)
		}

		if channel == "" {
			ci.Logger.Info("Block", seq, "doesn't contain a new channel")
			continue
		}

		ci.Logger.Info("Block", seq, "contains channel", channel)
		if channel == chainId {
			return &ChannelGenesisBlock{
				ChannelName:  channel,
				GenesisBlock: gb,
			}
		}
	}

	return nil
}

// Channels returns the list of ChannelGenesisBlocks
// for all channels. Each such ChannelGenesisBlock contains
// the genesis block of the channel.
func (ci *ChainInspector) Channels(systemChainId string) map[string]ChannelGenesisBlock {
	channels := make(map[string]ChannelGenesisBlock)
	lastConfigBlockNum := ci.LastConfigBlock.Header.Number
	var block *common.Block
	var prevHash []byte
	var err error
	ledger, _ := ledgermgmt.GetSignaledLedger(systemChainId)

	for seq := uint64(0); seq < lastConfigBlockNum+1; seq++ {
		block, err = ledger.GetBlockByNumber(seq)
		if block == nil || err != nil {
			ci.Logger.Errorf("Failed getting block [%d] from local system channel", seq)
			continue
		}
		ci.validateHashPointer(block, prevHash)
		// Set the previous hash for the next iteration
		prevHash = utils.BlockHeaderHash(block.Header)

		channel, gb, err := ExtractGenesisBlock(ci.Logger, block)
		if err != nil {
			// If we failed to inspect a block, something is wrong in the system chain
			// we're trying to pull, so abort.
			ci.Logger.Panicf("Failed extracting channel genesis block from config block: %v", err)
		}

		if channel == "" {
			ci.Logger.Info("Block", seq, "doesn't contain a new channel")
			continue
		}

		ci.Logger.Info("Block", seq, "contains channel", channel)
		channels[channel] = ChannelGenesisBlock{
			ChannelName:  channel,
			GenesisBlock: gb,
		}
	}

	return channels
}

func (ci *ChainInspector) validateHashPointer(block *common.Block, prevHash []byte) {
	if prevHash == nil {
		return
	}
	if bytes.Equal(block.Header.PreviousHash, prevHash) {
		return
	}
	ci.Logger.Panicf("Claimed previous hash of block [%d] is %x but actual previous hash is %x",
		block.Header.Number, block.Header.PreviousHash, prevHash)
}

// ExtractGenesisBlock determines if a config block creates new channel, in which
// case it returns channel name, genesis block and nil error.
func ExtractGenesisBlock(logger *log.RaftLogger, block *common.Block) (string, *common.Block, error) {
	if block == nil {
		return "", nil, errors.New("nil block")
	}
	env, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return "", nil, err
	}
	payload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return "", nil, err
	}
	if payload.Header == nil {
		return "", nil, errors.New("nil header in payload")
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return "", nil, err
	}
	// The transaction is not orderer transaction
	if common.HeaderType(chdr.Type) != common.HeaderType_ORDERER_TRANSACTION {
		return "", nil, nil
	}
	systemChannelName := chdr.ChannelId
	innerEnvelope, err := utils.UnmarshalEnvelope(payload.Data)
	if err != nil {
		return "", nil, err
	}
	innerPayload, err := utils.UnmarshalPayload(innerEnvelope.Payload)
	if err != nil {
		return "", nil, err
	}
	if innerPayload.Header == nil {
		return "", nil, errors.New("inner payload's header is nil")
	}
	chdr, err = utils.UnmarshalChannelHeader(innerPayload.Header.ChannelHeader)
	if err != nil {
		return "", nil, err
	}
	// The inner payload's header should be a config transaction
	if common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		logger.Warnf("Expecting %s envelope in block, got %s", common.HeaderType_CONFIG, common.HeaderType(chdr.Type))
		return "", nil, nil
	}
	// In any case, exclude all system channel transactions
	if chdr.ChannelId == systemChannelName {
		logger.Warnf("Expecting config envelope in %s block to target a different "+
			"channel other than system channel '%s'", common.HeaderType_ORDERER_TRANSACTION, systemChannelName)
		return "", nil, nil
	}

	metadata := &common.BlockMetadata{
		Metadata: make([][]byte, 4),
	}
	metadata.Metadata[common.BlockMetadataIndex_SIGNATURES] = utils.MarshalOrPanic(&common.OrdererBlockMetadata{
		LastConfig: &common.LastConfig{Index: 0},
		// This is a genesis block, peer never verify this signature because we can't bootstrap
		// trust from an earlier block, hence there are no signatures here.
	})
	metadata.Metadata[common.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&common.Metadata{
		Value: utils.MarshalOrPanic(&common.LastConfig{Index: 0}),
		// This is a genesis block, peer never verify this signature because we can't bootstrap
		// trust from an earlier block, hence there are no signatures here.
	})

	blockdata := &common.BlockData{Data: [][]byte{payload.Data}}
	b := &common.Block{
		Header:   &common.BlockHeader{DataHash: utils.BlockDataHash(blockdata)},
		Data:     blockdata,
		Metadata: metadata,
	}
	return chdr.ChannelId, b, nil
}
