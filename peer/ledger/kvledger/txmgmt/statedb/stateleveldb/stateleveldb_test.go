package stateleveldb

import (
	"os"
	"testing"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/testhelper"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
)

// TestGetStateMultipleKeys tests read for given multiple keys
func TestGetStateMultipleKeys(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/stateleveldb"
	conf.V.FileSystemPath = testPath

	dbProvider := NewVersionedDBProvider()
	defer func() {
		dbProvider.Close()
		os.RemoveAll(testPath)
	}()
	db, err := dbProvider.GetDBHandle("testgetmultiplekeys")
	testhelper.AssertNoError(t, err, "")

	// Test that savepoint is nil for a new state db
	sp, err := db.GetLatestSavePoint()
	testhelper.AssertNoError(t, err, "Error upon GetLatestSavePoint()")
	testhelper.AssertNil(t, sp)

	batch := statedb.NewUpdateBatch()
	expectedValues := make([]*statedb.VersionedValue, 2)
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	expectedValues[0] = &vv1
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	expectedValues[1] = &vv2
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte{}, Version: version.NewHeight(1, 4)}
	batch.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch.Put("ns1", "key2", vv2.Value, vv2.Version)
	batch.Put("ns2", "key3", vv3.Value, vv3.Version)
	batch.Put("ns2", "key4", vv4.Value, vv4.Version)
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)

	actualValues, _ := db.GetStateMultipleKeys("ns1", []string{"key1", "key2"})
	testhelper.AssertEquals(t, actualValues, expectedValues)
}

// TestBasicRW tests basic read-write
func TestBasicRW(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/stateleveldb1"
	conf.V.FileSystemPath = testPath

	dbProvider := NewVersionedDBProvider()
	defer func() {
		dbProvider.Close()
		os.RemoveAll(testPath)
	}()
	db, err := dbProvider.GetDBHandle("testbasicrw")
	testhelper.AssertNoError(t, err, "")

	// Test that savepoint is nil for a new state db
	sp, err := db.GetLatestSavePoint()
	testhelper.AssertNoError(t, err, "Error upon GetLatestSavePoint()")
	testhelper.AssertNil(t, sp)

	val, err := db.GetState("ns", "key1")
	testhelper.AssertNoError(t, err, "Should receive nil rather than error upon reading non existent key")
	testhelper.AssertNil(t, val)

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value3"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte{}, Version: version.NewHeight(1, 4)}
	batch.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch.Put("ns1", "key2", vv2.Value, vv2.Version)
	batch.Put("ns2", "key3", vv3.Value, vv3.Version)
	batch.Put("ns2", "key4", vv4.Value, vv4.Version)
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)

	vv, _ := db.GetState("ns1", "key1")
	testhelper.AssertEquals(t, vv, &vv1)

	vv, _ = db.GetState("ns2", "key4")
	testhelper.AssertEquals(t, vv, &vv4)

	sp, err = db.GetLatestSavePoint()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, sp, savePoint)
}

// TestMultiDBBasicRW tests basic read-write on multiple dbs
func TestMultiDBBasicRW(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/stateleveldb2"
	conf.V.FileSystemPath = testPath

	dbProvider := NewVersionedDBProvider()
	defer func() {
		dbProvider.Close()
		os.RemoveAll(testPath)
	}()

	db1, err := dbProvider.GetDBHandle("testmultidbbasicrw")
	testhelper.AssertNoError(t, err, "")

	db2, err := dbProvider.GetDBHandle("testmultidbbasicrw2")
	testhelper.AssertNoError(t, err, "")

	batch1 := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1_db1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2_db1"), Version: version.NewHeight(1, 2)}
	batch1.Put("ns1", "key1", vv1.Value, vv1.Version)
	batch1.Put("ns1", "key2", vv2.Value, vv2.Version)
	savePoint1 := version.NewHeight(1, 2)
	db1.ApplyUpdates(batch1, savePoint1)

	batch2 := statedb.NewUpdateBatch()
	vv3 := statedb.VersionedValue{Value: []byte("value1_db2"), Version: version.NewHeight(1, 4)}
	vv4 := statedb.VersionedValue{Value: []byte("value2_db2"), Version: version.NewHeight(1, 5)}
	batch2.Put("ns1", "key1", vv3.Value, vv3.Version)
	batch2.Put("ns1", "key2", vv4.Value, vv4.Version)
	savePoint2 := version.NewHeight(1, 5)
	db2.ApplyUpdates(batch2, savePoint2)

	vv, _ := db1.GetState("ns1", "key1")
	testhelper.AssertEquals(t, vv, &vv1)

	sp, err := db1.GetLatestSavePoint()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, sp, savePoint1)

	vv, _ = db2.GetState("ns1", "key1")
	testhelper.AssertEquals(t, vv, &vv3)

	sp, err = db2.GetLatestSavePoint()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, sp, savePoint2)
}

// TestDeletes tests deteles
func TestDeletes(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/stateleveldb3"
	conf.V.FileSystemPath = testPath

	dbProvider := NewVersionedDBProvider()
	defer func() {
		dbProvider.Close()
		os.RemoveAll(testPath)
	}()

	db, err := dbProvider.GetDBHandle("testdeletes")
	testhelper.AssertNoError(t, err, "")

	batch := statedb.NewUpdateBatch()
	vv1 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 1)}
	vv2 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 2)}
	vv3 := statedb.VersionedValue{Value: []byte("value1"), Version: version.NewHeight(1, 3)}
	vv4 := statedb.VersionedValue{Value: []byte("value2"), Version: version.NewHeight(1, 4)}

	batch.Put("ns", "key1", vv1.Value, vv1.Version)
	batch.Put("ns", "key2", vv2.Value, vv2.Version)
	batch.Put("ns", "key3", vv2.Value, vv3.Version)
	batch.Put("ns", "key4", vv2.Value, vv4.Version)
	batch.Delete("ns", "key3", version.NewHeight(1, 5))
	savePoint := version.NewHeight(1, 5)
	err = db.ApplyUpdates(batch, savePoint)
	testhelper.AssertNoError(t, err, "")
	vv, _ := db.GetState("ns", "key2")
	testhelper.AssertEquals(t, vv, &vv2)

	vv, err = db.GetState("ns", "key3")
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertNil(t, vv)

	batch = statedb.NewUpdateBatch()
	batch.Delete("ns", "key2", version.NewHeight(1, 6))
	err = db.ApplyUpdates(batch, savePoint)
	testhelper.AssertNoError(t, err, "")
	vv, err = db.GetState("ns", "key2")
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertNil(t, vv)
}

// TestIterator tests the iterator
func TestIterator(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/stateleveldb4"
	conf.V.FileSystemPath = testPath

	dbProvider := NewVersionedDBProvider()
	defer func() {
		dbProvider.Close()
		os.RemoveAll(testPath)
	}()

	db, err := dbProvider.GetDBHandle("testiterator")
	testhelper.AssertNoError(t, err, "")
	db.Open()
	defer db.Close()
	batch := statedb.NewUpdateBatch()
	batch.Put("ns1", "key1", []byte("value1"), version.NewHeight(1, 1))
	batch.Put("ns1", "key2", []byte("value2"), version.NewHeight(1, 2))
	batch.Put("ns1", "key3", []byte("value3"), version.NewHeight(1, 3))
	batch.Put("ns1", "key4", []byte("value4"), version.NewHeight(1, 4))
	batch.Put("ns2", "key5", []byte("value5"), version.NewHeight(1, 5))
	batch.Put("ns2", "key6", []byte("value6"), version.NewHeight(1, 6))
	batch.Put("ns3", "key7", []byte("value7"), version.NewHeight(1, 7))
	savePoint := version.NewHeight(2, 5)
	db.ApplyUpdates(batch, savePoint)

	itr1, _ := db.GetStateRangeScanIterator("ns1", "key1", "")
	testItr(t, itr1, []string{"ns1\x00key1", "ns1\x00key2", "ns1\x00key3", "ns1\x00key4"})

	itr2, _ := db.GetStateRangeScanIterator("ns1", "key2", "key3")
	testItr(t, itr2, []string{"ns1\x00key2"})

	itr3, _ := db.GetStateRangeScanIterator("ns1", "", "")
	testItr(t, itr3, []string{"ns1\x00key1", "ns1\x00key2", "ns1\x00key3", "ns1\x00key4"})

	itr4, _ := db.GetStateRangeScanIterator("ns2", "", "")
	testItr(t, itr4, []string{"ns2\x00key5", "ns2\x00key6"})
}

func testItr(t *testing.T, itr statedb.ResultsIterator, expectedKeys []string) {
	defer itr.Close()
	for _, expectedKey := range expectedKeys {
		queryResult, _ := itr.Next()
		vkv := queryResult.(*statedb.VersionedKV)
		key := vkv.Key
		testhelper.AssertEquals(t, key, expectedKey)
	}
	last, err := itr.Next()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertNil(t, last)
}
