package chain

import (
	"fmt"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/consensus"
	"github.com/rongzer/blockchain/peer/filters"
	"github.com/rongzer/blockchain/peer/ledger/kvledger"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// Chain 链结构中包含了支撑一条链所需的资源
type Chain struct {
	*LedgerResources
	Consenter       consensus.Consenter
	cutter          *consensus.Cutter
	filters         filters.Set
	signer          crypto.LocalSigner
	lastConfigIndex uint64
	lastConfigSeq   uint64
}

var chainInitializer func(string)

func SetChainInitializer(f func(string)) {
	chainInitializer = f
}

// Take care to initialize chain after peer joined, for example deploys system CCs
func InitChain(cid string) {
	if chainInitializer != nil {
		// Initialize chaincode, namely deploy system CC
		log.Logger.Infof("Init chain %s", cid)
		chainInitializer(cid)
	}
}

// 创建链
func newChain(set filters.Set, ledgerResources *LedgerResources, consenter consensus.Mode, signer crypto.LocalSigner) (*Chain, error) {
	c := &Chain{
		LedgerResources: ledgerResources,
		cutter:          consensus.NewCutter(ledgerResources.SharedConfig(), set),
		filters:         set,
		signer:          signer,
		lastConfigSeq:   ledgerResources.Sequence(),
	}

	lastBlock, err := c.GetBlockByNumber(c.Ledger().Height() - 1)
	if err != nil {
		return nil, fmt.Errorf("[chain: %s] Error get last block from block metadata: %w", c.ChainID(), err)
	}
	// 如果最新的块不是创世块, 则不存在lastconfig字段
	if lastBlock.Header.Number != 0 {
		lastConfigIndex, err := utils.GetLastConfigIndexFromBlock(lastBlock)
		if err != nil {
			return nil, fmt.Errorf("[chain: %s] Error extracting last config block from block metadata: %w", c.ChainID(), err)
		}
		c.lastConfigIndex = lastConfigIndex
	}
	if consenter != nil {
		// 获取元数据
		metadata, err := utils.GetMetadataFromBlock(lastBlock, cb.BlockMetadataIndex_ORDERER)
		if err != nil {
			return nil, fmt.Errorf("[chain: %s] Error extracting orderer metadata: %w", c.ChainID(), err)
		}
		log.Logger.Infof("[chain: %s] Get metadata (blockNumber=%d, lastConfigIndex=%d, lastConfigSeq=%d): %s", c.ChainID(), lastBlock.Header.Number, c.lastConfigIndex, c.lastConfigSeq, metadata.Value)
		// 创建链的共识器
		c.Consenter, err = consenter.NewConsenter(c, metadata)
		if err != nil {
			return nil, fmt.Errorf("Error creating Consenter of chain: %s: %w", c.ChainID(), err)
		}
	}
	return c, nil
}

// 启动链的共识器
func (c *Chain) Start() {
	c.Consenter.Start()
}

// NewSignatureHeader 创建签名头
func (c *Chain) NewSignatureHeader() (*cb.SignatureHeader, error) {
	return c.signer.NewSignatureHeader()
}

// Sign 对数据签名
func (c *Chain) Sign(message []byte) ([]byte, error) {
	return c.signer.Sign(message)
}

// Filters 获取链的过滤器集合
func (c *Chain) Filters() filters.Set {
	return c.filters
}

// BlockCutter 获取链的切块对象
func (c *Chain) BlockCutter() *consensus.Cutter {
	return c.cutter
}

// Ledger 获取链的账本
func (c *Chain) Ledger() *kvledger.SignaledLedger {
	return c.ledger
}

// Order 接收消息进入队列
func (c *Chain) Enqueue(env *cb.Envelope, committer filters.Committer) bool {
	return c.Consenter.Order(env, committer)
}

// Errored 获取链共识器的错误通道
func (c *Chain) Errored() <-chan struct{} {
	return c.Consenter.Errored()
}

// Height 获取账本的块高度
func (c *Chain) Height() uint64 {
	return c.Ledger().Height()
}

// GetBlockchainInfo 获取账本的基本信息
func (c *Chain) GetBlockchainInfo() (*cb.BlockchainInfo, error) {
	return c.ledger.GetBlockchainInfo()
}

// GetBlockByNumber 获取指定高度的块, 传入math.MaxUint64可获取最新的块
func (c *Chain) GetBlockByNumber(blockNumber uint64) (*cb.Block, error) {
	return c.ledger.GetBlockByNumber(blockNumber)
}

// GetBlockByHash 获取指定哈希的块
func (c *Chain) GetBlockByHash(blockHash []byte) (*cb.Block, error) {
	return c.ledger.GetBlockByHash(blockHash)
}

// GetBlockByTxID 获取包含指定交易的块
func (c *Chain) GetBlockByTxID(txID string) (*cb.Block, error) {
	return c.ledger.GetBlockByTxID(txID)
}

// GetTxByID 获取指定交易ID的交易
func (c *Chain) GetTxByID(txID string) (*cb.Envelope, error) {
	return c.ledger.GetTxByID(txID)
}

// GetAttach 获取附件数据
func (c *Chain) GetAttach(attachKey string) string {
	return c.ledger.GetAttach(attachKey)
}

// WriteBlock 写入块数据(且更新块的元数据)
func (c *Chain) WriteBlock(block *cb.Block, encodedMetadataValue []byte) *cb.Block {
	// 针对raft共识下对不同类型交易（配置更新、创建链、合约业务）的处理
	configEnv, txType, ableToCreateChain, err := utils.GetConfigEnvelopeFromBlock(block)
	// 创建链交易
	if txType == 4 && ableToCreateChain {
		manager.NewChain(configEnv)
	}
	// 其他两类交易不进行任何处理
	if txType == 1 || err != nil {
		log.Logger.Debugf("nothing to do for this kind of transaction: %v", txType)
	}

	// 写入orderer相关的元数据
	if encodedMetadataValue != nil {
		block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = utils.MarshalOrPanic(&cb.Metadata{Value: encodedMetadataValue})
	}
	// 写入签名
	c.addBlockSignature(block)
	// 写入最新配置块索引
	c.addLastConfigIndex(block)
	// 块数据入账本
	if err := c.ledger.Append(block); err != nil {
		log.Logger.Panicf("[chain: %s] Could not append block: %s", c.ChainID(), err)
	}
	log.Logger.Debugf("[chain: %s] Wrote block %d", c.ChainID(), block.GetHeader().Number)

	return block
}

// 添加块签名
func (c *Chain) addBlockSignature(block *cb.Block) {
	// 创建签名头并序列化
	if c.signer == nil {
		panic("Invalid signer. Must be different from nil.")
	}
	signatureHeader, err := c.signer.NewSignatureHeader()
	if err != nil {
		panic(err)
	}
	headerBytes, err := signatureHeader.Marshal()
	if err != nil {
		panic(err)
	}

	// 对签名头+块头签名
	s, err := c.signer.Sign(util.ConcatenateBytes(headerBytes, block.Header.Bytes()))
	if err != nil {
		panic(err)
	}

	// 序列化元数据项
	md := &cb.Metadata{
		Value:      []byte(nil), // 该值故意为空，因为此元数据仅与签名有关，不需要其他元数据信息
		Signatures: []*cb.MetadataSignature{{SignatureHeader: headerBytes, Signature: s}},
	}
	m, err := md.Marshal()
	if err != nil {
		panic(err)
	}

	// 写入元数据项到块中
	block.Metadata.Metadata[cb.BlockMetadataIndex_SIGNATURES] = m
}

// 写入最新配置块索引及签名
func (c *Chain) addLastConfigIndex(block *cb.Block) {
	// 创建签名头并序列化
	if c.signer == nil {
		panic("Invalid signer. Must be different from nil.")
	}
	signatureHeader, err := c.signer.NewSignatureHeader()
	if err != nil {
		panic(err)
	}
	headerBytes, err := signatureHeader.Marshal()
	if err != nil {
		panic(err)
	}

	// 获取当前最新配置序号, 如果比之前的记录新则更新序号及块号
	seq := c.Sequence()
	if seq > c.lastConfigSeq {
		log.Logger.Debugf("[chain: %s] Detected lastConfigSeq transitioning from %d to %d, setting lastConfigIndex from %d to %d", c.ChainID(), c.lastConfigSeq, seq, c.lastConfigIndex, block.Header.Number)
		c.lastConfigIndex = block.Header.Number
		c.lastConfigSeq = seq
	}

	log.Logger.Debugf("[chain: %s] About to write block, setting its LAST_CONFIG to %d", c.ChainID(), c.lastConfigIndex)

	// 序列化配置索引数据
	conf := &cb.LastConfig{Index: c.lastConfigIndex}
	confBytes, err := conf.Marshal()
	if err != nil {
		panic(err)
	}

	// 对配置数据+签名头+块头签名
	s, err := c.signer.Sign(util.ConcatenateBytes(confBytes, headerBytes, block.Header.Bytes()))
	if err != nil {
		panic(err)
	}

	// 序列化元数据项
	md := &cb.Metadata{
		Value:      confBytes,
		Signatures: []*cb.MetadataSignature{{SignatureHeader: headerBytes, Signature: s}},
	}
	m, err := md.Marshal()
	if err != nil {
		panic(err)
	}

	// 写入元数据项到块中
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = m
}
