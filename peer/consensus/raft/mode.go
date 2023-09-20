package raft

import (
	"code.cloudfoundry.org/clock"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/consensus"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"github.com/rongzer/blockchain/protos/utils"
	"go.etcd.io/etcd/raft"
	"google.golang.org/grpc"
	"path"
	"reflect"
	"strconv"
	"strings"
)

// ChainGetter obtains instances of ChainSupport for the given channel
type ChainGetter interface {
	// GetChain obtains the ChainSupport for the given channel.
	// Returns nil, false when the ChainSupport for the given channel
	// isn't found.
	GetChain(chainID string) (*chain.Chain, bool)
}

// Config contains etcdraft configurations
type Config struct {
	WALDir            string // WAL data of <my-channel> is stored in WALDir/<my-channel>
	SnapDir           string // Snapshots of <my-channel> are stored in SnapDir/<my-channel>
	EvictionSuspicion string // Duration threshold that the node samples in order to suspect its eviction from the channel.
}

// mode implements raft Consenter
type mode struct {
	Dialer        *cluster.PredicateDialer
	Communication cluster.Communicator
	*Dispatcher
	Chains     ChainGetter
	Logger     *log.RaftLogger
	RaftConfig *Config
	BCCSP      bccsp.BCCSP
}

// ReceiverByChain returns the MessageReceiver for the given channelID or nil
// if not found.
func (m *mode) ReceiverByChain(channelID string) MessageReceiver {
	cs, ok := m.Chains.GetChain(channelID)
	if cs == nil || !ok {
		return nil
	}

	if etcdRaftChain, isEtcdRaftChain := cs.Consenter.(*Consenter); isEtcdRaftChain {
		return etcdRaftChain
	}
	log.Logger.Warnf("Chain %s is of type %v and not raft.Consenter", channelID, reflect.TypeOf(cs.Consenter))
	return nil
}

func (m *mode) detectSelfID(consenters map[uint64]*etcdraft.Consenter) (uint64, error) {
	for nodeID, cst := range consenters {
		if fmt.Sprintf("%v:%v", cst.Host, cst.Port) == conf.V.Sealer.Raft.EndPoint {
			return nodeID, nil
		}
	}
	log.Logger.Warn("Could not find myself in raft cluster")
	return 0, cluster.ErrNotInChannel
}

func determineIdFromEndpoint(consenters map[uint64]*etcdraft.Consenter, endPoint string) (uint64, error) {
	for nodeID, cst := range consenters {
		if fmt.Sprintf("%v:%v", cst.Host, cst.Port) == endPoint {
			return nodeID, nil
		}
	}
	log.Logger.Warnf("Could not find endPoint: %v in raft cluster", endPoint)
	return 0, cluster.ErrNotInChannel
}

// HandleChain returns a new Consenter instance or an error upon failure
func (m *mode) NewConsenter(chain consensus.ChainResource, metadata *common.Metadata) (consensus.Consenter, error) {
	blockMetadata, configMeta, err := ReadBlockMetadata(metadata)
	if err != nil {
		log.Logger.Errorf("there is no Raft metadata to pass to NewConsenter")
		return nil, errors.Wrapf(err, "failed to read Raft metadata")
	}
	var envelope *common.Envelope
	if configMeta != nil {
		payloadChannelHeader := utils.MakeChannelHeader(common.HeaderType_CONFIG, chain.ChainID())
		payloadSignatureHeader := utils.MakeSignatureHeader(nil, utils.CreateNonceOrPanic())
		utils.SetTxID(payloadChannelHeader, payloadSignatureHeader)
		payloadHeader := utils.MakePayloadHeader(payloadChannelHeader, payloadSignatureHeader)
		payload := &common.Payload{Header: payloadHeader, Data: utils.MarshalOrPanic(configMeta)}
		envelope = &common.Envelope{Payload: utils.MarshalOrPanic(payload), Signature: nil}
	}
	consenters := CreateConsentersMap(blockMetadata)
	id, err := m.detectSelfID(consenters)
	if err != nil {
		return nil, err
	}

	opts := Options{
		RaftID:        id,
		Clock:         clock.NewClock(),
		MemoryStorage: raft.NewMemoryStorage(),
		Logger:        m.Logger,

		TickInterval:         conf.V.Sealer.Raft.TickInterval,
		ElectionTick:         conf.V.Sealer.Raft.ElectionTick,
		HeartbeatTick:        conf.V.Sealer.Raft.HeartbeatTick,
		MaxInflightBlocks:    conf.V.Sealer.Raft.MaxInflightBlocks,
		MaxSizePerMsg:        uint64(chain.SharedConfig().BatchSize().PreferredMaxBytes),
		SnapshotIntervalSize: conf.V.Sealer.Raft.SnapshotIntervalSize,

		BlockMetadata: blockMetadata,
		Consenters:    consenters,

		WALDir:  path.Join(m.RaftConfig.WALDir, chain.ChainID()),
		SnapDir: path.Join(m.RaftConfig.SnapDir, chain.ChainID()),
	}

	rpc := &cluster.RPC{
		Timeout:       conf.V.Sealer.Raft.RPCTimeout,
		Logger:        m.Logger,
		Channel:       chain.ChainID(),
		Comm:          m.Communication,
		StreamsByType: cluster.NewStreamsByType(),
	}

	return NewChain(
		chain,
		envelope,
		opts,
		m.Communication,
		rpc,
		m.BCCSP,
		func() (BlockPuller, error) {
			return NewBlockPuller(chain, m.Dialer, blockMetadata)
		},
		nil,
	)
}

// New creates a etcdraft Consenter
func Init(
	dialer *cluster.PredicateDialer,
	srv *grpc.Server,
	manager *chain.Manager,
	bccsp bccsp.BCCSP,
) consensus.Mode {

	cfg := &Config{SnapDir: conf.V.Sealer.Raft.SnapDir,
		WALDir:            conf.V.Sealer.Raft.WALDir,
		EvictionSuspicion: conf.V.Sealer.Raft.EvictionSuspicion,
	}
	raftLog := log.NewRaftLogger(log.Logger)
	mode := &mode{
		Logger:     raftLog,
		Chains:     manager,
		RaftConfig: cfg,
		Dialer:     dialer,
		BCCSP:      bccsp,
	}
	mode.Dispatcher = &Dispatcher{
		Logger:        log.Logger,
		ChainSelector: mode,
	}

	comm := createComm(dialer, mode)
	mode.Communication = comm
	svc := &cluster.Service{
		StreamCountReporter: &cluster.StreamCountReporter{},
		StepLogger:          raftLog,
		Logger:              raftLog,
		Dispatcher:          comm,
	}
	orderer.RegisterClusterServer(srv, svc)
	return mode
}

// ReadBlockMetadata attempts to read raft metadata from block metadata, if available.
// otherwise, it reads raft metadata from config metadata supplied.
func ReadBlockMetadata(blockMetadata *common.Metadata) (*etcdraft.BlockMetadata, *etcdraft.ConfigMetadata, error) {
	// 从blockMetadata中读取*etcdraft.BlockMetadata
	if blockMetadata == nil || len(blockMetadata.Value) == 0 {
		return nil, nil, fmt.Errorf("there is no block metadata")
	}
	m := &etcdraft.BlockMetadata{}
	if err := proto.Unmarshal(blockMetadata.Value, m); err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal block's metadata")
	}
	maybeAddedEndpoint := conf.V.Sealer.Raft.EndPoint
	for _, endPoint := range m.ConsenterEndpoints {
		if endPoint == maybeAddedEndpoint {
			return m, nil, nil
		}
	}
	m.ConsenterEndpoints = append(m.ConsenterEndpoints, maybeAddedEndpoint)
	m.PeerEndpoints = append(m.PeerEndpoints, conf.V.Peer.Endpoint)
	m.NextConsenterId = m.NextConsenterId + 1
	// 若有新增节点，则由*etcdraft.BlockMetadata构造*etcdraft.ConfigMetadata
	var consentersFromConfig []*etcdraft.Consenter
	for i, endPoint := range m.ConsenterEndpoints {
		components := strings.Split(endPoint, ":")
		host := components[0]
		port, _ := strconv.Atoi(components[1])
		peerPortStr := strings.Split(m.PeerEndpoints[i], ":")[1]
		peerPort, _ := strconv.Atoi(peerPortStr)
		consentersFromConfig = append(consentersFromConfig, &etcdraft.Consenter{Host: host, Port: int32(port), PeerPort: int32(peerPort)})
	}
	options := &etcdraft.Options{
		TickInterval:         conf.V.Sealer.Raft.TickInterval.String(),
		ElectionTick:         uint32(conf.V.Sealer.Raft.ElectionTick),
		HeartbeatTick:        uint32(conf.V.Sealer.Raft.HeartbeatTick),
		MaxInflightBlocks:    uint32(conf.V.Sealer.Raft.MaxInflightBlocks),
		SnapshotIntervalSize: conf.V.Sealer.Raft.SnapshotIntervalSize,
	}
	// 构造配置元数据
	configMeta := &etcdraft.ConfigMetadata{
		Consenters: consentersFromConfig,
		Options:    options,
	}

	return m, configMeta, nil
}

// CreateConsentersMap creates a map of Raft Node IDs to Consenter given the block metadata and the config metadata.
func CreateConsentersMap(blockMetadata *etcdraft.BlockMetadata) map[uint64]*etcdraft.Consenter {
	consenters := map[uint64]*etcdraft.Consenter{}
	for index, endPoint := range blockMetadata.ConsenterEndpoints {
		components := strings.Split(endPoint, ":")
		host := components[0]
		port, _ := strconv.Atoi(components[1])
		peerPortStr := strings.Split(blockMetadata.PeerEndpoints[index], ":")[1]
		peerPort, _ := strconv.Atoi(peerPortStr)
		consenters[uint64(index+1)] = &etcdraft.Consenter{Host: host, Port: int32(port), PeerPort: int32(peerPort)}
	}
	return consenters
}

func createComm(clusterDialer *cluster.PredicateDialer, m *mode) *cluster.Comm {
	comm := &cluster.Comm{
		SendBufferSize: cluster.SendBufferSize,
		Logger:         log.NewRaftLogger(log.Logger),
		Chan2Members:   make(map[string]cluster.MemberMapping),
		Connections:    cluster.NewConnectionStore(clusterDialer),
		H:              m,
	}
	m.Communication = comm
	return comm
}
