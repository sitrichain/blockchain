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

package kvledger

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/ledger/blkstorage/fsblkstorage"
	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/ledger/util/leveldbhelper"
	"github.com/rongzer/blockchain/common/ledger/util/rocksdbhelper"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/history/historydb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/history/historydb/historyleveldb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb/stateleveldb"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	gr "github.com/tecbot/gorocksdb"
)

var (
	// ErrLedgerIDExists is thrown by a CreateLedger call if a ledger with the given id already exists
	ErrLedgerIDExists = errors.New("LedgerID already exists")
	// ErrNonExistingLedgerID is thrown by a OpenLedger call if a ledger with the given id does not exist
	ErrNonExistingLedgerID = errors.New("LedgerID does not exist")

	underConstructionLedgerKey = []byte("underConstructionLedgerKey")
	ledgerKeyPrefix            = []byte("l")
)

// Provider implements interface ledger.PeerLedgerProvider
type Provider struct {
	idStore            *idStore
	blockStoreProvider blkstorage.BlockStoreProvider
	vdbProvider        statedb.VersionedDBProvider
	historydbProvider  historydb.HistoryDBProvider
}

// NewProvider instantiates a new Provider.
// This is not thread-safe and assumed to be synchronized be the caller
func NewProvider() (ledger.PeerLedgerProvider, error) {

	log.Logger.Info("Initializing ledger provider")

	// Initialize the ID store (inventory of chainIds/ledgerIds)
	dbtype := viper.GetString("ledger.state.stateDatabase")
	var idStore *idStore
	var provider *Provider
	if dbtype == "rocksdb" {
		idStore = openIDStoreRocksDb(ledgerconfig.GetLedgerProviderPath())
	} else {
		idStore = openIDStoreLevelDb(ledgerconfig.GetLedgerProviderPath())
	}

	// Initialize the block storage
	attrsToIndex := []blkstorage.IndexableAttr{
		blkstorage.IndexableAttrBlockHash,
		blkstorage.IndexableAttrBlockNum,
		blkstorage.IndexableAttrTxID,
		blkstorage.IndexableAttrBlockNumTranNum,
		blkstorage.IndexableAttrBlockTxID,
		blkstorage.IndexableAttrTxValidationCode,
	}
	indexConfig := &blkstorage.IndexConfig{AttrsToIndex: attrsToIndex}
	blockStoreProvider := fsblkstorage.NewProvider(
		fsblkstorage.NewConf(ledgerconfig.GetBlockStorePath(), ledgerconfig.GetMaxBlockfileSize()),
		indexConfig)

	// Initialize the versioned database (state database)
	var vdbProvider statedb.VersionedDBProvider
	log.Logger.Debug("Constructing leveldb VersionedDBProvider")
	vdbProvider = stateleveldb.NewVersionedDBProvider()

	// Initialize the history database (index for history of values by key)
	var historydbProvider historydb.HistoryDBProvider
	historydbProvider = historyleveldb.NewHistoryDBProvider()

	log.Logger.Info("ledger provider Initialized")
	provider = &Provider{idStore, blockStoreProvider, vdbProvider, historydbProvider}
	provider.recoverUnderConstructionLedger()
	return provider, nil
}

// Create implements the corresponding method from interface ledger.PeerLedgerProvider
// This functions sets a under construction flag before doing any thing related to ledger creation and
// upon a successful ledger creation with the committed genesis block, removes the flag and add entry into
// created ledgers list (atomically). If a crash happens in between, the 'recoverUnderConstructionLedger'
// function is invoked before declaring the provider to be usable
func (provider *Provider) Create(genesisBlock *common.Block) (ledger.PeerLedger, error) {
	ledgerID, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return nil, err
	}
	exists, err := provider.idStore.ledgerIDExists(ledgerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrLedgerIDExists
	}
	if err = provider.idStore.setUnderConstructionFlag(ledgerID); err != nil {
		return nil, err
	}
	ldger, err := provider.openInternal(ledgerID)
	if err != nil {
		log.Logger.Errorf("Error in opening a new empty ledger. Unsetting under construction flag. Err: %s", err)
		panicOnErr(provider.runCleanup(ledgerID), "Error while running cleanup for ledger id [%s]", ledgerID)
		panicOnErr(provider.idStore.unsetUnderConstructionFlag(), "Error while unsetting under construction flag")
		return nil, err
	}
	if err := ldger.Commit(genesisBlock); err != nil {
		log.Logger.Errorf("commit genesis block to ledger error. %s", err)
		ldger.Close()
		return nil, err
	}
	panicOnErr(provider.idStore.createLedgerID(ledgerID, genesisBlock), "Error while marking ledger as created")
	return ldger, nil
}

// Open implements the corresponding method from interface ledger.PeerLedgerProvider
func (provider *Provider) Open(ledgerID string) (ledger.PeerLedger, error) {
	log.Logger.Debugf("Open() opening kvledger: %s", ledgerID)
	// Check the ID store to ensure that the chainId/ledgerId exists
	exists, err := provider.idStore.ledgerIDExists(ledgerID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNonExistingLedgerID
	}
	return provider.openInternal(ledgerID)
}

func (provider *Provider) openInternal(ledgerID string) (ledger.PeerLedger, error) {
	// Get the block store for a chain/ledger
	blockStore, err := provider.blockStoreProvider.OpenBlockStore(ledgerID)
	if err != nil {
		return nil, err
	}

	// Get the versioned database (state database) for a chain/ledger
	vDB, err := provider.vdbProvider.GetDBHandle(ledgerID)
	if err != nil {
		return nil, err
	}

	// Get the history database (index for history of values by key) for a chain/ledger
	historyDB, err := provider.historydbProvider.GetDBHandle(ledgerID)
	if err != nil {
		return nil, err
	}

	// Create a kvLedger for this chain/ledger, which encasulates the underlying data stores
	// (id store, blockstore, state database, history database)
	l, err := newKVLedger(ledgerID, blockStore, vDB, historyDB)
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Exists implements the corresponding method from interface ledger.PeerLedgerProvider
func (provider *Provider) Exists(ledgerID string) (bool, error) {
	return provider.idStore.ledgerIDExists(ledgerID)
}

// List implements the corresponding method from interface ledger.PeerLedgerProvider
func (provider *Provider) List() ([]string, error) {
	return provider.idStore.getAllLedgerIds()
}

// Close implements the corresponding method from interface ledger.PeerLedgerProvider
func (provider *Provider) Close() {
	provider.idStore.close()
	provider.blockStoreProvider.Close()
	provider.vdbProvider.Close()
	provider.historydbProvider.Close()
}

// recoverUnderConstructionLedger checks whether the under construction flag is set - this would be the case
// if a crash had happened during creation of ledger and the ledger creation could have been left in intermediate
// state. Recovery checks if the ledger was created and the genesis block was committed successfully then it completes
// the last step of adding the ledger id to the list of created ledgers. Else, it clears the under construction flag
func (provider *Provider) recoverUnderConstructionLedger() {
	log.Logger.Debugf("Recovering under construction ledger")
	ledgerID, err := provider.idStore.getUnderConstructionFlag()
	panicOnErr(err, "Error while checking whether the under construction flag is set")
	if ledgerID == "" {
		log.Logger.Debugf("No under construction ledger found. Quitting recovery")
		return
	}
	log.Logger.Infof("ledger [%s] found as under construction", ledgerID)
	ldger, err := provider.openInternal(ledgerID)
	panicOnErr(err, "Error while opening under construction ledger [%s]", ledgerID)
	bcInfo, err := ldger.GetBlockchainInfo()
	panicOnErr(err, "Error while getting blockchain info for the under construction ledger [%s]", ledgerID)
	ldger.Close()

	switch bcInfo.Height {
	case 0:
		log.Logger.Infof("Genesis block was not committed. Hence, the peer ledger not created. unsetting the under construction flag")
		panicOnErr(provider.runCleanup(ledgerID), "Error while running cleanup for ledger id [%s]", ledgerID)
		panicOnErr(provider.idStore.unsetUnderConstructionFlag(), "Error while unsetting under construction flag")
	case 1:
		log.Logger.Infof("Genesis block was committed. Hence, marking the peer ledger as created")
		genesisBlock, err := ldger.GetBlockByNumber(0)
		panicOnErr(err, "Error while retrieving genesis block from blockchain for ledger [%s]", ledgerID)
		panicOnErr(provider.idStore.createLedgerID(ledgerID, genesisBlock), "Error while adding ledgerID [%s] to created list", ledgerID)
	default:
		panic(fmt.Errorf(
			"Data inconsistency: under construction flag is set for ledger [%s] while the height of the blockchain is [%d]",
			ledgerID, bcInfo.Height))
	}
	return
}

// runCleanup cleans up blockstorage, statedb, and historydb for what
// may have got created during in-complete ledger creation
func (provider *Provider) runCleanup(_ string) error {
	// TODO - though, not having this is harmless for kv ledger.
	// If we want, following could be done:
	// - blockstorage could remove empty folders
	// - couchdb backed statedb could delete the database if got created
	// - leveldb backed statedb and history db need not perform anything as it uses a single db shared across ledgers
	return nil
}

func panicOnErr(err error, mgsFormat string, args ...interface{}) {
	if err == nil {
		return
	}
	args = append(args, err)
	panic(fmt.Sprintf(mgsFormat+" Err:%s ", args...))
}

//////////////////////////////////////////////////////////////////////
// Ledger id persistence related code
///////////////////////////////////////////////////////////////////////
type idStore struct {
	db util.IdStoreDb
}

func openIDStoreLevelDb(path string) *idStore {
	db := leveldbhelper.CreateDB(&leveldbhelper.Conf{DBPath: path})
	if err := db.Open(); err != nil {
		log.Logger.Errorf("Create leveldb error. %s", err)
	}
	return &idStore{db}
}

func openIDStoreRocksDb(path string) *idStore {
	db := rocksdbhelper.CreateDB(&rocksdbhelper.Conf{DBPath: path})
	if err := db.Open(); err != nil {
		log.Logger.Errorf("Create rocksdb error. %s", err)
	}
	return &idStore{db}
}

func (s *idStore) setUnderConstructionFlag(ledgerID string) error {
	return s.db.Put(underConstructionLedgerKey, []byte(ledgerID), true)
}

func (s *idStore) unsetUnderConstructionFlag() error {
	return s.db.Delete(underConstructionLedgerKey, true)
}

func (s *idStore) getUnderConstructionFlag() (string, error) {
	val, err := s.db.Get(underConstructionLedgerKey)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (s *idStore) createLedgerID(ledgerID string, gb *common.Block) error {
	key := s.encodeLedgerKey(ledgerID)
	var val []byte
	var err error
	if val, err = gb.Marshal(); err != nil {
		return err
	}
	if val, err = s.db.Get(key); err != nil {
		return err
	}
	if val != nil {
		return ErrLedgerIDExists
	}
	dbtype := viper.GetString("ledger.state.stateDatabase")
	var batch util.DbUpdateBatch
	if dbtype == "rocksdb" {
		batch = gr.NewWriteBatch()
	} else {
		batch = &leveldb.Batch{}
	}
	batch.Put(key, val)
	batch.Delete(underConstructionLedgerKey)
	return s.db.WriteBatch(batch, true)
}

func (s *idStore) ledgerIDExists(ledgerID string) (bool, error) {
	key := s.encodeLedgerKey(ledgerID)
	var val []byte
	err := error(nil)
	if val, err = s.db.Get(key); err != nil {
		return false, err
	}
	return val != nil, nil
}

func (s *idStore) getAllLedgerIds() ([]string, error) {
	var ids []string
	itr := s.db.GetIterator(nil, nil)
	for itr.SeekToFirst(); itr.Valid(); itr.Next() {
		if bytes.Equal(itr.Key(), underConstructionLedgerKey) {
			continue
		}
		id := s.decodeLedgerID(itr.Key())
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *idStore) close() {
	s.db.Close()
}

func (s *idStore) encodeLedgerKey(ledgerID string) []byte {
	return append(ledgerKeyPrefix, []byte(ledgerID)...)
}

func (s *idStore) decodeLedgerID(key []byte) string {
	return string(key[len(ledgerKeyPrefix):])
}
