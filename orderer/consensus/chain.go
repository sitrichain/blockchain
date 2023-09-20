package consensus

import (
	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/orderer/filters"
	"github.com/rongzer/blockchain/orderer/ledger"
	cb "github.com/rongzer/blockchain/protos/common"
)

// Chain 共识模式内使用链相关资源的抽象
type ChainResource interface {
	crypto.LocalSigner
	BlockCutter() *Cutter
	SharedConfig() config.Orderer
	CreateNextBlock(messages []*cb.Envelope) (*cb.Block, error)
	WriteBlock(block *cb.Block, committers []filters.Committer, encodedMetadataValue []byte) *cb.Block
	ChainID() string
	Height() uint64
}

// 链接口, 解耦用
type Chain interface {
	ChainResource
	// Enqueue 接收消息进入队列
	Enqueue(env *cb.Envelope, committer filters.Committer) bool
	// Filters 获取链的过滤器集合
	Filters() filters.Set
	// LegderHeight 获取账本的块高度
	LegderHeight() uint64
	// GetBlockchainInfo 获取账本的基本信息
	GetBlockchainInfo() (*cb.BlockchainInfo, error)
	// GetBlockByNumber 获取指定高度的块, 传入math.MaxUint64可获取最新的块
	GetBlockByNumber(blockNumber uint64) (*cb.Block, error)
	// GetBlockByHash 获取指定哈希的块
	GetBlockByHash(blockHash []byte) (*cb.Block, error)
	// GetBlockByTxID 获取包含指定交易的块
	GetBlockByTxID(txID string) (*cb.Block, error)
	// GetTxByID 获取指定交易ID的交易
	GetTxByID(txID string) (*cb.Envelope, error)
	// GetAttach 获取附件数据
	GetAttach(attachKey string) string
	// PolicyManager 获取链的策略管理器
	PolicyManager() policies.Manager
	// Ledger 获取链的账本
	Ledger() *ledger.Ledger
	// Errored 获取共识器的错误通道
	Errored() <-chan struct{}
	// Sequence 获取最新配置序号
	Sequence() uint64
	// ProposeConfigUpdate 根据CONFIG_UPDATE消息, 生成对应配置消息
	ProposeConfigUpdate(env *cb.Envelope) (*cb.ConfigEnvelope, error)
}
