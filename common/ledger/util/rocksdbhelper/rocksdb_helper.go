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
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/metadata"

	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/log"
	"github.com/spf13/viper"
	gr "github.com/tecbot/gorocksdb"
)

type dbState int32

const (
	closed dbState = iota
	opened
)

type Conf struct {
	DBPath string
}

type DB struct {
	conf    *Conf
	db      *gr.DB
	dbState dbState
	sync.Mutex

	readOpts        *gr.ReadOptions
	writeOptsNoSync *gr.WriteOptions
	writeOptsSync   *gr.WriteOptions
}

func CreateDB(conf *Conf) *DB {
	readOpts := gr.NewDefaultReadOptions()
	writeOptsSync := gr.NewDefaultWriteOptions()
	writeOptsNoSync := gr.NewDefaultWriteOptions()
	writeOptsSync.SetSync(true)

	return &DB{
		conf:            conf,
		dbState:         closed,
		readOpts:        readOpts,
		writeOptsSync:   writeOptsSync,
		writeOptsNoSync: writeOptsNoSync,
	}
}

func getViperKeys(m map[string]interface{}) []string {
	// 数组默认长度为map长度,后面append时,不需要重新申请内存和拷贝,效率较高
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func setRocksdbOption(name string, opts *gr.Options) {
	switch name {
	case "createifmissing":
		para := viper.GetBool("ledger.state.rocksDBConfig.createIfMissing")
		opts.SetCreateIfMissing(para)
	case "maxbackgroundcompactions":
		para := viper.GetInt("ledger.state.rocksDBConfig.maxBackgroundCompactions")
		opts.SetMaxBackgroundCompactions(para)
	case "maxbackgroundflushes":
		para := viper.GetInt("ledger.state.rocksDBConfig.maxBackgroundFlushes")
		opts.SetMaxBackgroundFlushes(para)
	case "bytespersync":
		para := viper.GetInt("ledger.state.rocksDBConfig.bytesPerSync")
		opts.SetBytesPerSync(uint64(para))
	case "bytespersecond":
		para := viper.GetInt("ledger.state.rocksDBConfig.bytesPerSecond")
		rateLimiter := gr.NewRateLimiter(int64(para), 100000, 10)
		opts.SetRateLimiter(rateLimiter)
	case "statsdumpperiodsec":
		para := viper.GetInt("ledger.state.rocksDBConfig.statsDumpPeriodSec")
		opts.SetStatsDumpPeriodSec(uint(para))
	case "maxopenfiles":
		para := viper.GetInt("ledger.state.rocksDBConfig.maxOpenFiles")
		opts.SetMaxOpenFiles(para)
	case "levelcompactiondynamiclevelbytes":
		para := viper.GetBool("ledger.state.rocksDBConfig.levelCompactionDynamicLevelBytes")
		opts.SetLevelCompactionDynamicLevelBytes(para)
	case "numlevels":
		para := viper.GetInt("ledger.state.rocksDBConfig.numLevels")
		opts.SetNumLevels(para)
	case "maxbytesforlevelbase":
		para := viper.GetInt("ledger.state.rocksDBConfig.maxBytesForLevelBase")
		opts.SetMaxBytesForLevelBase(uint64(para))
	case "maxbytesforlevelmultiplier":
		para := viper.GetFloat64("ledger.state.rocksDBConfig.maxBytesForLevelMultiplier")
		opts.SetMaxBytesForLevelMultiplier(para)
	case "arenablocksize":
		para := viper.GetInt("ledger.state.rocksDBConfig.arenaBlockSize")
		opts.SetArenaBlockSize(para)
	case "maxwritebuffernumber":
		para := viper.GetInt("ledger.state.rocksDBConfig.maxWriteBufferNumber")
		opts.SetMaxWriteBufferNumber(para)
	case "level0stopwritestrigger":
		para := viper.GetInt("ledger.state.rocksDBConfig.level0StopWritesTrigger")
		opts.SetLevel0StopWritesTrigger(para)
	case "hardpendingcompactionbyteslimit":
		para := viper.GetInt("ledger.state.rocksDBConfig.hardPendingCompactionBytesLimit")
		opts.SetHardPendingCompactionBytesLimit(uint64(para))
	case "level0slowdownwritestrigger":
		para := viper.GetInt("ledger.state.rocksDBConfig.level0SlowdownWritesTrigger")
		opts.SetLevel0SlowdownWritesTrigger(para)
	case "targetfilesizebase":
		para := viper.GetInt("ledger.state.rocksDBConfig.targetFileSizeBase")
		opts.SetTargetFileSizeBase(uint64(para))
	case "targetfilesizemultiplier":
		para := viper.GetInt("ledger.state.rocksDBConfig.targetFileSizeMultiplier")
		opts.SetTargetFileSizeMultiplier(para)
	case "writebuffersize":
		para := viper.GetInt("ledger.state.rocksDBConfig.writeBufferSize")
		opts.SetWriteBufferSize(para)
	case "minwritebuffernumbertomerge":
		para := viper.GetInt("ledger.state.rocksDBConfig.minWriteBufferNumberToMerge")
		opts.SetMinWriteBufferNumberToMerge(para)
	case "level0filenumcompactiontrigger":
		para := viper.GetInt("ledger.state.rocksDBConfig.level0FileNumCompactionTrigger")
		opts.SetLevel0FileNumCompactionTrigger(para)
	default:
		log.Logger.Errorf("此配置项暂不支持: %s", name)
	}
}

func (dbInst *DB) Open() error {
	dbInst.Lock()
	defer dbInst.Unlock()

	if dbInst.dbState == opened {
		return errors.New("db is opened")
	}

	dbPath := dbInst.conf.DBPath

	if _, err := util.CreateDirIfMissing(dbPath); err != nil {
		log.Logger.Errorf("创建rocksdb目录失败: %s", err)
		return err
	}

	rocksDBConfigMap := viper.GetStringMap("ledger.state.rocksDBConfig")
	rocksDBConfigKeys := getViperKeys(rocksDBConfigMap)
	// 读取配置
	opts := gr.NewDefaultOptions()
	bbtOpts := gr.NewDefaultBlockBasedTableOptions()
	fp := gr.NewBloomFilter(10)
	bbtOpts.SetFilterPolicy(fp)
	opts.SetBlockBasedTableFactory(bbtOpts)
	opts.SetCompressionPerLevel([]gr.CompressionType{gr.NoCompression, gr.NoCompression, gr.LZ4Compression, gr.LZ4Compression, gr.LZ4Compression, gr.ZSTDCompression, gr.ZSTDCompression})
	if !strings.Contains(metadata.Version, "release") && !strings.Contains(metadata.Version, "feature") {
		opts.SetCompressionPerLevel([]gr.CompressionType{gr.NoCompression, gr.NoCompression, gr.LZ4Compression, gr.LZ4Compression, gr.LZ4Compression, gr.LZ4Compression, gr.LZ4Compression})
	}
	for _, k := range rocksDBConfigKeys {
		setRocksdbOption(k, opts)
	}
	db, err := gr.OpenDb(opts, dbPath)
	if err != nil {
		log.Logger.Errorf("rocksdb打开数据库失败: %s", err)
		return err
	}

	dbInst.db = db
	dbInst.dbState = opened
	return nil
}

func (dbInst *DB) Close() error {
	dbInst.Lock()
	defer dbInst.Unlock()
	if dbInst.dbState == closed {
		return errors.New("db is closed")
	}
	if f := dbInst.db.Close; f != nil {
		f()
		return nil
	}
	dbInst.dbState = closed
	return errors.New("db is closed")
}

func (dbInst *DB) Get(key []byte) ([]byte, error) {
	value, err := dbInst.db.GetBytes(dbInst.readOpts, key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (dbInst *DB) Put(key []byte, value []byte, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	err := dbInst.db.Put(wo, key, value)
	if err != nil {
		log.Logger.Errorf("Error while trying to write key [%#v]", key)
		return err
	}
	return nil
}

func (dbInst *DB) Delete(key []byte, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	err := dbInst.db.Delete(wo, key)
	if err != nil {
		log.Logger.Errorf("Error while trying to delete key [%#v]", key)
		return err
	}

	return nil
}

func (dbInst *DB) GetIterator(startKey []byte, endKey []byte) util.Iterator {
	dbInst.Lock()
	defer dbInst.Unlock()

	dbInst.readOpts.SetIterateUpperBound(endKey)
	it := dbInst.db.NewIterator(dbInst.readOpts)
	it.Seek(startKey)
	// 确保迭代器的startKey是闭区间
	it.Prev()
	return &Iterator{it}
}

type Iterator struct {
	*gr.Iterator
}

func (it *Iterator) Next() bool {
	if it.Valid() {
		it.Iterator.Next()
		return true
	}
	return false
}

func (it *Iterator) Key() []byte {
	k := it.Iterator.Key()
	return k.Data()
}

func (it *Iterator) Value() []byte {
	if !it.Valid() {
		return nil
	}
	v := it.Iterator.Value()
	return v.Data()
}

// Release方法应保证：可以被多次调用而不报错
func (it *Iterator) Release() {
	if reflect.ValueOf(it.Iterator).Elem().Field(0).IsNil() {
		return
	}
	it.Iterator.Close()
}

func (it *Iterator) SeekToFirst() {
	it.Iterator.SeekToFirst()
}

// Valid方法配合Next方法使用
func (it *Iterator) Valid() bool {
	return it.Iterator.Valid()
}

func (dbInst *DB) WriteBatch(batch util.DbUpdateBatch, sync bool) error {
	wo := dbInst.writeOptsNoSync
	if sync {
		wo = dbInst.writeOptsSync
	}
	theBatch := batch.(*gr.WriteBatch)
	if err := dbInst.db.Write(wo, theBatch); err != nil {
		return err
	}
	theBatch.Destroy()
	return nil
}
