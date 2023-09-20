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

package leveldbhelper

import (
	"bytes"
	"errors"
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	goleveldbutil "github.com/syndtr/goleveldb/leveldb/util"
)

type dbState int32

const (
	closed dbState = iota
	opened
)

// Conf configuration for `DB`
type Conf struct {
	DBPath string
}

// DB - a wrapper on an actual store
type DB struct {
	conf    *Conf
	db      *leveldb.DB
	dbState dbState
	sync.Mutex

	readOpts        *opt.ReadOptions
	writeOptsNoSync *opt.WriteOptions
	writeOptsSync   *opt.WriteOptions
}

// CreateDB constructs a `DB`
func CreateDB(conf *Conf) *DB {
	readOpts := &opt.ReadOptions{}
	writeOptsNoSync := &opt.WriteOptions{}
	writeOptsSync := &opt.WriteOptions{}
	writeOptsSync.Sync = true

	return &DB{
		conf:            conf,
		dbState:         closed,
		readOpts:        readOpts,
		writeOptsNoSync: writeOptsNoSync,
		writeOptsSync:   writeOptsSync}
}

// Open opens the underlying db
func (dbInst *DB) Open() error {
	dbInst.Lock()
	defer dbInst.Unlock()
	if dbInst.dbState == opened {
		return errors.New("db is opened")
	}
	//dbOpts := &opt.Options{}
	// DB参数
	// 缓冲大小
	handles := 32
	cache := 512
	log.Logger.Infof(dbInst.conf.DBPath)
	if strings.HasSuffix(dbInst.conf.DBPath, "indexLeveldb") {
		log.Logger.Infof("indexLeveldb %s", dbInst.conf.DBPath)

		cache = 512 * 2
	}

	dbOpts := &opt.Options{
		Compression:            opt.SnappyCompression,
		OpenFilesCacheCapacity: handles,
		CompactionTableSize:    32 * opt.MiB,
		BlockCacheCapacity:     cache * opt.MiB,
		WriteBuffer:            256 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	}

	if strings.HasSuffix(dbInst.conf.DBPath, "attach") {
		log.Logger.Infof("attach %s", dbInst.conf.DBPath)
		dbOpts = &opt.Options{}
	}

	dbPath := dbInst.conf.DBPath
	var err error
	var dirEmpty bool
	if dirEmpty, err = util.CreateDirIfMissing(dbPath); err != nil {
		//panic(fmt.Sprintf("Error while trying to create dir if missing: %s", err))
		return err
	}
	dbOpts.ErrorIfMissing = !dirEmpty
	if dbInst.db, err = leveldb.OpenFile(dbPath, dbOpts); err != nil {
		//panic(fmt.Sprintf("Error while trying to open DB: %s", err))
		return err
	}
	dbInst.dbState = opened
	return nil
}

// Close closes the underlying db
func (dbInst *DB) Close() error {
	dbInst.Lock()
	defer dbInst.Unlock()
	if dbInst.dbState == closed {
		return errors.New("db is closed")
	}
	if err := dbInst.db.Close(); err != nil {
		log.Logger.Errorf("Error while closing DB: %s", err)
		return err
	}
	dbInst.dbState = closed
	return errors.New("db is closed")
}

// Get returns the value for the given key
func (dbInst *DB) Get(key []byte) ([]byte, error) {
	value, err := dbInst.db.Get(key, dbInst.readOpts)
	if err == leveldb.ErrNotFound {
		value = nil
		err = nil
	}
	if err != nil {
		log.Logger.Errorf("Error while trying to retrieve key [%#v]: %s", key, err)
		return nil, err
	}
	return value, nil
}

// Put saves the key/value
func (dbInst *DB) Put(key []byte, value []byte, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	err := dbInst.db.Put(key, value, wo)
	if err != nil {
		log.Logger.Errorf("Error while trying to write key [%#v]", key)
		return err
	}
	return nil
}

// Delete deletes the given key
func (dbInst *DB) Delete(key []byte, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	err := dbInst.db.Delete(key, wo)
	if err != nil {
		log.Logger.Errorf("Error while trying to delete key [%#v]", key)
		return err
	}
	return nil
}

// GetIterator returns an iterator over key-value store. The iterator should be released after the use.
// The resultset contains all the keys that are present in the db between the startKey (inclusive) and the endKey (exclusive).
// A nil startKey represents the first available key and a nil endKey represent a logical key after the last available key
func (dbInst *DB) GetIterator(startKey []byte, endKey []byte) util.Iterator {
	return &Iterator{dbInst.db.NewIterator(&goleveldbutil.Range{Start: startKey, Limit: endKey}, dbInst.readOpts)}
}

// Iterator extends actual leveldb iterator
type Iterator struct {
	iterator.Iterator
}

func retrieveAppKey(levelKey []byte) []byte {
	s := bytes.SplitN(levelKey, dbNameKeySep, 2)
	if len(s) <= 1 {
		return levelKey
	}
	return s[1]
}

// Key wraps actual leveldb iterator method
func (itr *Iterator) Key() []byte {
	return retrieveAppKey(itr.Iterator.Key())
}

func (itr *Iterator) Next() bool {
	return itr.Iterator.Next()
}

func (itr *Iterator) Value() []byte {
	return itr.Iterator.Value()
}

// Release方法应保证：可以被多次调用而不报错
func (itr *Iterator) Release() {
	itr.Iterator.Release()
}

func (itr *Iterator) SeekToFirst() {
	itr.Iterator.First()
}

// Valid方法配合Next方法使用
func (itr *Iterator) Valid() bool {
	return itr.Iterator.Valid()
}

// WriteBatch writes a batch
func (dbInst *DB) WriteBatch(batch util.DbUpdateBatch, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	theBatch := batch.(*leveldb.Batch)
	if err := dbInst.db.Write(theBatch, wo); err != nil {
		return err
	}
	return nil
}
