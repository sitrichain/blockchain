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

package stateleveldb

import (
	"bytes"
	"errors"
	"strings"

	"github.com/rongzer/blockchain/common/ledger/util"
	lh "github.com/rongzer/blockchain/common/ledger/util/leveldbhelper"
	rh "github.com/rongzer/blockchain/common/ledger/util/rocksdbhelper"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/spf13/viper"
)

var compositeKeySep = []byte{0x00}
var lastKeyIndicator = byte(0x01)
var savePointKey = []byte{0x00}
var readNum = 0
var writeNum = 0

//VersionedDBProvider implements interface VersionedDBProvider
type VersionedDBProvider struct {
	dbProvider util.Provider
	// RList的索引信息
	indexDBProvider util.Provider
	// RBC SYS系统合约的数据信息
	rbcDBProvider util.Provider
}

// NewVersionedDBProvider instantiates VersionedDBProvider
func NewVersionedDBProvider() *VersionedDBProvider {
	var dbPath, indexPath, rbcPath string
	var dbProvider, indexDBProvider, rbcDBProvider util.Provider
	dbtype := viper.GetString("ledger.state.stateDatabase")
	if dbtype == "rocksdb" {
		dbPath = ledgerconfig.GetStateRocksDBPath()
		log.Logger.Debugf("constructing VersionedDBProvider dbPath=%s", dbPath)
		indexPath = ledgerconfig.GetIndexRocksDBPath()
		rbcPath = ledgerconfig.GetRbcRocksDBPath()
		dbProvider = rh.GetProvider(dbPath)
		indexDBProvider = rh.GetProvider(indexPath)
		rbcDBProvider = rh.GetProvider(rbcPath)
	} else {
		dbPath = ledgerconfig.GetStateLevelDBPath()
		log.Logger.Debugf("constructing VersionedDBProvider dbPath=%s", dbPath)
		indexPath = ledgerconfig.GetIndexLevelDBPath()
		rbcPath = ledgerconfig.GetRbcLevelDBPath()
		dbProvider = lh.GetProvider(dbPath)
		indexDBProvider = lh.GetProvider(indexPath)
		rbcDBProvider = lh.GetProvider(rbcPath)
	}

	return &VersionedDBProvider{dbProvider, indexDBProvider, rbcDBProvider}
}

// GetDBHandle gets the handle to a named database
func (provider *VersionedDBProvider) GetDBHandle(dbName string) (statedb.VersionedDB, error) {
	dbhandle := provider.dbProvider.GetDBHandle(dbName)
	indexdbhandle := provider.indexDBProvider.GetDBHandle(dbName)
	rbcdbhandle := provider.rbcDBProvider.GetDBHandle(dbName)
	return newVersionedDB(dbhandle, indexdbhandle, rbcdbhandle, dbName), nil
}

// Close closes the underlying db
func (provider *VersionedDBProvider) Close() {
	provider.dbProvider.Close()
	provider.indexDBProvider.Close()
	provider.rbcDBProvider.Close()
}

// VersionedDB implements VersionedDB interface
type versionedDB struct {
	db      util.DBHandle
	indexDB util.DBHandle
	rbcDB   util.DBHandle
	dbName  string
}

// newVersionedDB constructs an instance of VersionedDB
func newVersionedDB(db util.DBHandle, indexDB util.DBHandle, rbcDB util.DBHandle, dbName string) *versionedDB {
	return &versionedDB{db, indexDB, rbcDB, dbName}
}

// Open implements method in VersionedDB interface
func (vdb *versionedDB) Open() error {
	// do nothing because shared db is used
	return nil
}

// Close implements method in VersionedDB interface
func (vdb *versionedDB) Close() {
	// do nothing because shared db is used
}

// ValidateKey implements method in VersionedDB interface
func (vdb *versionedDB) ValidateKey(_ string) error {
	return nil
}

//获取读写次数
func (vdb *versionedDB) GetReadWriteNum() (int, int) {
	return readNum, writeNum
}

// GetState implements method in VersionedDB interface
func (vdb *versionedDB) GetState(namespace string, key string) (*statedb.VersionedValue, error) {
	readNum++
	if readNum >= 1000000 {
		readNum = 0
	}

	compositeKey := constructCompositeKey(namespace, key)

	db := vdb.db
	if strings.HasPrefix(key, "__RLIST_") {
		db = vdb.indexDB
	} else if strings.HasPrefix(key, "__RBC_") {
		db = vdb.rbcDB
	}

	dbVal, err := db.Get(compositeKey)

	if err != nil {
		return nil, err
	}
	if dbVal == nil {
		return nil, nil
	}
	val, ver := statedb.DecodeValue(dbVal)

	return &statedb.VersionedValue{Value: val, Version: ver}, nil
}

// GetStateMultipleKeys implements method in VersionedDB interface
func (vdb *versionedDB) GetStateMultipleKeys(namespace string, keys []string) ([]*statedb.VersionedValue, error) {
	vals := make([]*statedb.VersionedValue, len(keys))
	for i, key := range keys {
		val, err := vdb.GetState(namespace, key)
		if err != nil {
			return nil, err
		}
		vals[i] = val
	}
	return vals, nil
}

// GetStateRangeScanIterator implements method in VersionedDB interface
// startKey is inclusive
// endKey is exclusive
func (vdb *versionedDB) GetStateRangeScanIterator(namespace string, startKey string, endKey string) (statedb.ResultsIterator, error) {
	compositeStartKey := constructCompositeKey(namespace, startKey)
	compositeEndKey := constructCompositeKey(namespace, endKey)
	if endKey == "" {
		compositeEndKey[len(compositeEndKey)-1] = lastKeyIndicator
	}
	db := vdb.db
	if strings.HasPrefix(startKey, "__RLIST_") {
		db = vdb.indexDB
	} else if strings.HasPrefix(startKey, "__RBC_") {
		db = vdb.rbcDB
	}

	dbItr := db.GetIterator(compositeStartKey, compositeEndKey)
	return newKVScanner(namespace, dbItr), nil
}

// ExecuteQuery implements method in VersionedDB interface
func (vdb *versionedDB) ExecuteQuery(_, _ string) (statedb.ResultsIterator, error) {
	return nil, errors.New("ExecuteQuery not supported for leveldb")
}

// ApplyUpdates implements method in VersionedDB interface
func (vdb *versionedDB) ApplyUpdates(batch *statedb.UpdateBatch, height *version.Height) error {
	dbBatch := util.NewUpdateBatch()
	indexBatch := util.NewUpdateBatch()
	rbcBatch := util.NewUpdateBatch()
	namespaces := batch.GetUpdatedNamespaces()

	for _, ns := range namespaces {
		updates := batch.GetUpdates(ns)
		for k, vv := range updates {
			compositeKey := constructCompositeKey(ns, k)
			theBatch := dbBatch
			if strings.HasPrefix(k, "__RLIST_") {
				theBatch = indexBatch
			} else if strings.HasPrefix(k, "__RBC_") {
				theBatch = rbcBatch
			}
			//logger.Debugf("Channel [%s]: Applying key=[%#v]", vdb.dbName, compositeKey)
			writeNum++
			if writeNum >= 1000000 {
				writeNum = 0
			}
			if vv.Value == nil {
				theBatch.Delete(compositeKey)
			} else {
				theBatch.Put(compositeKey, statedb.EncodeValue(vv.Value, vv.Version))
			}
		}
	}

	if height != nil {
		dbBatch.Put(savePointKey, height.ToBytes())
	}

	if height != nil {
		indexBatch.Put(savePointKey, height.ToBytes())
	}

	if height != nil {
		rbcBatch.Put(savePointKey, height.ToBytes())
	}

	if err := vdb.db.WriteBatch(dbBatch, false); err != nil {
		return err
	}

	if err := vdb.indexDB.WriteBatch(indexBatch, false); err != nil {
		return err
	}

	if err := vdb.rbcDB.WriteBatch(rbcBatch, false); err != nil {
		return err
	}
	return nil
}

// GetLatestSavePoint implements method in VersionedDB interface
func (vdb *versionedDB) GetLatestSavePoint() (*version.Height, error) {
	versionBytes, err := vdb.db.Get(savePointKey)
	if err != nil {
		return nil, err
	}
	if versionBytes == nil {
		return nil, nil
	}
	ver, _ := version.NewHeightFromBytes(versionBytes)
	return ver, nil
}

func constructCompositeKey(ns string, key string) []byte {

	return append(append([]byte(ns), compositeKeySep...), []byte(key)...)
}

func splitCompositeKey(compositeKey []byte) (string, string) {
	split := bytes.SplitN(compositeKey, compositeKeySep, 2)
	return string(split[0]), string(split[1])
}

type kvScanner struct {
	namespace string
	dbItr     util.Iterator
}

func newKVScanner(namespace string, dbItr util.Iterator) *kvScanner {
	return &kvScanner{namespace, dbItr}
}

func (scanner *kvScanner) Next() (statedb.QueryResult, error) {
	if !scanner.dbItr.Next() {
		return nil, nil
	}
	dbKey := scanner.dbItr.Key()
	dbVal := scanner.dbItr.Value()
	// 这里调用Value方法，是为了防止重复拿到迭代器的最后一个元素的Key，因为：当迭代器到达最后一个元素后，再次Next依然会返回true，
	// 这时调用Key方法，依然拿到的是最后一个元素的key，但这时调用Value方法，无法拿到最后一个元素的value，
	// 因此，Value方法可以作为Next方法的补充判定，即：若Value的值为nil，则迭代器"到头"了，不进行后续逻辑。
	if dbVal == nil {
		return nil, nil
	}
	dbValCopy := make([]byte, len(dbVal))
	copy(dbValCopy, dbVal)
	_, key := splitCompositeKey(dbKey)
	value, ver := statedb.DecodeValue(dbValCopy)
	return &statedb.VersionedKV{
		CompositeKey:   statedb.CompositeKey{Namespace: scanner.namespace, Key: key},
		VersionedValue: statedb.VersionedValue{Value: value, Version: ver}}, nil
}

func (scanner *kvScanner) Close() {
	scanner.dbItr.Release()
}
