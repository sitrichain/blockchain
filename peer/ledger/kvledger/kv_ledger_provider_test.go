package kvledger

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/ledger/blkstorage/fsblkstorage"
	"github.com/rongzer/blockchain/common/testhelper"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/ledger/queryresult"
	putils "github.com/rongzer/blockchain/protos/utils"
)

// 测试Provider的各个方法
func TestLedgerProvider(t *testing.T) {
	// 创建测试环境
	testPath := "/tmp/rongzer/test/kvledger5"
	conf.V.FileSystemPath = testPath
	defer func() {
		os.RemoveAll(testPath)
	}()
	numLedgers := 10
	provider, _ := NewProvider()
	defer provider.Close()
	existingLedgerIDs, err := provider.List()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, len(existingLedgerIDs), 0)
	for i := 0; i < numLedgers; i++ {
		genesisBlock, _ := testhelper.MakeGenesisBlock(constructTestLedgerID(i))
		provider.Create(genesisBlock)
	}
	existingLedgerIDs, err = provider.List()
	testhelper.AssertNoError(t, err, "")
	for _, id := range existingLedgerIDs {
		fmt.Println(id, "........")
	}
	testhelper.AssertEquals(t, len(existingLedgerIDs), numLedgers)

	for i := 0; i < numLedgers; i++ {
		status, _ := provider.Exists(constructTestLedgerID(i))
		testhelper.AssertEquals(t, status, true)
		ledger, err := provider.Open(constructTestLedgerID(i))
		testhelper.AssertNoError(t, err, "")
		bcInfo, err := ledger.GetBlockchainInfo()
		ledger.Close()
		testhelper.AssertNoError(t, err, "")
		testhelper.AssertEquals(t, bcInfo.Height, uint64(1))
	}
	gb, _ := testhelper.MakeGenesisBlock(constructTestLedgerID(2))
	_, err = provider.Create(gb)
	testhelper.AssertEquals(t, err, ErrLedgerIDExists)

	status, err := provider.Exists(constructTestLedgerID(numLedgers))
	testhelper.AssertNoError(t, err, "Failed to check for ledger existence")
	testhelper.AssertEquals(t, false, status)

	_, err = provider.Open(constructTestLedgerID(numLedgers))
	testhelper.AssertEquals(t, err, ErrNonExistingLedgerID)
}

// 测试idStore崩溃后能否正常恢复
func TestIdStoreRecovery_beforeFail(t *testing.T) {
	// 创建测试环境
	conf.V.FileSystemPath = "/tmp/rongzer/test/kvledger6"
	provider, _ := NewProvider()
	defer provider.Close()
	// now create the genesis block
	genesisBlock, _ := testhelper.MakeGenesisBlock(constructTestLedgerID(1))
	ledger, _ := provider.(*Provider).openInternal(constructTestLedgerID(1))
	ledger.Commit(genesisBlock)
	ledger.Close()

	// 设置idStore的flag为underconstruction，以模拟ledgerid正在创建
	provider.(*Provider).idStore.setUnderConstructionFlag(constructTestLedgerID(1))
}

func TestIdStoreRecovery_AfterFail(t *testing.T) {
	// 创建测试环境
	conf.V.FileSystemPath = "/tmp/rongzer/test/kvledger6"
	defer func() {
		os.RemoveAll("/tmp/rongzer/test/kvledger6")
	}()
	// construct a new provider to invoke recovery
	provider, err := NewProvider()
	defer provider.Close()
	testhelper.AssertNoError(t, err, "Provider failed to recover an underConstructionLedger")
	// verify the underecoveryflag and open the ledger
	flag, err := provider.(*Provider).idStore.getUnderConstructionFlag()
	testhelper.AssertNoError(t, err, "Failed to read the underconstruction flag")
	testhelper.AssertEquals(t, flag, "")
	ledger, err := provider.Open(constructTestLedgerID(1))
	testhelper.AssertNoError(t, err, "Failed to open the ledger")
	ledger.Close()
}

func TestMultipleLedgerBasicRead(t *testing.T) {
	// 创建测试环境
	conf.V.FileSystemPath = "/tmp/rongzer/test/kvledger7"
	numLedgers := 10
	provider, _ := NewProvider()
	defer provider.Close()
	ledgers := make([]ledger.PeerLedger, numLedgers)
	for i := 0; i < numLedgers; i++ {
		bg, gb := testhelper.NewBlockGenerator(t, constructTestLedgerID(i), false)
		l, err := provider.Create(gb)
		testhelper.AssertNoError(t, err, "")
		ledgers[i] = l
		s, _ := l.NewTxSimulator()
		err = s.SetState("ns", "testKey", []byte(fmt.Sprintf("testValue_%d", i)))
		s.Done()
		testhelper.AssertNoError(t, err, "")
		res, err := s.GetTxSimulationResults()
		testhelper.AssertNoError(t, err, "")
		b := bg.NextBlock([][]byte{res})
		err = l.Commit(b)
		l.Close()
		testhelper.AssertNoError(t, err, "")
	}
}

func TestMultipleLedgerBasicWrite(t *testing.T) {
	// 创建测试环境
	conf.V.FileSystemPath = "/tmp/rongzer/test/kvledger7"
	defer func() {
		os.RemoveAll("/tmp/rongzer/test/kvledger7")
	}()
	provider, _ := NewProvider()
	defer provider.Close()
	numLedgers := 10
	ledgers := make([]ledger.PeerLedger, numLedgers)
	for i := 0; i < numLedgers; i++ {
		l, err := provider.Open(constructTestLedgerID(i))
		testhelper.AssertNoError(t, err, "")
		ledgers[i] = l
	}

	for i, l := range ledgers {
		q, _ := l.NewQueryExecutor()
		val, err := q.GetState("ns", "testKey")
		q.Done()
		testhelper.AssertNoError(t, err, "")
		testhelper.AssertEquals(t, val, []byte(fmt.Sprintf("testValue_%d", i)))
		l.Close()
	}
}

func TestLedgerBackup(t *testing.T) {
	ledgerid := "TestLedger"
	originalPath := "/tmp/rongzer/test/kvledgerOld"
	restorePath := "/tmp/rongzer/test/kvledgerNew"
	conf.V.Ledger.EnableHistoryDatabase = true

	// create and populate a ledger in the original environment
	conf.V.FileSystemPath = originalPath
	defer func() {
		os.RemoveAll(originalPath)
		os.RemoveAll(restorePath)
	}()
	provider, _ := NewProvider()
	bg, gb := testhelper.NewBlockGenerator(t, ledgerid, false)
	gbHash := gb.Header.Hash()
	ledger, _ := provider.Create(gb)

	simulator, _ := ledger.NewTxSimulator()
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	block1 := bg.NextBlock([][]byte{simRes})
	ledger.Commit(block1)

	simulator, _ = ledger.NewTxSimulator()
	simulator.SetState("ns1", "key1", []byte("value4"))
	simulator.SetState("ns1", "key2", []byte("value5"))
	simulator.SetState("ns1", "key3", []byte("value6"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	block2 := bg.NextBlock([][]byte{simRes})
	ledger.Commit(block2)

	ledger.Close()
	provider.Close()

	// Create restore environment
	conf.V.FileSystemPath = restorePath

	// remove the statedb, historydb, and block indexes (they are supposed to be auto created during opening of an existing ledger)
	// and rename the originalPath to restorePath
	testhelper.AssertNoError(t, os.RemoveAll(ledgerconfig.GetStateLevelDBPath()), "")
	testhelper.AssertNoError(t, os.RemoveAll(ledgerconfig.GetHistoryLevelDBPath()), "")
	testhelper.AssertNoError(t, os.RemoveAll(filepath.Join(ledgerconfig.GetBlockStorePath(), fsblkstorage.IndexDir)), "")
	testhelper.AssertNoError(t, os.Rename(originalPath, restorePath), "")

	// Instantiate the ledger from restore environment and this should behave exactly as it would have in the original environment
	provider, _ = NewProvider()
	defer provider.Close()

	_, err := provider.Create(gb)
	testhelper.AssertEquals(t, err, ErrLedgerIDExists)

	ledger, _ = provider.Open(ledgerid)
	defer ledger.Close()

	block1Hash := block1.Header.Hash()
	block2Hash := block2.Header.Hash()
	bcInfo, _ := ledger.GetBlockchainInfo()
	testhelper.AssertEquals(t, bcInfo, &common.BlockchainInfo{
		Height: 3, CurrentBlockHash: block2Hash, PreviousBlockHash: block1Hash})

	b0, _ := ledger.GetBlockByHash(gbHash)
	testhelper.AssertEquals(t, b0, gb)

	b1, _ := ledger.GetBlockByHash(block1Hash)
	testhelper.AssertEquals(t, b1, block1)

	b2, _ := ledger.GetBlockByHash(block2Hash)
	testhelper.AssertEquals(t, b2, block2)

	b0, _ = ledger.GetBlockByNumber(0)
	testhelper.AssertEquals(t, b0, gb)

	b1, _ = ledger.GetBlockByNumber(1)
	testhelper.AssertEquals(t, b1, block1)

	b2, _ = ledger.GetBlockByNumber(2)
	testhelper.AssertEquals(t, b2, block2)

	// get the tran id from the 2nd block, then use it to test GetTransactionByID()
	txEnvBytes2 := block1.Data.Data[0]
	txEnv2, err := putils.GetEnvelopeFromBlock(txEnvBytes2)
	testhelper.AssertNoError(t, err, "Error upon GetEnvelopeFromBlock")
	payload2, err := putils.GetPayload(txEnv2)
	testhelper.AssertNoError(t, err, "Error upon GetPayload")
	chdr, err := putils.UnmarshalChannelHeader(payload2.Header.ChannelHeader)
	testhelper.AssertNoError(t, err, "Error upon GetChannelHeaderFromBytes")
	txID2 := chdr.TxId
	processedTran2, err := ledger.GetTransactionByID(txID2)
	testhelper.AssertNoError(t, err, "Error upon GetTransactionByID")
	// get the tran envelope from the retrieved ProcessedTransaction
	retrievedTxEnv2 := processedTran2.TransactionEnvelope
	testhelper.AssertEquals(t, retrievedTxEnv2, txEnv2)

	qe, _ := ledger.NewQueryExecutor()
	value1, _ := qe.GetState("ns1", "key1")
	testhelper.AssertEquals(t, value1, []byte("value4"))

	hqe, err := ledger.NewHistoryQueryExecutor()
	testhelper.AssertNoError(t, err, "")
	itr, err := hqe.GetHistoryForKey("ns1", "key1")
	testhelper.AssertNoError(t, err, "")
	defer itr.Close()

	result1, err := itr.Next()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, result1.(*queryresult.KeyModification).Value, []byte("value1"))
	result2, err := itr.Next()
	testhelper.AssertNoError(t, err, "")
	testhelper.AssertEquals(t, result2.(*queryresult.KeyModification).Value, []byte("value4"))
}

func constructTestLedgerID(i int) string {
	return fmt.Sprintf("ledger_%06d", i)
}
