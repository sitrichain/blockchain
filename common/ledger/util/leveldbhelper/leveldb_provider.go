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
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/log"
	"github.com/syndtr/goleveldb/leveldb"
)

var dbNameKeySep = []byte{0x00}
var lastKeyIndicator = byte(0x01)
var dbProvides = map[string]*Provider{}

// Provider enables to use a single leveldb as multiple logical leveldbs
type Provider struct {
	db *DB
	//dbHandles map[string]*DBHandle
	//mux       sync.Mutex
	dbHandles sync.Map
}

// NewProvider constructs a Provider
func NewProvider(conf *Conf) *Provider {
	db := CreateDB(conf)
	db.Open()
	return &Provider{db: db}
}

func GetProvider(dbpath string) *Provider {
	if dbProvides == nil || dbProvides[dbpath] == nil {
		conf := &Conf{DBPath: dbpath}
		np := NewProvider(conf)
		dbProvides[dbpath] = np
	}
	return dbProvides[dbpath]
}

// GetDBHandle returns a handle to a named db
func (p *Provider) GetDBHandle(dbName string) util.DBHandle {
	dbHandle, _ := p.dbHandles.LoadOrStore(dbName, &DBHandle{dbName, p.db})
	return dbHandle.(*DBHandle)
}

// Close closes the underlying leveldb
func (p *Provider) Close() error {
	if err := p.db.Close(); err != nil {
		return err
	}
	return nil
}

// DBHandle is an handle to a named db
type DBHandle struct {
	dbName string
	db     *DB
}

// Get returns the value for the given key
func (h *DBHandle) Get(key []byte) ([]byte, error) {
	return h.db.Get(constructLevelKey(h.dbName, key))
}

// Put saves the key/value
func (h *DBHandle) Put(key []byte, value []byte, sync bool) error {
	return h.db.Put(constructLevelKey(h.dbName, key), value, sync)
}

// Delete deletes the given key
func (h *DBHandle) Delete(key []byte, sync bool) error {
	return h.db.Delete(constructLevelKey(h.dbName, key), sync)
}

// WriteBatch writes a batch in an atomic way
func (h *DBHandle) WriteBatch(batch *util.UpdateBatch, sync bool) error {
	levelBatch := &leveldb.Batch{}
	for k, v := range batch.GetKVS() {
		key := constructLevelKey(h.dbName, []byte(k))
		if v == nil {
			levelBatch.Delete(key)
		} else {
			levelBatch.Put(key, v)
		}
	}
	if err := h.db.WriteBatch(levelBatch, sync); err != nil {
		return err
	}
	return nil
}

// GetIterator gets an handle to iterator. The iterator should be released after the use.
// The resultset contains all the keys that are present in the db between the startKey (inclusive) and the endKey (exclusive).
// A nil startKey represents the first available key and a nil endKey represent a logical key after the last available key
func (h *DBHandle) GetIterator(startKey []byte, endKey []byte) util.Iterator {
	sKey := constructLevelKey(h.dbName, startKey)
	eKey := constructLevelKey(h.dbName, endKey)
	if endKey == nil {
		// replace the last byte 'dbNameKeySep' by 'lastKeyIndicator'
		eKey[len(eKey)-1] = lastKeyIndicator
	}
	log.Logger.Debugf("Getting iterator for range [%#v] - [%#v]", sKey, eKey)
	iterator := h.db.GetIterator(sKey, eKey)
	return iterator
}

func constructLevelKey(dbName string, key []byte) []byte {
	if strings.Index(string(key), "__RBC_MODEL_PUB_") >= 0 {
		//特殊处理主体公钥进rbcchannel通道
		dbName = "rbcchannel"
	}

	b := append(append([]byte(dbName), dbNameKeySep...), key...)
	return b
}
