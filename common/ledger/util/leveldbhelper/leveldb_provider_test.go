package leveldbhelper

import (
	"github.com/rongzer/blockchain/common/ledger/util"
	"os"
	"testing"
)

// 测试Provider和DBHandle相关方法
func TestProvider(t *testing.T) {
	p := GetProvider("/tmp/rongzer/test/leveldbhelper")
	defer func() {
		p.Close()
		os.RemoveAll("/tmp/rongzer/test/leveldbhelper")
	}()
	dh := p.GetDBHandle("blockChain")
	err := dh.Put([]byte("test_key"), []byte("test_value"), true)
	if err != nil {
		t.Fatal("cannot put into db")
	}
	v, err := dh.Get([]byte("test_key"))
	if err != nil {
		t.Fatal("cannot get from db")
	}
	t.Logf("value of test_key is %v", string(v))
	err = dh.Delete([]byte("test_key"), true)
	if err != nil {
		t.Fatal("cannot delete from db")
	}
	batch := util.NewUpdateBatch()
	batch.Put([]byte("1"), []byte("1"))
	batch.Put([]byte("2"), []byte("2"))
	err = dh.WriteBatch(batch, true)
	if err != nil {
		t.Fatal("cannot WriteBatch to db")
	}
	iter := dh.GetIterator([]byte("1"), nil)
	for iter.SeekToFirst(); iter.Valid(); iter.Next() {
		t.Logf("this key is %s", iter.Key())
		t.Logf("this value is %s", iter.Value())
	}
	iter.Release()
}
