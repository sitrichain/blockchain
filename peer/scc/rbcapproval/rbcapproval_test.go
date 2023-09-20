package rbcapproval

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

	lq := new(RBCApproval)
	stub := shim.NewMockStub("LedgerQuerier", lq)
	if res := stub.MockInit("1", nil); res.Status != shim.OK {
		return nil, fmt.Errorf("Init failed for test ledger [%s] with message: %s", chainid, string(res.Message))
	}
	return stub, nil
}

func TestRBCApprovalQueryRBCCName(t *testing.T) {
	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	args := [][]byte{[]byte(chainid), []byte(queryRBCCName)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)
}

func TestRBCApprovalQueryStateHistory(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	key1 := "key1"
	args := [][]byte{[]byte(chainid), []byte(queryStateHistory),  []byte(key1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "queryStateHistory failed with err: %s", res.Message)

	key2 := "key2"
	args = [][]byte{[]byte(chainid), []byte(queryStateHistory),  []byte(key2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryStateHistory should have failed because no channel id was provided")

	key3 := "key3"
	args = [][]byte{[]byte(chainid), []byte(queryStateHistory),  []byte(key3)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "queryStateHistory should have failed because the channel id does not exist")
}

func TestRBCApprovalNewApproval(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	approval1 := new(ApprovalEntity)
	approval1Bytes, _ := json.Marshal(approval1)
	args := [][]byte{[]byte(chainid), []byte(newApproval),  approval1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	approval2 := new(ApprovalEntity)
	approval2Bytes, _ := json.Marshal(approval2)
	args = [][]byte{[]byte(chainid), []byte(newApproval),  approval2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	approval3 := new(ApprovalEntity)
	approval3Bytes, _ := json.Marshal(approval3)
	args = [][]byte{[]byte(chainid), []byte(newApproval),  approval3Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")

}

func TestRBCApprovalApproval(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	approval1 := new(ApprovalEntity)
	approval1Bytes, _ := json.Marshal(approval1)
	args := [][]byte{[]byte(chainid), []byte(approval),  approval1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	approval2 := new(ApprovalEntity)
	approval2Bytes, _ := json.Marshal(approval2)
	args = [][]byte{[]byte(chainid), []byte(approval), approval2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	approval3 := new(ApprovalEntity)
	approval3Bytes, _ := json.Marshal(approval3)
	args = [][]byte{[]byte(chainid), []byte(approval),  approval3Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCApprovalQueryOneApproval(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	key1 := "key1"
	args := [][]byte{[]byte(chainid), []byte(queryOneApproval),  []byte(key1)}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	key2 := "key1"
	args = [][]byte{[]byte(chainid), []byte(queryOneApproval),  []byte(key2)}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	key3 := "key1"
	args = [][]byte{[]byte(chainid), []byte(queryOneApproval),  []byte(key3)}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}

func TestRBCApprovalQueryAllApproval(t *testing.T) {

	chainid := "mytestchainid1"
	path := "/var/rongzer/test1/"
	stub, err := setupTestLedger(chainid, path)
	defer os.RemoveAll(path)
	if err != nil {
		t.Fatalf(err.Error())
	}

	map1 := map[string]string{"cpno":"0", "approvalStatus":"0"}
	map1Bytes, _ := json.Marshal(map1)
	args := [][]byte{[]byte(chainid), []byte(queryAllApproval),  map1Bytes}
	res := stub.MockInvoke("1", args)
	assert.Equal(t, int32(shim.OK), res.Status, "GetChainInfo failed with err: %s", res.Message)

	map2 := map[string]string{"cpno":"0", "approvalStatus":"0"}
	map2Bytes, _ := json.Marshal(map2)
	args = [][]byte{[]byte(chainid), []byte(queryAllApproval),  map2Bytes}
	res = stub.MockInvoke("2", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because no channel id was provided")

	map3 := map[string]string{"cpno":"0", "approvalStatus":"0"}
	map3Bytes, _ := json.Marshal(map3)
	args = [][]byte{[]byte(chainid), []byte(queryAllApproval),  map3Bytes}
	res = stub.MockInvoke("3", args)
	assert.Equal(t, int32(shim.ERROR), res.Status, "GetChainInfo should have failed because the channel id does not exist")
}
