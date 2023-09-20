package rocksdbhelper

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tecbot/gorocksdb"
)

// 测试对DB的基本操作：GET/PUT/DELETE/OPEN/CLOSE
func TestDB(t *testing.T) {
	conf := &Conf{DBPath: "/tmp/rongzer/test/rocksdbhelper"}
	db := CreateDB(conf)
	err := db.Open()
	if err != nil {
		t.Fatal("cannot open db")
	}
	defer db.Close()
	err = db.Put([]byte("test_key"), []byte("test_value"), true)
	if err != nil {
		t.Fatal("cannot put into db")
	}
	v, err := db.Get([]byte("test_key"))
	if err != nil {
		t.Fatal("cannot get from db")
	}
	t.Logf("value of test_key is %v", string(v))
	err = db.Delete([]byte("test_key"), true)
	if err != nil {
		t.Fatal("cannot delete from db")
	}
}

func TestDB_ReadNotExist(t *testing.T) {
	conf := &Conf{DBPath: "/tmp/rongzer/test/rocksdbhelper"}
	db := CreateDB(conf)
	err := db.Open()
	if err != nil {
		t.Fatal("cannot open db")
	}
	defer db.Close()
	value, err := db.Get([]byte("this_is_a_not_exist_pair"))
	assert.NoError(t, err)
	assert.Nil(t, value)
}

// 测试db的批量写入
func TestDB_WriteBatch(t *testing.T) {
	conf := &Conf{DBPath: "/tmp/rongzer/test/rocksdbhelper"}
	db := CreateDB(conf)
	err := db.Open()
	if err != nil {
		t.Fatal("cannot open db")
	}
	defer db.Close()
	batch := gorocksdb.NewWriteBatch()
	for i := 1; i < 10; i++ {
		key := []byte(strconv.Itoa(i))
		value := key
		batch.Put(key, value)
	}
	err = db.WriteBatch(batch, true)
	if err != nil {
		t.Fatal("cannot WriteBatch to db")
	}
}

// 测试db的迭代器功能
func TestDB_GetIterator(t *testing.T) {
	conf := &Conf{DBPath: "/tmp/rongzer/test/rocksdbhelper"}
	db := CreateDB(conf)
	err := db.Open()
	if err != nil {
		t.Fatal("cannot open db")
	}
	defer func() {
		db.Close()
		os.RemoveAll("/tmp/rongzer/test/rocksdbhelper")
	}()
	startKey := []byte("1")
	iter := db.GetIterator(startKey, nil)
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		t.Logf("this key is %s", iter.Key())
		t.Logf("this value is %s", iter.Value())
	}
	iter.Release()
}
