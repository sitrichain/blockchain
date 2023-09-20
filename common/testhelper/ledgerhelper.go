package testhelper

import (
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"testing"

	"github.com/gogo/protobuf/proto"
	lutils "github.com/rongzer/blockchain/peer/ledger/util"
	"github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

//BlockGenerator generates a series of blocks for testing
type BlockGenerator struct {
	blockNum     uint64
	previousHash []byte
	signTxs      bool
	t            *testing.T
}

// NewBlockGenerator instantiates new BlockGenerator for testing
func NewBlockGenerator(t *testing.T, ledgerID string, signTxs bool) (*BlockGenerator, *common.Block) {
	gb, err := MakeGenesisBlock(ledgerID)
	AssertNoError(t, err, "")
	gb.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlags(len(gb.Data.Data))
	return &BlockGenerator{1, gb.GetHeader().Hash(), signTxs, t}, gb
}

// NextBlock constructs next block in sequence that includes a number of transactions - one per simulationResults
func (bg *BlockGenerator) NextBlock(simulationResults [][]byte) *common.Block {
	block := ConstructBlock(bg.t, bg.blockNum, bg.previousHash, simulationResults, bg.signTxs)
	bg.blockNum++
	bg.previousHash = block.Header.Hash()
	return block
}

// NextBlock constructs next block in sequence that includes a number of transactions - one per simulationResults
func (bg *BlockGenerator) NextBlockWithSpecEnvBytes(envBytes []byte) *common.Block {
	block := newBlockWithSpecEnvBytes(envBytes, bg.blockNum, bg.previousHash)
	bg.blockNum++
	bg.previousHash = block.Header.Hash()
	return block
}

// ConstructTransaction constructs a transaction for testing
func ConstructTransaction(_ *testing.T, simulationResults []byte, sign bool) (*common.Envelope, string, error) {
	ccid := &pb.ChaincodeID{
		Name:    "foo",
		Version: "v1",
	}
	//response := &pb.Response{Status: 200}
	var txID string
	var txEnv *common.Envelope
	var err error
	if sign {
		txEnv, txID, err = ConstructSingedTxEnvWithDefaultSigner("testchainid", ccid, nil, simulationResults, nil, nil)
	} else {
		txEnv, txID, err = ConstructUnsingedTxEnv("testchainid", ccid, nil, simulationResults, nil, nil)
	}
	return txEnv, txID, err
}

// ConstructBlock constructs a single block
func ConstructBlock(t *testing.T, blockNum uint64, previousHash []byte, simulationResults [][]byte, sign bool) *common.Block {
	var envs []*common.Envelope
	for i := 0; i < len(simulationResults); i++ {
		env, _, err := ConstructTransaction(t, simulationResults[i], sign)
		if err != nil {
			t.Fatalf("ConstructTestTransaction failed, err %s", err)
		}
		envs = append(envs, env)
	}
	return newBlock(envs, blockNum, previousHash)
}

//ConstructTestBlock constructs a single block with random contents
func ConstructTestBlock(t *testing.T, blockNum uint64, numTx int, txSize int) *common.Block {
	var simulationResults [][]byte
	for i := 0; i < numTx; i++ {
		simulationResults = append(simulationResults, ConstructRandomBytes(t, txSize))
	}
	return ConstructBlock(t, blockNum, ConstructRandomBytes(t, 32), simulationResults, false)
}

// ConstructRandomBytes constructs random bytes of given size
func ConstructRandomBytes(t testing.TB, size int) []byte {
	value := make([]byte, size)
	_, err := rand.Read(value)
	if err != nil {
		t.Fatalf("Error while generating random bytes: %s", err)
	}
	return value
}

func newBlock(env []*common.Envelope, blockNum uint64, previousHash []byte) *common.Block {
	block := common.NewBlock(blockNum, previousHash)
	for i := 0; i < len(env); i++ {
		txEnvBytes, _ := proto.Marshal(env[i])
		block.Data.Data = append(block.Data.Data, txEnvBytes)
	}
	block.Header.DataHash = block.Data.Hash()
	utils.InitBlockMetadata(block)

	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlags(len(env))

	return block
}

func newBlockWithSpecEnvBytes(envBytes []byte, blockNum uint64, previousHash []byte) *common.Block {
	block := common.NewBlock(blockNum, previousHash)
	block.Data.Data = append(block.Data.Data, envBytes)
	block.Header.DataHash = block.Data.Hash()
	utils.InitBlockMetadata(block)

	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = lutils.NewTxValidationFlags(1)

	return block
}

// AssertNil varifies that the value is nil
func AssertNil(t testing.TB, value interface{}) {
	if !isNil(value) {
		t.Fatalf("Value not nil. value=[%#v]\n %s", value, getCallerInfo())
	}
}

// AssertEquals varifies that the two values are equal
func AssertEquals(t testing.TB, actual interface{}, expected interface{}) {
	t.Logf("%s: AssertEquals [%#v] and [%#v]", getCallerInfo(), actual, expected)
	if expected == nil && isNil(actual) {
		return
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Values are not equal.\n Actual=[%#v], \n Expected=[%#v]\n %s", actual, expected, getCallerInfo())
	}
}

// AssertNoError varifies that the err is nil
func AssertNoError(t testing.TB, err error, message string) {
	if err != nil {
		t.Fatalf("%s - Error: %s\n %s", message, err, getCallerInfo())
	}
}

// Contains returns true iff the `value` is present in the `slice`
func Contains(slice interface{}, value interface{}) bool {
	array := reflect.ValueOf(slice)
	for i := 0; i < array.Len(); i++ {
		element := array.Index(i).Interface()
		if value == element || reflect.DeepEqual(element, value) {
			return true
		}
	}
	return false
}

func isNil(in interface{}) bool {
	return in == nil || reflect.ValueOf(in).IsNil() || (reflect.TypeOf(in).Kind() == reflect.Slice && reflect.ValueOf(in).Len() == 0)
}

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "Could not retrieve caller's info"
	}
	return fmt.Sprintf("CallerInfo = [%s:%d]", file, line)
}

// TestRandomNumberGenerator a random number generator for testing
type TestRandomNumberGenerator struct {
	rand      *rand.Rand
	maxNumber int
}

// Next generates next random number
func (randNumGenerator *TestRandomNumberGenerator) Next() int {
	return randNumGenerator.rand.Intn(randNumGenerator.maxNumber)
}
