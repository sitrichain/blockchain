package rocksdbhelper

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/ledger/util"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/metadata"
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

func setRocksdbOption(opts *gr.Options) {
	opts.SetCreateIfMissing(conf.V.Ledger.RocksDB.CreateIfMissing)
	opts.SetMaxBackgroundCompactions(conf.V.Ledger.RocksDB.MaxBackgroundCompactions)
	opts.SetMaxBackgroundFlushes(conf.V.Ledger.RocksDB.MaxBackgroundFlushes)
	opts.SetBytesPerSync(uint64(conf.V.Ledger.RocksDB.BytesPerSync))
	opts.SetRateLimiter(gr.NewRateLimiter(conf.V.Ledger.RocksDB.BytesPerSecond, 100000, 10))
	opts.SetStatsDumpPeriodSec(uint(conf.V.Ledger.RocksDB.StatsDumpPeriodSec))
	opts.SetMaxOpenFiles(conf.V.Ledger.RocksDB.MaxOpenFiles)
	opts.SetLevelCompactionDynamicLevelBytes(conf.V.Ledger.RocksDB.LevelCompactionDynamicLevelBytes)
	opts.SetNumLevels(conf.V.Ledger.RocksDB.NumLevels)
	opts.SetMaxBytesForLevelBase(uint64(conf.V.Ledger.RocksDB.MaxBytesForLevelBase))
	opts.SetMaxBytesForLevelMultiplier(conf.V.Ledger.RocksDB.MaxBytesForLevelMultiplier)
	opts.SetMaxWriteBufferNumber(conf.V.Ledger.RocksDB.MaxWriteBufferNumber)
	opts.SetLevel0StopWritesTrigger(conf.V.Ledger.RocksDB.Level0StopWritesTrigger)
	opts.SetHardPendingCompactionBytesLimit(uint64(conf.V.Ledger.RocksDB.HardPendingCompactionBytesLimit))
	opts.SetLevel0SlowdownWritesTrigger(conf.V.Ledger.RocksDB.Level0SlowdownWritesTrigger)
	opts.SetTargetFileSizeBase(uint64(conf.V.Ledger.RocksDB.TargetFileSizeBase))
	opts.SetTargetFileSizeMultiplier(conf.V.Ledger.RocksDB.TargetFileSizeMultiplier)
	opts.SetWriteBufferSize(conf.V.Ledger.RocksDB.WriteBufferSize)
	opts.SetMinWriteBufferNumberToMerge(conf.V.Ledger.RocksDB.MinWriteBufferNumberToMerge)
	opts.SetLevel0FileNumCompactionTrigger(conf.V.Ledger.RocksDB.Level0FileNumCompactionTrigger)
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
	setRocksdbOption(opts)
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
