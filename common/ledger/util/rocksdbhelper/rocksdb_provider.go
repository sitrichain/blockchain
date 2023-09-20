/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 SPDX-License-Identifier: Apache-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package rocksdbhelper

import (
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/ledger/util"

	"github.com/tecbot/gorocksdb"
)

var dbNameKeySep = []byte{0x00}
var lastKeyIndicator = byte(0x01)
var dbProvides = map[string]*Provider{}

type Provider struct {
	db        *DB
	dbHandles sync.Map
}

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

func (p *Provider) GetDBHandle(dbName string) util.DBHandle {
	dbHandle, _ := p.dbHandles.LoadOrStore(dbName, &DBHandle{dbName, p.db})
	return dbHandle.(*DBHandle)
}

func (p *Provider) Close() error {
	if err := p.db.Close(); err != nil {
		return err
	}
	return nil
}

type DBHandle struct {
	dbName string
	db     *DB
}

// Get value from given key
func (h *DBHandle) Get(key []byte) ([]byte, error) {
	val, err := h.db.Get(constructRocksKey(h.dbName, key))
	return val, err
}

// Put saves the key/value
func (h *DBHandle) Put(key []byte, value []byte, _ bool) error {
	return h.db.Put(constructRocksKey(h.dbName, key), value, false)
}

// Delete the given key
func (h *DBHandle) Delete(key []byte, _ bool) error {
	return h.db.Delete(constructRocksKey(h.dbName, key), false)
}

func (h *DBHandle) WriteBatch(batch *util.UpdateBatch, _ bool) error {
	//rocksBatch := blockchain_rocksdbtool.NewWriteBatch()
	rocksBatch := gorocksdb.NewWriteBatch()
	for k, v := range batch.GetKVS() {
		key := constructRocksKey(h.dbName, []byte(k))
		if v == nil {
			rocksBatch.Delete(key)
		} else {
			rocksBatch.Put(key, v)
		}
	}
	if err := h.db.WriteBatch(rocksBatch, false); err != nil {
		return err
	}
	return nil
}

func (h *DBHandle) GetIterator(startKey []byte, endKey []byte) util.Iterator {
	sKey := constructRocksKey(h.dbName, startKey)
	eKey := constructRocksKey(h.dbName, endKey)
	if endKey == nil {
		// replace the last byte 'dbNameKeySep' by 'lastKeyIndicator'
		eKey[len(eKey)-1] = lastKeyIndicator
	}
	//logger.Infof("Getting iterator for range [%#v] - [%#v]", string(sKey), string(eKey))
	iterator := h.db.GetIterator(sKey, eKey)

	return iterator
}

func constructRocksKey(dbName string, key []byte) []byte {
	if strings.Index(string(key), "__RBC_MODEL_PUB_") >= 0 {
		//特殊处理主体公钥进rbcchannel通道
		dbName = "rbcchannel"
	}

	b := append(append([]byte(dbName), dbNameKeySep...), key...)
	return b
}
