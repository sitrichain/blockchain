package rbcmodel

import (
	"encoding/json"
	"fmt"
	chainmock "github.com/rongzer/blockchain/peer/chain/mock"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/scc/rbccustomer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func setupTestLedger(chainid string, path string) (*shim.MockStub, error) {
	viper.Set("peer.fileSystemPath", path)
	chainmock.MockInitialize()
	chainmock.MockCreateChain(chainid)

	rm := new(RBCModel)
	stub := shim.NewMockStub("rbcmodel", rm)
	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		return nil, fmt.Errorf("Init failed for test ledger [%s] with message: %s", chainid, string(res.Message))
	}

	rc := new(rbccustomer.RBCCustomer)
	otherStub := shim.NewMockStub("rbccustomer", rc)
	stub.MockPeerChaincode("rbccustomer", otherStub)

	return stub, nil
}

func TestRBCModelQueryStateHistory(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	historyKey1 := "historyKey1"
	args := [][]byte{[]byte(chainid), []byte(QueryStateHistory), []byte(historyKey1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateHistory failed with err: %s", res.Message)

	historyKey2 := "historyKey2"
	args = [][]byte{[]byte(chainid), []byte(QueryStateHistory),  []byte(historyKey2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because historykeys2 not exist")

	args = [][]byte{[]byte(chainid), []byte(QueryStateHistory),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because the the historykey is nil")
}

func TestRBCModelQueryStateRList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	historyKey1 := "historyKey1"
	args := [][]byte{[]byte(chainid), []byte(QueryStateHistory),  []byte(historyKey1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateHistory failed with err: %s", res.Message)

	historyKey2 := "historyKey2"
	args = [][]byte{[]byte(chainid), []byte(QueryStateHistory),  []byte(historyKey2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because historykeys2 not exist")

	args = [][]byte{[]byte(chainid), []byte(QueryStateHistory),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because the the historykey is nil")
}

func TestRBCModelSetModel(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := new(RBCModelInfo)
	model1Bytes, _ := json.Marshal(model1)
	args := [][]byte{[]byte(chainid), []byte(SetModel),  model1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetModel failed with err: %s", res.Message)

	model2 := new(RBCModelInfo)
	model2Bytes, _ := json.Marshal(model2)
	args = [][]byte{[]byte(chainid), []byte(SetModel),  []byte(model2Bytes)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetModel should have failed because historykeys2 not exist")

	args = [][]byte{[]byte(chainid), []byte(SetModel),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetModel should have failed because the the historykey is nil")
}

func TestRBCModelGetModel(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	args := [][]byte{[]byte(chainid), []byte(SetModel),  []byte(model1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateHistory failed with err: %s", res.Message)

	model2 := "model2"
	args = [][]byte{[]byte(chainid), []byte(SetModel),  []byte(model2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(SetModel),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because the the modle name is nil")
}

func TestRBCModelDeleteModel(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	args := [][]byte{[]byte(chainid), []byte(DeleteModel),  []byte(model1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateHistory failed with err: %s", res.Message)

	model2 := "model2"
	args = [][]byte{[]byte(chainid), []byte(DeleteModel),  []byte(model2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(DeleteModel),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateHistory should have failed because the the modle name is nil")
}

func TestRBCModelGetModelList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(chainid), []byte(GetModelList)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateHistory failed with err: %s", res.Message)

}

func TestRBCModelSetTable(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	table1 := new(RBCTable)
	table1Bytes, err := json.Marshal(table1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(SetTable), []byte(model1), table1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetTable failed with err: %s", res.Message)

	model2 := "model2"
	table2 := new(RBCTable)
	table2Bytes, err := json.Marshal(table2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(SetTable),  []byte(model2), table2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetTable should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(SetTable),  []byte(model2), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetTable should have failed because the the modle name is nil")

}

func TestRBCModelDeleteTable(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	table1 := "table1"
	args := [][]byte{[]byte(chainid), []byte(DeleteTable), []byte(model1), []byte(table1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "DeleteTable failed with err: %s", res.Message)

	model2 := "model2"
	table2 := "table2"
	args = [][]byte{[]byte(chainid), []byte(DeleteTable),  []byte(model2), []byte(table2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "DeleteTable should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(DeleteTable),  []byte(model2), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "DeleteTable should have failed because the the modle name is nil")

}

func TestRBCModelGetTable(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	table1 := "table1"
	args := [][]byte{[]byte(chainid), []byte(DeleteTable),  []byte(model1), []byte(table1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "DeleteTable failed with err: %s", res.Message)

	model2 := "model2"
	table2 := "table2"
	args = [][]byte{[]byte(chainid), []byte(DeleteTable),  []byte(model2), []byte(table2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "DeleteTable should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(DeleteTable),  []byte(model2), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "DeleteTable should have failed because the the modle name is nil")

}

func TestRBCModelGetTableList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	args := [][]byte{ []byte(chainid), []byte(GetTableList), []byte(model1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetTableList failed with err: %s", res.Message)

	model2 := "model2"
	args = [][]byte{[]byte(chainid), []byte(GetTableList),  []byte(model2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTableList should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetTableList),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTableList should have failed because the the modle name is nil")

}

func TestRBCModelGetTableListWithPageCat(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"modelName":"model1", "cpno":"0"}
	map1Json, err := json.Marshal(map1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{ []byte(chainid), []byte(GetTableListWithPageCat), map1Json}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetTableListWithPageCat failed with err: %s", res.Message)

	map2 := map[string]string{"modelName":"model2", "cpno":"0"}
	map2Json, err := json.Marshal(map2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(GetTableListWithPageCat),  map2Json}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTableListWithPageCat should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetTableListWithPageCat),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTableListWithPageCat should have failed because the the modle name is nil")

}

func TestRBCModelSetMainPublicKey(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	pubKey1 := &PublicKey{ModelName:"model1", PublicKey: "", MainId:""}
	pubKey1Bytes, err := json.Marshal(pubKey1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(SetMainPublicKey),  pubKey1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetMainPublicKey failed with err: %s", res.Message)

	pubKey2 := &PublicKey{ModelName:"model2", PublicKey: "", MainId:""}
	pubKey2Bytes, err := json.Marshal(pubKey2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(SetMainPublicKey),  pubKey2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetMainPublicKey should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(SetMainPublicKey),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetMainPublicKey should have failed because the the modle name is nil")

}

func TestRBCModelGetMainPublicKey(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	model1 := "model1"
	vals := map[string]Cryto{}
	valsBytes, err := json.Marshal(vals)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(GetMainPublicKey),  []byte(model1), valsBytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetMainPublicKey failed with err: %s", res.Message)

	model2 := "model2"
	vals2 := map[string]Cryto{}
	vals2Bytes, err := json.Marshal(vals2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(GetMainPublicKey),  []byte(model2), vals2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetMainPublicKey should have failed because model2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetMainPublicKey),  []byte(model1), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetMainPublicKey should have failed because the the params is nil")

}

func TestRBCModelGetMainCryptogram(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	modelName1 := "model1"
	tableName1 := "table1"
	pubG1 := ""
	reqInfo1 := ""
	args := [][]byte{[]byte(chainid), []byte(GetMainCryptogram),  []byte(modelName1), []byte(tableName1), []byte(pubG1), []byte(reqInfo1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetMainPublicKey failed with err: %s", res.Message)

	modelName2 := "model2"
	tableName2 := "table2"
	pubG2 := ""
	reqInfo2 := ""
	args = [][]byte{[]byte(chainid), []byte(GetMainCryptogram),  []byte(modelName2), []byte(tableName2), []byte(pubG2), []byte(reqInfo2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetMainPublicKey should have failed because modelName2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetMainCryptogram),  []byte(modelName1), nil, nil, nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetMainPublicKey should have failed because the the params is nil")

}

func TestRBCModelGetFromHttp(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	url1 := "192.168.1.239:8080"
	map1 := map[string]string{"abc":"123"}
	map1Bytes, err := json.Marshal(map1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(GetFromHttp),  []byte(url1), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetFromHttp failed with err: %s", res.Message)

	url2 := "192.168.1.239:8090"
	map2 := map[string]string{"abc":"123"}
	map2Bytes, err := json.Marshal(map2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(GetFromHttp),  []byte(url2), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetFromHttp should have failed because url2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetFromHttp),  []byte(url2), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetFromHttp should have failed because the the params is nil")

}

func TestRBCModelGetAttach(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	attachKey1 := "attachKey"
	args := [][]byte{[]byte(chainid), []byte(GetAttach),  []byte(attachKey1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetAttach failed with err: %s", res.Message)

	attachKey2 := "attachKey"
	args = [][]byte{[]byte(chainid), []byte(GetAttach),  []byte(attachKey2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetAttach should have failed because url2 not exist")

	args = [][]byte{[]byte(chainid), []byte(GetAttach),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetAttach should have failed because the the params is nil")
}
