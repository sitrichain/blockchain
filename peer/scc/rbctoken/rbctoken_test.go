package rbctoken

import (
	"encoding/json"
	"fmt"
	chainmock "github.com/rongzer/blockchain/peer/chain/mock"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func setupTestLedger(chainid string, path string) (*shim.MockStub, error) {
	viper.Set("peer.fileSystemPath", path)
	chainmock.MockInitialize()
	chainmock.MockCreateChain(chainid)

	rt := new(RBCToken)
	stub := shim.NewMockStub("rbctoken", rt)
	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		return nil, fmt.Errorf("Init failed for test ledger [%s] with message: %s", chainid, string(res.Message))
	}
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

func TestRBCTokenQueryStateRList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	rlistName1 := "__RBCTOKEN_"
	map1 := map[string]string{"cpno":"0"}
	map1Bytes, err := json.Marshal(map1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(QueryStateRList),  []byte(rlistName1), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "QueryStateRList failed with err: %s", res.Message)

	rlistName2 := "__RBCTOKEN_xxx"
	map2 := map[string]string{"cpno":"0"}
	map2Bytes, err := json.Marshal(map2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(QueryStateRList),  []byte(rlistName2), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateRList should have failed because rlist not exist")

	args = [][]byte{[]byte(chainid), []byte(QueryStateRList),  nil, nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "QueryStateRList should have failed because the the rilist is nil")

}

func TestRBCTokenSetContract(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	contract1 := &ContractEntity{}
	contract1Bytes, err := json.Marshal(contract1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(SetContract),  contract1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetContract failed with err: %s", res.Message)

	contract2 := &ContractEntity{}
	contract2Bytes, err := json.Marshal(contract2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(SetContract),  contract2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(SetContract),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because the the contract is nil")

}

func TestRBCTokenGetContract(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	contractName1 := "contractName1"
	args := [][]byte{[]byte(chainid), []byte(GetContract),  []byte(contractName1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetContract failed with err: %s", res.Message)

	contractName2 := "contractName2"
	args = [][]byte{[]byte(chainid), []byte(GetContract),  []byte(contractName2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetContract should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(GetContract),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetContract should have failed because the the contract is nil")

}

func TestRBCTokenGetBalance(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	contractName1 := "contractName1"
	args := [][]byte{[]byte(chainid), []byte(GetBalance),  []byte(contractName1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetContract failed with err: %s", res.Message)

	contractName2 := "contractName2"
	args = [][]byte{[]byte(chainid), []byte(GetBalance),  []byte(contractName2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(GetBalance),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because the the contract is nil")

}

func TestRBCTokenContractTransfer(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	contractName1 := "contractName1"
	args := [][]byte{[]byte(chainid), []byte(GetBalance),  []byte(contractName1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "SetContract failed with err: %s", res.Message)

	contractName2 := "contractName2"
	args = [][]byte{[]byte(chainid), []byte(GetBalance),  []byte(contractName2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(GetBalance),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "SetContract should have failed because the the contract is nil")

}

func TestRBCTokenGetTokenList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"cpno":"0"}
	map1Bytes, err := json.Marshal(map1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(GetTokenList),  map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetTokenList failed with err: %s", res.Message)

	map2 := map[string]string{"cpno":"1"}
	map2Bytes, err := json.Marshal(map2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{ []byte(chainid), []byte(GetTokenList), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTokenList should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(GetTokenList),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetTokenList should have failed because the the contract is nil")

}

func TestRBCTokenGetDealList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	customerId1 := "customerId1"
	dealType1 := "dealType1"
	map1 := map[string]string{"cpno":"0"}
	map1Bytes, err := json.Marshal(map1)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args := [][]byte{[]byte(chainid), []byte(GetDealList),  []byte(customerId1), []byte(dealType1), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetDealList failed with err: %s", res.Message)

	customerId2 := "customerId2"
	dealType2 := "dealType2"
	map2 := map[string]string{"cpno":"0"}
	map2Bytes, err := json.Marshal(map2)
	if err != nil {
		t.Fatalf(err.Error())
	}
	args = [][]byte{[]byte(chainid), []byte(GetDealList),  []byte(customerId2), []byte(dealType2), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetDealList should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(GetDealList),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetDealList should have failed because the the contract is nil")

}

func TestRBCTokenIncreaseSupply(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	tokenName1 := "tokenName1"
	args := [][]byte{[]byte(chainid), []byte(IncreaseSupply),  []byte(tokenName1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "IncreaseSupply failed with err: %s", res.Message)

	tokenName2 := "tokenName2"
	args = [][]byte{[]byte(chainid), []byte(IncreaseSupply),  []byte(tokenName2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "IncreaseSupply should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(IncreaseSupply),  nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "IncreaseSupply should have failed because the the contract is nil")
}

func TestRBCTokenMining(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	tokenName1 := "tokenName1"
	customerId1 := "customerId1"
	value1 := "10"
	args := [][]byte{[]byte(chainid), []byte(Mining),  []byte(tokenName1), []byte(customerId1), []byte(value1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "Mining failed with err: %s", res.Message)

	tokenName2 := "tokenName2"
	customerId2 := "customerId2"
	value2 := "10"
	args = [][]byte{[]byte(chainid), []byte(Mining),  []byte(tokenName2), []byte(customerId2), []byte(value2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "Mining should have failed because contract is existed")

	args = [][]byte{[]byte(chainid), []byte(Mining),  nil, nil, nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "Mining should have failed because the the contract is nil")

}
