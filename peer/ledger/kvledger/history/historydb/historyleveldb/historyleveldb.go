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

package historyleveldb

import (
	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	comutils "github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/ledger/util/leveldbhelper"
	"github.com/rongzer/blockchain/common/ledger/util/rocksdbhelper"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/history/historydb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/rongzer/blockchain/peer/ledger/util"
	"github.com/rongzer/blockchain/protos/common"
	putils "github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
)

var savePointKey = []byte{0x00}

// HistoryDBProvider implements interface HistoryDBProvider
type HistoryDBProvider struct {
	dbProvider comutils.Provider
}

// NewHistoryDBProvider instantiates HistoryDBProvider
func NewHistoryDBProvider() *HistoryDBProvider {
	var dbPath string
	var dbProvider comutils.Provider
	dbtype := viper.GetString("ledger.state.stateDatabase")
	if dbtype == "rocksdb" {
		dbPath = ledgerconfig.GetHistoryRocksDBPath()
		dbProvider = rocksdbhelper.NewProvider(&rocksdbhelper.Conf{DBPath: dbPath})
	} else {
		dbPath = ledgerconfig.GetHistoryLevelDBPath()
		dbProvider = leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	}
	log.Logger.Debugf("constructing HistoryDBProvider dbPath=%s", dbPath)
	return &HistoryDBProvider{dbProvider}
}

// GetDBHandle gets the handle to a named database
func (provider *HistoryDBProvider) GetDBHandle(dbName string) (historydb.HistoryDB, error) {
	return newHistoryDB(provider.dbProvider.GetDBHandle(dbName), dbName), nil
}

// Close closes the underlying db
func (provider *HistoryDBProvider) Close() {
	provider.dbProvider.Close()
}

// historyDB implements HistoryDB interface
type historyDB struct {
	db     comutils.DBHandle
	dbName string
}

// newHistoryDB constructs an instance of HistoryDB
func newHistoryDB(db comutils.DBHandle, dbName string) *historyDB {
	return &historyDB{db, dbName}
}

// Open implements method in HistoryDB interface
func (historyDB *historyDB) Open() error {
	// do nothing because shared db is used
	return nil
}

// Close implements method in HistoryDB interface
func (historyDB *historyDB) Close() {
	// do nothing because shared db is used
}

// Commit implements method in HistoryDB interface
func (historyDB *historyDB) Commit(block *common.Block) error {

	blockNo := block.Header.Number
	//Set the starting tranNo to 0
	var tranNo uint64

	dbBatch := comutils.NewUpdateBatch()

	log.Logger.Debugf("Channel [%s]: Updating history database for blockNo [%v] with [%d] transactions",
		historyDB.dbName, blockNo, len(block.Data.Data))

	// Get the invalidation byte array for the block
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	// Initialize txsFilter if it does not yet exist (e.g. during testing, for genesis block, etc)
	if len(txsFilter) == 0 {
		txsFilter = util.NewTxValidationFlags(len(block.Data.Data))
		block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
	}

	// write each tran's write set to history db
	for _, envBytes := range block.Data.Data {

		// If the tran is marked as invalid, skip it
		if txsFilter.IsInvalid(int(tranNo)) {
			log.Logger.Debugf("Channel [%s]: Skipping history write for invalid transaction number %d",
				historyDB.dbName, tranNo)
			tranNo++
			continue
		}

		env, err := putils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			return err
		}

		payload, err := putils.GetPayload(env)
		if err != nil {
			return err
		}

		chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return err
		}

		if common.HeaderType(chdr.Type) == common.HeaderType_ENDORSER_TRANSACTION {

			// extract actions from the envelope message
			respPayload, err := putils.GetActionFromEnvelope(envBytes)
			if err != nil {
				return err
			}

			//preparation for extracting RWSet from transaction
			txRWSet := &rwsetutil.TxRwSet{}

			// Get the Result from the Action and then Unmarshal
			// it into a TxReadWriteSet using custom unmarshalling
			if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
				return err
			}
			// for each transaction, loop through the namespaces and writesets
			// and add a history record for each write
			for _, nsRWSet := range txRWSet.NsRwSets {
				ns := nsRWSet.NameSpace

				for _, kvWrite := range nsRWSet.KvRwSet.Writes {
					writeKey := kvWrite.Key

					//composite key for history records is in the form ns~key~blockNo~tranNo
					compositeHistoryKey := historydb.ConstructCompositeHistoryKey(ns, writeKey, blockNo, tranNo)

					// No value is required, write an empty byte array (emptyValue) since Put() of nil is not allowed
					dbBatch.Put(compositeHistoryKey, []byte{})
				}
			}

		} else {
			log.Logger.Debugf("Skipping transaction [%d] since it is not an endorsement transaction\n", tranNo)
		}
		tranNo++
	}

	// add savepoint for recovery purpose
	height := version.NewHeight(blockNo, tranNo)
	dbBatch.Put(savePointKey, height.ToBytes())

	// write the block's history records and savepoint to LevelDB
	if err := historyDB.db.WriteBatch(dbBatch, false); err != nil {
		return err
	}

	log.Logger.Debugf("Channel [%s]: Updates committed to history database for blockNo [%v]", historyDB.dbName, blockNo)
	return nil
}

// NewHistoryQueryExecutor implements method in HistoryDB interface
func (historyDB *historyDB) NewHistoryQueryExecutor(blockStore blkstorage.BlockStore) (ledger.HistoryQueryExecutor, error) {
	return &LevelHistoryDBQueryExecutor{historyDB, blockStore}, nil
}

// GetBlockNumFromSavepoint implements method in HistoryDB interface
func (historyDB *historyDB) GetLastSavepoint() (*version.Height, error) {
	versionBytes, err := historyDB.db.Get(savePointKey)
	if err != nil || versionBytes == nil {
		return nil, err
	}
	height, _ := version.NewHeightFromBytes(versionBytes)
	return height, nil
}

// ShouldRecover implements method in interface kvledger.Recoverer
func (historyDB *historyDB) ShouldRecover(lastAvailableBlock uint64) (bool, uint64, error) {
	if !ledgerconfig.IsHistoryDBEnabled() {
		return false, 0, nil
	}
	savepoint, err := historyDB.GetLastSavepoint()
	if err != nil {
		return false, 0, err
	}
	if savepoint == nil {
		return true, 0, nil
	}
	return savepoint.BlockNum != lastAvailableBlock, savepoint.BlockNum + 1, nil
}

// CommitLostBlock implements method in interface kvledger.Recoverer
func (historyDB *historyDB) CommitLostBlock(block *common.Block) error {
	if err := historyDB.Commit(block); err != nil {
		return err
	}
	return nil
}
