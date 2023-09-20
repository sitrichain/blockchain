package ledger

import (
	"fmt"

	cb "github.com/rongzer/blockchain/protos/common"
)

var closedChan chan struct{}

func init() {
	closedChan = make(chan struct{})
	close(closedChan)
}

// Iterator 迭代器接口, 可流式读取块数据
type Iterator interface {
	Next() (*cb.Block, cb.Status, error)
	ReadyChan() <-chan struct{}
}

// 账本数据迭代器
type ledgerIterator struct {
	ledger      *Ledger
	blockNumber uint64
}

// ReadyChan 该通道将阻塞，直到有新的块数据
func (iter *ledgerIterator) ReadyChan() <-chan struct{} {
	if iter.blockNumber > iter.ledger.Height()-1 {
		return iter.ledger.signal
	}
	return closedChan
}

// Next 获取下一个块, 如果达到账本高度则等待 ,如果不存在则报错
func (iter *ledgerIterator) Next() (*cb.Block, cb.Status, error) {
	for {
		if iter.blockNumber < iter.ledger.Height() {
			block, err := iter.ledger.blockStore.RetrieveBlockByNumber(iter.blockNumber)
			if err != nil {
				return nil, cb.Status_SERVICE_UNAVAILABLE, err
			}
			iter.blockNumber++
			return block, cb.Status_SUCCESS, nil
		}
		<-iter.ledger.signal
	}
}

// notFoundErrorIterator 用于查找范围找不到时使用
type notFoundErrorIterator struct{}

// Next 固定返回错误
func (nfei *notFoundErrorIterator) Next() (*cb.Block, cb.Status, error) {
	return nil, cb.Status_NOT_FOUND, fmt.Errorf("not found")
}

// ReadyChan 固定返回一个关闭的通道
func (nfei *notFoundErrorIterator) ReadyChan() <-chan struct{} {
	return closedChan
}
