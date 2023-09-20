package ledger

import (
	"sync"

	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/ledger/blkstorage/fsblkstorage"
	"github.com/rongzer/blockchain/common/log"
	"github.com/spf13/viper"
)

// Manager 账本管理器, 其中包含多个链的账本
type Manager struct {
	blkstorageProvider blkstorage.BlockStoreProvider
	ledgers            sync.Map // map[string]*Ledger
}

// NewManager 创建账本管理器
func NewManager(directory string) *Manager {
	log.Logger.Infof("KV type: %s. Database location is %s", viper.GetString("ledger.state.stateDatabase"), directory)
	return &Manager{
		blkstorageProvider: fsblkstorage.NewProvider(
			fsblkstorage.NewConf(directory, -1),
			&blkstorage.IndexConfig{
				AttrsToIndex: []blkstorage.IndexableAttr{
					blkstorage.IndexableAttrBlockNum,
					blkstorage.IndexableAttrBlockHash,
					blkstorage.IndexableAttrBlockNumTranNum,
					blkstorage.IndexableAttrBlockTxID,
				}},
		),
	}
}

// GetOrCreate 如果链的账本存在则返回, 不存在则先创建
func (m *Manager) GetOrCreate(chainID string) (*Ledger, error) {
	ledger, ok := m.ledgers.Load(chainID)
	if ok {
		return ledger.(*Ledger), nil
	}

	blockStore, err := m.blkstorageProvider.OpenBlockStore(chainID)
	if err != nil {
		return nil, err
	}

	ledger = &Ledger{blockStore: blockStore, signal: make(chan struct{})}
	m.ledgers.Store(chainID, ledger)
	return ledger.(*Ledger), nil
}

// ChainIDs 获取账本中所有链的ID, 也就是chains目录下子目录
func (m *Manager) ChainIDs() ([]string, error) {
	return m.blkstorageProvider.List()
}

// Close 释放资源
func (m *Manager) Close() {
	m.blkstorageProvider.Close()
}
