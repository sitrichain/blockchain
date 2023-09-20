package chain

import (
	"fmt"

	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/orderer/consensus"
	"github.com/rongzer/blockchain/orderer/filters"
	"github.com/rongzer/blockchain/orderer/ledger"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// Chain 链结构中包含了支撑一条链所需的资源
type Chain struct {
	*ledgerResources
	consenter       consensus.Consenter
	cutter          *consensus.Cutter
	filters         filters.Set
	signer          crypto.LocalSigner
	lastConfigIndex uint64
	lastConfigSeq   uint64
}

// 创建链
func newChain(set filters.Set, ledgerResources *ledgerResources, consenter consensus.Mode, signer crypto.LocalSigner) (*Chain, error) {
	c := &Chain{
		ledgerResources: ledgerResources,
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
	// 获取元数据
	metadata, err := utils.GetMetadataFromBlock(lastBlock, cb.BlockMetadataIndex_ORDERER)
	if err != nil {
		return nil, fmt.Errorf("[chain: %s] Error extracting orderer metadata: %w", c.ChainID(), err)
	}
	log.Logger.Infof("[chain: %s] Get metadata (blockNumber=%d, lastConfigIndex=%d, lastConfigSeq=%d): %s", c.ChainID(), lastBlock.Header.Number, c.lastConfigIndex, c.lastConfigSeq, metadata.Value)
	// 创建链的共识器
	c.consenter, err = consenter.NewConsenter(c, metadata)
	if err != nil {
		return nil, fmt.Errorf("Error creating consenter of chain: %s: %w", c.ChainID(), err)
	}

	return c, nil
}

// 启动链的共识器
func (c *Chain) start() {
	c.consenter.Start()
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
func (c *Chain) Ledger() *ledger.Ledger {
	return c.ledger
}

// Order 接收消息进入队列
func (c *Chain) Enqueue(env *cb.Envelope, committer filters.Committer) bool {
	return c.consenter.Order(env, committer)
}

// Errored 获取链共识器的错误通道
func (c *Chain) Errored() <-chan struct{} {
	return c.consenter.Errored()
}

// CreateNextBlock 使用传入的交易列表创建新块
func (c *Chain) CreateNextBlock(messages []*cb.Envelope) (*cb.Block, error) {
	return c.ledger.CreateNextBlock(messages)
}

// Height 获取账本的块高度
func (c *Chain) Height() uint64 {
	return c.Ledger().Height()
}

// LegderHeight 获取账本的块高度
func (c *Chain) LegderHeight() uint64 {
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

// WriteBlock 写入块数据
func (c *Chain) WriteBlock(block *cb.Block, committers []filters.Committer, encodedMetadataValue []byte) *cb.Block {
	// 块内的交易确认提交
	for _, committer := range committers {
		committer.Commit()
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
