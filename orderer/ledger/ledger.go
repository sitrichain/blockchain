package ledger

import (
	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/log"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
)

var newestSeekPosition = &ab.SeekPosition{
	Type: &ab.SeekPosition_Newest{
		&ab.SeekNewest{},
	},
}

// Ledger 文件账本
type Ledger struct {
	blockStore blkstorage.BlockStore
	signal     chan struct{}
}

// Iterator 获取指定起始位置的迭代器, 同时返回开始的块序号
func (l *Ledger) Iterator(startPosition *ab.SeekPosition) (Iterator, uint64) {
	switch start := startPosition.Type.(type) {
	case *ab.SeekPosition_Oldest:
		return &ledgerIterator{ledger: l, blockNumber: 0}, 0
	case *ab.SeekPosition_Newest:
		info, err := l.blockStore.GetBlockchainInfo()
		if err != nil {
			log.Logger.Panic(err)
		}
		newestBlockNumber := info.Height - 1
		return &ledgerIterator{ledger: l, blockNumber: newestBlockNumber}, newestBlockNumber
	case *ab.SeekPosition_Specified:
		height := l.Height()
		if start.Specified.Number > height {
			return &notFoundErrorIterator{}, 0
		}
		return &ledgerIterator{ledger: l, blockNumber: start.Specified.Number}, start.Specified.Number
	default:
		return &notFoundErrorIterator{}, 0
	}
}

// Height 获取账本的块高度
func (l *Ledger) Height() uint64 {
	info, err := l.blockStore.GetBlockchainInfo()
	if err != nil {
		log.Logger.Panic(err)
	}
	return info.Height
}

// Append 追加一个块进入账本
func (l *Ledger) Append(block *cb.Block) error {
	err := l.blockStore.AddBlock(block)
	if err == nil {
		close(l.signal)
		l.signal = make(chan struct{})
	}
	return err
}

// GetBlockchainInfo 获取账本的基本信息
func (l *Ledger) GetBlockchainInfo() (*cb.BlockchainInfo, error) {
	return l.blockStore.GetBlockchainInfo()
}

// GetBlockByNumber 获取指定高度的块, 传入math.MaxUint64可获取最新的块
func (l *Ledger) GetBlockByNumber(blockNumber uint64) (*cb.Block, error) {
	return l.blockStore.RetrieveBlockByNumber(blockNumber)
}

// GetBlockByHash 获取指定哈希的块
func (l *Ledger) GetBlockByHash(blockHash []byte) (*cb.Block, error) {
	return l.blockStore.RetrieveBlockByHash(blockHash)
}

// GetBlockByTxID 获取包含指定交易的块
func (l *Ledger) GetBlockByTxID(txID string) (*cb.Block, error) {
	return l.blockStore.RetrieveBlockByTxID(txID)
}

// GetTxByID 获取指定交易ID的交易
func (l *Ledger) GetTxByID(txID string) (*cb.Envelope, error) {
	return l.blockStore.RetrieveTxByID(txID)
}

// GetAttach 获取附件数据
func (l *Ledger) GetAttach(attachKey string) string {
	return l.blockStore.GetAttach(attachKey)
}

// CreateNextBlock 使用传入的交易列表创建新块
func (l *Ledger) CreateNextBlock(messages []*cb.Envelope) (*cb.Block, error) {
	var nextBlockNumber uint64
	var previousBlockHash []byte

	if l.Height() > 0 {
		it, _ := l.Iterator(newestSeekPosition)
		block, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		nextBlockNumber = block.Header.Number + 1
		previousBlockHash = block.Header.Hash()
	}

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = msg.Marshal()
		if err != nil {
			return nil, err
		}
	}
	// Attachs不参与Hash
	data1 := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	for i, msg := range messages {
		msg1 := &cb.Envelope{Payload: msg.Payload, Signature: msg.Signature}
		data1.Data[i], err = msg1.Marshal()
		if err != nil {
			return nil, err
		}
	}

	block := cb.NewBlock(nextBlockNumber, previousBlockHash)
	block.Header.DataHash = data1.Hash()
	block.Data = data

	return block, nil
}
