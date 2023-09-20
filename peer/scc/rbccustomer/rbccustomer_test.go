package rbccustomer

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

	lq := new(RBCCustomer)
	stub := shim.NewMockStub("RBCCustomer", lq)
	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		return nil, fmt.Errorf("Init failed for test ledger [%s] with message: %s", chainid, string(res.Message))
	}
	return stub, nil
}

func TestRBCCustomerQueryStateHistory(t *testing.T) {
	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	key1 := "key1"
	args := [][]byte{[]byte(queryStateHistory), []byte(chainid), []byte(key1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	key2 := "key2"
	args = [][]byte{[]byte(queryStateHistory), []byte(chainid), []byte(key2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	key3 := "key3"
	args = [][]byte{[]byte(queryStateHistory), []byte(chainid), []byte(key3)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCCustomerQueryStateRList(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"cpno":"0"}
	map1Bytes, _ := json.Marshal(map1)
	args := [][]byte{[]byte(queryStateRList), []byte(chainid), []byte("rlist1"), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	args = [][]byte{[]byte(queryStateRList), []byte(chainid), []byte("rlist2"), map1Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	args = [][]byte{[]byte(queryStateRList), []byte(chainid), []byte("rlist3"), map1Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCCustomerRegister(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	customer1 := new(CustomerEntity)
	customer1Bytes, _ := json.Marshal(customer1)
	args := [][]byte{[]byte(register), []byte(chainid),  customer1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	customer2 := new(CustomerEntity)
	customer2Bytes, _ := json.Marshal(customer2)
	args = [][]byte{[]byte(register), []byte(chainid), customer2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	customer3 := new(CustomerEntity)
	customer3Bytes, _ := json.Marshal(customer3)
	args = [][]byte{[]byte(register), []byte(chainid), []byte("rlist3"), customer3Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCCustomerModify(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	customer1 := new(CustomerEntity)
	customer1Bytes, _ := json.Marshal(customer1)
	args := [][]byte{[]byte(modify), []byte(chainid),  customer1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "modify failed with err: %s", res.Message)

	customer2 := new(CustomerEntity)
	customer2Bytes, _ := json.Marshal(customer2)
	args = [][]byte{[]byte(modify), []byte(chainid), customer2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "modify should have failed because customer not exist")

	args = [][]byte{[]byte(modify), []byte(chainid), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "modify should have failed because the customer should not be nil")
}

func TestRBCCustomerQueryOne(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	customerId1 := "customerId1"
	args := [][]byte{[]byte(queryOne), []byte(chainid), []byte(customerId1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "queryOne failed with err: %s", res.Message)

	customerId2 := "customerId2"
	args = [][]byte{[]byte(queryOne), []byte(chainid), []byte(customerId2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryOne should have failed because id not exist")

	args = [][]byte{[]byte(queryOne), []byte(chainid), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryOne should have failed because id should not be nil")

}

func TestRBCCustomerQueryAll(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(queryAll), []byte(chainid), }
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "queryOne failed with err: %s", res.Message)
}

func TestRBCCustomerQueryPerform(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"cpno":"0"}
	map1Bytes, _ := json.Marshal(map1)
	args := [][]byte{[]byte(queryPerform), []byte(chainid), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "queryPerform failed with err: %s", res.Message)

	map2 := map[string]string{"cpno":"1"}
	map2Bytes, _ := json.Marshal(map2)
	args = [][]byte{[]byte(queryPerform), []byte(chainid), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryPerform should have failed because cpno not exist")

	args = [][]byte{[]byte(queryPerform), []byte(chainid), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryPerform should have failed because id should not be nil")

}

func TestRBCCustomerQueryPerformByHeight(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"height":"0"}
	map1Bytes, _ := json.Marshal(map1)
	args := [][]byte{[]byte(queryPerform), []byte(chainid), map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "queryPerform failed with err: %s", res.Message)

	map2 := map[string]string{"height":"1"}
	map2Bytes, _ := json.Marshal(map2)
	args = [][]byte{[]byte(queryPerform), []byte(chainid), map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryPerform should have failed because this height not exist")

	args = [][]byte{[]byte(queryPerform), []byte(chainid), nil}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryPerform should have failed because height should not be nil")
}


func TestRBCCustomerSetRole(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	role1 := new(RoleEntity)
	role1Bytes, _ := json.Marshal(role1)
	args := [][]byte{[]byte(setRole), []byte(chainid), role1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "setRole failed with err: %s", res.Message)

	role2 := new(RoleEntity)
	role2Bytes, _ := json.Marshal(role2)
	args = [][]byte{[]byte(setRole), []byte(chainid), role2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "setRole should have failed because no channel id was provided")

	role3 := new(RoleEntity)
	role3Bytes, _ := json.Marshal(role3)
	args = [][]byte{[]byte(setRole), []byte(chainid), role3Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "setRole should have failed because the channel id does not exist")
}

func TestRBCCustomerDeleteRole(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	roleNo1 := "roleNo1"
	args := [][]byte{[]byte(setRole), []byte(chainid), []byte(roleNo1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	roleNo2 := "roleNo2"
	args = [][]byte{[]byte(setRole), []byte(chainid), []byte(roleNo2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	roleNo3 := "roleNo3"
	args = [][]byte{[]byte(setRole), []byte(chainid), []byte(roleNo3)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCCustomerGetRole(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	roleNo1 := "roleNo1"
	args := [][]byte{[]byte(getRole), []byte(chainid), []byte(roleNo1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	roleNo2 := "roleNo2"
	args = [][]byte{[]byte(getRole), []byte(chainid), []byte(roleNo2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	roleNo3 := "roleNo3"
	args = [][]byte{[]byte(getRole), []byte(chainid), []byte(roleNo3)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCCustomerGetAllRole(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(getAllRole), []byte(chainid)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	args = [][]byte{[]byte(getAllRole), []byte(chainid)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	args = [][]byte{[]byte(getAllRole), []byte(chainid)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}
