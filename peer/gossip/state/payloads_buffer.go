/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package state

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	proto "github.com/rongzer/blockchain/protos/gossip"
)

// PayloadsBuffer is used to store payloads into which used to
// support payloads with blocks reordering according to the
// sequence numbers. It also will provide the capability
// to signal whenever expected block has arrived.
type PayloadsBuffer interface {
	// Adds new block into the buffer
	Push(payload *proto.Payload) error

	// Returns next expected sequence number
	Next() uint64

	// Remove and return payload with given sequence number
	Pop() *proto.Payload

	// Get current buffer size
	Size() int

	// Channel to indicate event when new payload pushed with sequence
	// number equal to the next expected value.
	Ready() chan struct{}

	Close()

	//判断payloadBuffer中数据是否存在，避免重复请求
	IsExist(start uint64, end uint64) (bool, uint64)
}

// PayloadsBufferImpl structure to implement PayloadsBuffer interface
// store inner state of available payloads and sequence numbers
type PayloadsBufferImpl struct {
	next uint64

	buf map[uint64]*proto.Payload

	readyChan chan struct{}

	mutex sync.RWMutex

	maxChain chan int // 最大缓冲数5000
}

// NewPayloadsBuffer is factory function to create new payloads buffer
func NewPayloadsBuffer(next uint64) PayloadsBuffer {
	return &PayloadsBufferImpl{
		buf:       make(map[uint64]*proto.Payload, 5000),
		readyChan: make(chan struct{}, 0),
		next:      next,
		maxChain:  make(chan int, 5000), // 最大缓冲数5000
	}
}

// Ready function returns the channel which indicates whenever expected
// next block has arrived and one could safely pop out
// next sequence of blocks
func (b *PayloadsBufferImpl) Ready() chan struct{} {
	return b.readyChan
}

// Push new payload into the buffer structure in case new arrived payload
// sequence number is below the expected next block number payload will be
// thrown away and error will be returned.
func (b *PayloadsBufferImpl) Push(payload *proto.Payload) error {
	b.mutex.Lock()

	seqNum := payload.SeqNum

	if seqNum < b.next || b.buf[seqNum] != nil {
		b.mutex.Unlock()
		return fmt.Errorf("Payload with sequence number = %s has been already processed",
			strconv.FormatUint(payload.SeqNum, 10))
	}

	b.buf[seqNum] = payload
	b.mutex.Unlock()

	// Send notification that next sequence has arrived
	if seqNum == b.next {
		// Do not block execution of current routine
		go func() {
			b.readyChan <- struct{}{}
		}()
	}
	b.maxChain <- 1 // 最大缓冲数5000

	return nil
}

// Next function provides the number of the next expected block
func (b *PayloadsBufferImpl) Next() uint64 {
	// Atomically read the value of the top sequence number
	return atomic.LoadUint64(&b.next)
}

// Pop function extracts the payload according to the next expected block
// number, if no next block arrived yet, function returns nil.
func (b *PayloadsBufferImpl) Pop() *proto.Payload {
	b.mutex.Lock()

	result := b.buf[b.Next()]

	if result != nil {
		// If there is such sequence in the buffer need to delete it
		delete(b.buf, b.Next())
		// Increment next expect block index
		atomic.AddUint64(&b.next, 1)
		b.mutex.Unlock()
		<-b.maxChain // 最大缓冲数5000
	} else {
		b.mutex.Unlock()
	}
	return result
}

// Size returns current number of payloads stored within buffer
func (b *PayloadsBufferImpl) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return len(b.buf)
}

//判断payloadBuffer中数据是否存在，避免重复请求
func (b *PayloadsBufferImpl) IsExist(start uint64, end uint64) (bool, uint64) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	bExist := true
	start1 := start
	for prev := start; prev <= end; prev++ {
		if prev < b.Next() {
			continue
		}

		if b.buf[prev] == nil {
			start1 = prev
			bExist = false
			break
		}
	}
	return bExist, start1
}

// Close cleanups resources and channels in maintained
func (b *PayloadsBufferImpl) Close() {
	close(b.readyChan)
}
