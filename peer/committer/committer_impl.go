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

package committer

import (
	"fmt"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/committer/txvalidator"
	"github.com/rongzer/blockchain/peer/events/producer"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

//--------!!!IMPORTANT!!-!!IMPORTANT!!-!!IMPORTANT!!---------
// This is used merely to complete the loop for the "skeleton"
// path so we can reason about and  modify committer component
// more effectively using code.

// LedgerCommitter is the implementation of  Committer interface
// it keeps the reference to the ledger to commit blocks and retrieve
// chain information
type LedgerCommitter struct {
	ledger      ledger.PeerLedger
	validator   txvalidator.Validator
	eventer     ConfigBlockEventer
	payloadSize int
}

// ConfigBlockEventer callback function proto type to define action
// upon arrival on new configuaration update block
type ConfigBlockEventer func(block *common.Block) error

// NewLedgerCommitterReactive is a factory function to create an instance of the committer
// same as way as NewLedgerCommitter, while also provides an option to specify callback to
// be called upon new configuration block arrival and commit event
func NewLedgerCommitterReactive(ledger ledger.PeerLedger, validator txvalidator.Validator, eventer ConfigBlockEventer) *LedgerCommitter {
	lc := &LedgerCommitter{ledger: ledger, validator: validator, eventer: eventer, payloadSize: 0}
	go lc.indexState()

	return lc
}

func (lc *LedgerCommitter) indexState() {
	var t time.Duration
	for {
		//未记账交易超过5块，暂时不进行索引处理
		indexNum := lc.ledger.IndexState()
		if indexNum > 10 {
			indexNum = 10
		}
		if lc.payloadSize > 50 {
			indexNum = 1
		}

		if indexNum == 0 {
			t = time.Millisecond * 20
		} else {
			t = time.Millisecond * time.Duration(indexNum*indexNum*10+5)
		}
		time.Sleep(t)
	}
}

// Commit commits block to into the ledger
// Note, it is important that this always be called serially
func (lc *LedgerCommitter) Commit(block *common.Block, payloadSize int) error {
	// Validate and mark invalid transactions
	log.Logger.Debugf("Validating block [%d]", block.Header.Number)
	lc.payloadSize = payloadSize
	//不在批量同步时，验证交易
	if payloadSize < 50 {
		// 若某"轻量区块"的交易数为0，则不进行验证
		if len(block.Data.Data) != 0 {
			if err := lc.validator.Validate(block); err != nil {
				return err
			}
		}
	}

	// Updating CSCC with new configuration block
	if utils.IsConfigBlock(block) {
		log.Logger.Debug("Received configuration update, calling CSCC ConfigUpdate")
		if err := lc.eventer(block); err != nil {
			return fmt.Errorf("Could not update CSCC with new configuration update due to %s", err)
		}
	}

	if err := lc.ledger.Commit(block); err != nil {
		return err
	}

	// send block event *after* the block has been committed
	if err := producer.SendProducerBlockEvent(block); err != nil {
		log.Logger.Errorf("Error publishing block %d, because: %v", block.Header.Number, err)
	}

	return nil
}

// LedgerHeight returns recently committed block sequence number
func (lc *LedgerCommitter) LedgerHeight() (uint64, error) {
	var info *common.BlockchainInfo
	var err error
	if info, err = lc.ledger.GetBlockchainInfo(); err != nil {
		log.Logger.Errorf("Cannot get blockchain info, %s\n", info)
		return uint64(0), err
	}

	return info.Height, nil
}

// GetBlocks used to retrieve blocks with sequence numbers provided in the slice
func (lc *LedgerCommitter) GetBlocks(blockSeqs []uint64) []*common.Block {
	var blocks []*common.Block

	for _, seqNum := range blockSeqs {
		if blck, err := lc.ledger.GetBlockByNumber(seqNum); err != nil {
			log.Logger.Errorf("Not able to acquire block num %d, from the ledger skipping...%v\n", seqNum, err)
			continue
		} else {
			log.Logger.Debug("Appending next block with seqNum = ", seqNum, " to the resulting set")
			blocks = append(blocks, blck)
		}
	}

	return blocks
}

// Close the ledger
func (lc *LedgerCommitter) Close() {
	lc.ledger.Close()
}
