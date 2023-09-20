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

package ledgermgmt

import (
	"errors"
	"fmt"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/kvledger"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// ErrLedgerAlreadyOpened is thrown by a CreateLedger call if a ledger with the given id is already opened
var ErrLedgerAlreadyOpened = errors.New("Ledger already opened")

// ErrLedgerMgmtNotInitialized is thrown when ledger mgmt is used before initializing this
var ErrLedgerMgmtNotInitialized = errors.New("ledger mgmt should be initialized before using")

//var OpenedLedgers map[string]ledger.PeerLedger
var OpenedLedgers sync.Map   // map[string]ledger.PeerLedger
var SignaledLedgers sync.Map //map[string]*ledger.SignaledLedger
var LedgerProvider ledger.PeerLedgerProvider
var lock sync.Mutex
var initialized bool
var once sync.Once

// Initialize initializes ledgermgmt
func Initialize() {
	once.Do(func() {
		initialize()
	})
}

func initialize() {
	log.Logger.Info("Initializing ledger mgmt")
	lock.Lock()
	defer lock.Unlock()
	initialized = true
	provider, err := kvledger.NewProvider()
	if err != nil {
		panic(fmt.Errorf("Error in instantiating ledger provider: %s", err))
	}
	LedgerProvider = provider
	log.Logger.Info("ledger mgmt initialized")
}

// CreateLedger creates a new ledger with the given genesis block.
// This function guarantees that the creation of ledger and committing the genesis block would an atomic action
// The chain id retrieved from the genesis block is treated as a ledger id
func CreateLedger(genesisBlock *common.Block) (ledger.PeerLedger, error) {
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	id, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return nil, err
	}

	log.Logger.Infof("Creating ledger [%s] with genesis block", id)
	l, err := LedgerProvider.Create(genesisBlock)
	if err != nil {
		return nil, err
	}
	l = wrapLedger(id, l)
	OpenedLedgers.Store(id, l)

	log.Logger.Infof("Created ledger [%s] with genesis block", id)
	return l, nil
}

// OpenLedger returns a ledger for the given id
func OpenLedger(id string) (ledger.PeerLedger, error) {
	log.Logger.Infof("Opening ledger with id = %s", id)
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	_, ok := OpenedLedgers.Load(id)
	if ok {
		return nil, ErrLedgerAlreadyOpened
	}
	l, err := LedgerProvider.Open(id)
	if err != nil {
		return nil, err
	}
	l = wrapLedger(id, l)
	OpenedLedgers.Store(id, l)
	log.Logger.Infof("Opened ledger with id = %s", id)
	return l, nil
}

// GetLedgerIDs returns the ids of the ledgers created
func GetLedgerIDs() ([]string, error) {
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return nil, ErrLedgerMgmtNotInitialized
	}
	return LedgerProvider.List()
}

func GetSignaledLedger(chainID string) (*kvledger.SignaledLedger, error) {
	// 获取链的账本
	ledger, ok := SignaledLedgers.Load(chainID)
	if !ok {
		l, err := LedgerProvider.CreateWithChainID(chainID)
		if err != nil {
			return nil, err
		}
		kvLedger, ok := l.(*kvledger.KvLedger)
		if !ok {
			return nil, fmt.Errorf("peerLedger cannot convert to kvLedger for chainID %s", chainID)
		}
		signaledLedger := &kvledger.SignaledLedger{KvLedger: kvLedger, Signal: make(chan struct{})}
		SignaledLedgers.Store(chainID, signaledLedger)
		return signaledLedger, nil
	}
	signaledLedger, ok := ledger.(*kvledger.SignaledLedger)
	if !ok {
		return nil, fmt.Errorf("Found exist chainID %s but could not retrieve its ledger.", chainID)
	}
	return signaledLedger, nil
}

func wrapLedger(id string, l ledger.PeerLedger) ledger.PeerLedger {
	return &closableLedger{id, l}
}

// closableLedger extends from actual validated ledger and overwrites the Close method
type closableLedger struct {
	id string
	ledger.PeerLedger
}

// Close closes the actual ledger and removes the entries from opened ledgers map
func (l *closableLedger) Close() {
	lock.Lock()
	defer lock.Unlock()
	l.closeWithoutLock()
}

func (l *closableLedger) closeWithoutLock() {
	l.PeerLedger.Close()
	OpenedLedgers.Delete(l.id)
}

// Close closes all the opened ledgers and any resources held for ledger management
func Close() {
	log.Logger.Infof("Closing ledger mgmt")
	lock.Lock()
	defer lock.Unlock()
	if !initialized {
		return
	}
	OpenedLedgers.Range(func(key, value interface{}) bool {
		value.(*closableLedger).closeWithoutLock()
		return true
	})
	LedgerProvider.Close()
	log.Logger.Infof("ledger mgmt closed")
}
