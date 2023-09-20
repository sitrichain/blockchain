package kvledger

import (
	"github.com/rongzer/blockchain/common/testhelper"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/ledger/queryresult"
	"github.com/rongzer/blockchain/protos/peer"
	putils "github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"testing"
)

// 测试peer账本的块存储/获取功能
func TestKVLedgerBlockStorage(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	testPath := "/tmp/rongzer/test/kvledger"
	viper.Set("peer.fileSystemPath", testPath)
	defer func() {
		os.RemoveAll(testPath)
	}()
	// 创建一个Provider对象
	provider, _ := NewProvider()
	defer provider.Close()
	// 创建一个"模拟块"生成器
	bg1, gb1 := testhelper.NewBlockGenerator(t, "testLedger", false)
	gbHash := gb1.Header.Hash()
	// 用指定的创世块创建一个kvLedger对象
	ledger, _ := provider.Create(gb1)
	defer ledger.Close()
	// 从账本中获取链信息——当前链高度、当前块哈希、上一个块哈希，进行比对
	bcInfo, _ := ledger.GetBlockchainInfo()
	testhelper.AssertEquals(t, bcInfo, &common.BlockchainInfo{
		Height: 1, CurrentBlockHash: gbHash, PreviousBlockHash: nil})
	// 创建一个交易模拟器
	simulator, _ := ledger.NewTxSimulator()
	// 将三笔提案的模拟执行结果写入读写集
	simulator.SetState("ns1", "key1", []byte("value1"))
	simulator.SetState("ns1", "key2", []byte("value2"))
	simulator.SetState("ns1", "key3", []byte("value3"))
	simulator.Done()
	// 将读写集序列化为[]byte
	simRes, _ := simulator.GetTxSimulationResults()
	// 根据序列化后的读写集，先构造三笔交易，再将这三笔交易塞入envelopes中，最后把envelopes结合header和metadata组装为下一个块
	block1 := bg1.NextBlock([][]byte{simRes})
	// 向账本添加这个块
	ledger.Commit(block1)
	// 再次从账本中获取链信息，进行比对，观察账本是否更新成功
	bcInfo, _ = ledger.GetBlockchainInfo()
	block1Hash := block1.Header.Hash()
	testhelper.AssertEquals(t, bcInfo, &common.BlockchainInfo{
		Height: 2, CurrentBlockHash: block1Hash, PreviousBlockHash: gbHash})
	// 重复上述步骤，向账本添加block2
	simulator, _ = ledger.NewTxSimulator()
	simulator.SetState("ns1", "key1", []byte("value4"))
	simulator.SetState("ns1", "key2", []byte("value5"))
	simulator.SetState("ns1", "key3", []byte("value6"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	block2 := bg1.NextBlock([][]byte{simRes})
	ledger.Commit(block2)

	bcInfo, _ = ledger.GetBlockchainInfo()
	block2Hash := block2.Header.Hash()
	testhelper.AssertEquals(t, bcInfo, &common.BlockchainInfo{
		Height: 3, CurrentBlockHash: block2Hash, PreviousBlockHash: block1Hash})
	// 验证kvLedger对象的GetBlockByHash方法
	b0, _ := ledger.GetBlockByHash(gbHash)
	testhelper.AssertEquals(t, b0, gb1)

	b1, _ := ledger.GetBlockByHash(block1Hash)
	testhelper.AssertEquals(t, b1, block1)
	// 验证kvLedger对象的GetBlockByNumber方法
	b0, _ = ledger.GetBlockByNumber(0)
	testhelper.AssertEquals(t, b0, gb1)

	b1, _ = ledger.GetBlockByNumber(1)
	testhelper.AssertEquals(t, b1, block1)

	// 从block1中反序列化出第一笔交易的交易id，据此验证kvLedger对象的GetTransactionByID方法
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
	retrievedTxEnv2 := processedTran2.TransactionEnvelope
	testhelper.AssertEquals(t, retrievedTxEnv2, txEnv2)

	// 根据交易id验证kvLedger对象的GetBlockByTxID方法
	b1, _ = ledger.GetBlockByTxID(txID2)
	testhelper.AssertEquals(t, b1, block1)

	// 根据交易id验证kvLedger对象的GetTxValidationCodeByTxID方法
	validCode, _ := ledger.GetTxValidationCodeByTxID(txID2)
	testhelper.AssertEquals(t, validCode, peer.TxValidationCode_VALID)
}

// 测试场景1：block2已写入peer账本块存储文件，但未应用此块更新世界状态数据库，
// 即kvLedger对象的Commit方法只完成了验证区块和向存储文件添加区块的工作时，peer"挂"了
func TestKVLedgerRecoveryNotUpdateStateDb_beforeFail(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger1")
	// 创建一个Provider对象
	provider, _ := NewProvider()
	defer provider.Close()
	// 创建一个全局的"模拟块"生成器
	bg, gb := testhelper.NewBlockGenerator(t, "testLedger", false)
	// 用指定的创世块创建一个kvLedger对象
	ledger, _ := provider.Create(gb)
	defer ledger.Close()
	// 创建并向账本提交block1
	simulator, _ := ledger.NewTxSimulator()
	simulator.SetState("ns1", "key1", []byte("value1.1"))
	simulator.SetState("ns1", "key2", []byte("value2.1"))
	simulator.SetState("ns1", "key3", []byte("value3.1"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	block1 := bg.NextBlock([][]byte{simRes})
	ledger.Commit(block1)
	// 创建block2
	simulator, _ = ledger.NewTxSimulator()
	simulator.SetState("ns1", "key1", []byte("value1.2"))
	simulator.SetState("ns1", "key2", []byte("value2.2"))
	simulator.SetState("ns1", "key3", []byte("value3.2"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	block2 := bg.NextBlock([][]byte{simRes})
	// kvLedger对象的Commit方法只完成了的这部分工作——验证区块和向存储文件添加区块
	ledger.(*kvLedger).txtmgmt.ValidateAndPrepare(block2, true)
	err := ledger.(*kvLedger).blockStore.AddBlock(block2)
	assert.NoError(t, err)
	// 从世界状态数据库中查询key1、key2、key3的值，应为block1提交后的值
	simulator, _ = ledger.NewTxSimulator()
	value, _ := simulator.GetState("ns1", "key1")
	testhelper.AssertEquals(t, value, []byte("value1.1"))
	value, _ = simulator.GetState("ns1", "key2")
	testhelper.AssertEquals(t, value, []byte("value2.1"))
	value, _ = simulator.GetState("ns1", "key3")
	testhelper.AssertEquals(t, value, []byte("value3.1"))
	// 从世界状态数据库中查询LastSavepoint，应为数据库中保存的当前块高度——1
	stateDBSavepoint, _ := ledger.(*kvLedger).txtmgmt.GetLastSavepoint()
	testhelper.AssertEquals(t, stateDBSavepoint.BlockNum, uint64(1))
	// 同理，验证历史数据库是否更新——未更新
	if ledgerconfig.IsHistoryDBEnabled() == true {
		qhistory, _ := ledger.NewHistoryQueryExecutor()
		itr, _ := qhistory.GetHistoryForKey("ns1", "key1")
		count := 0
		for {
			kmod, err := itr.Next()
			testhelper.AssertNoError(t, err, "Error upon Next()")
			if kmod == nil {
				break
			}
			retrievedValue := kmod.(*queryresult.KeyModification).Value
			count++
			expectedValue := []byte("value1." + strconv.Itoa(count))
			testhelper.AssertEquals(t, retrievedValue, expectedValue)
		}
		testhelper.AssertEquals(t, count, 1)
		historyDBSavepoint, _ := ledger.(*kvLedger).historyDB.GetLastSavepoint()
		testhelper.AssertEquals(t, historyDBSavepoint.BlockNum, uint64(1))
	}

	simulator.Done()
}

// 假设，这个时候peer"恢复"了，会重新调用NewProvider()函数，此时世界状态数据库会根据块存储文件进行recover，即把block2进行应用，
// 因为在NewProvider函数中调用了Provider的recoverUnderConstructionLedger方法
func TestKVLedgerRecoveryNotUpdateStateDb_afterFail(t *testing.T) {
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger1")
	defer func() {
		os.RemoveAll("/tmp/rongzer/test/kvledger1")
	}()
	provider, _ := NewProvider()
	defer provider.Close()
	ledger, _ := provider.Open("testLedger")
	defer ledger.Close()
	// 从世界状态数据库中查询key1、key2、key3的值，应为block2应用后的值
	simulator, _ := ledger.NewTxSimulator()
	value, _ := simulator.GetState("ns1", "key1")
	testhelper.AssertEquals(t, value, []byte("value1.2"))
	value, _ = simulator.GetState("ns1", "key2")
	testhelper.AssertEquals(t, value, []byte("value2.2"))
	value, _ = simulator.GetState("ns1", "key3")
	testhelper.AssertEquals(t, value, []byte("value3.2"))
	stateDBSavepoint, _ := ledger.(*kvLedger).txtmgmt.GetLastSavepoint()
	testhelper.AssertEquals(t, stateDBSavepoint.BlockNum, uint64(2))
	// 同理，验证历史数据库是否更新——已更新
	if ledgerconfig.IsHistoryDBEnabled() == true {
		qhistory, _ := ledger.NewHistoryQueryExecutor()
		itr, _ := qhistory.GetHistoryForKey("ns1", "key1")
		count := 0
		for {
			kmod, err := itr.Next()
			testhelper.AssertNoError(t, err, "Error upon Next()")
			if kmod == nil {
				break
			}
			retrievedValue := kmod.(*queryresult.KeyModification).Value
			count++
			expectedValue := []byte("value1." + strconv.Itoa(count))
			testhelper.AssertEquals(t, retrievedValue, expectedValue)
		}
		testhelper.AssertEquals(t, count, 2)
		historyDBSavepoint, _ := ledger.(*kvLedger).historyDB.GetLastSavepoint()
		testhelper.AssertEquals(t, historyDBSavepoint.BlockNum, uint64(2))
	}
	simulator.Done()

}

// 测试场景2：block1已被提交应用于更新世界状态数据库，但未被应用于更新历史数据库，
// 即kvLedger对象的Commit方法只完成了验证区块/向存储文件添加区块/更新世界状态数据库的工作时，peer"挂"了
func TestKVLedgerRecoveryNotUpdateHistoryDb_beforeFail(t *testing.T) {
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger2")
	provider, _ := NewProvider()
	defer provider.Close()
	// 创建一个"模拟块"生成器
	bg, gb := testhelper.NewBlockGenerator(t, "testLedger", false)
	// 用指定的创世块创建一个kvLedger对象
	ledger, _ := provider.Create(gb)
	defer ledger.Close()
	simulator, _ := ledger.NewTxSimulator()
	// 构造block1
	simulator.SetState("ns1", "key1", []byte("value1.1"))
	simulator.SetState("ns1", "key2", []byte("value2.1"))
	simulator.SetState("ns1", "key3", []byte("value3.1"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	block1 := bg.NextBlock([][]byte{simRes})
	// kvLedger对象的Commit方法只完成了的这部分工作——验证区块/向存储文件添加区块/更新世界状态数据库
	ledger.(*kvLedger).txtmgmt.ValidateAndPrepare(block1, true)
	err := ledger.(*kvLedger).blockStore.AddBlock(block1)
	err = ledger.(*kvLedger).txtmgmt.Commit()
	assert.NoError(t, err)
	// 检查历史数据库是否更新——未更新
	if ledgerconfig.IsHistoryDBEnabled() == true {
		qhistory, _ := ledger.NewHistoryQueryExecutor()
		itr, _ := qhistory.GetHistoryForKey("ns1", "key1")
		count := 0
		for {
			kmod, err := itr.Next()
			testhelper.AssertNoError(t, err, "Error upon Next()")
			if kmod == nil {
				break
			}
			retrievedValue := kmod.(*queryresult.KeyModification).Value
			count++
			expectedValue := []byte("value1." + strconv.Itoa(count))
			testhelper.AssertEquals(t, retrievedValue, expectedValue)
		}
		testhelper.AssertEquals(t, count, 0)
		historyDBSavepoint, _ := ledger.(*kvLedger).historyDB.GetLastSavepoint()
		testhelper.AssertEquals(t, historyDBSavepoint.BlockNum, uint64(0))
	}
	simulator.Done()
}

// 假设，这个时候peer"恢复"了，会重新调用NewProvider()函数，此时历史数据库会根据块存储文件进行recover，即把block1进行应用，
// 因为在NewProvider函数中调用了Provider的recoverUnderConstructionLedger方法
func TestKVLedgerRecoveryNotUpdateHistoryDb_afterFail(t *testing.T) {
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger2")
	defer func() {
		os.RemoveAll("/tmp/rongzer/test/kvledger2")
	}()
	provider, _ := NewProvider()
	defer provider.Close()
	ledger, _ := provider.Open("testLedger")
	defer ledger.Close()
	simulator, _ := ledger.NewTxSimulator()
	stateDBSavepoint, _ := ledger.(*kvLedger).txtmgmt.GetLastSavepoint()
	testhelper.AssertEquals(t, stateDBSavepoint.BlockNum, uint64(1))
	// 检查历史数据库是否更新——已更新
	if ledgerconfig.IsHistoryDBEnabled() == true {
		qhistory, _ := ledger.NewHistoryQueryExecutor()
		itr, _ := qhistory.GetHistoryForKey("ns1", "key1")
		count := 0
		for {
			kmod, err := itr.Next()
			testhelper.AssertNoError(t, err, "Error upon Next()")
			if kmod == nil {
				break
			}
			retrievedValue := kmod.(*queryresult.KeyModification).Value
			count++
			expectedValue := []byte("value1." + strconv.Itoa(count))
			testhelper.AssertEquals(t, retrievedValue, expectedValue)
		}
		testhelper.AssertEquals(t, count, 1)
		historyDBSavepoint, _ := ledger.(*kvLedger).historyDB.GetLastSavepoint()
		testhelper.AssertEquals(t, historyDBSavepoint.BlockNum, uint64(1))
	}
	simulator.Done()
}

// 测试场景3：block2已被提交应用于历史数据库，但未被应用于更新世界状态数据库，此场景为罕见场景。
// 即kvLedger对象的Commit方法只完成了验证区块/向存储文件添加区块/更新历史数据库的工作时，peer"挂"了
func TestKVLedgerRecoveryRareScenario_beforeFail(t *testing.T) {
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger3")
	provider, _ := NewProvider()
	defer provider.Close()
	// 创建一个"模拟块"生成器
	bg, gb := testhelper.NewBlockGenerator(t, "testLedger", false)
	// 用指定的创世块创建一个kvLedger对象
	ledger, _ := provider.Create(gb)
	defer ledger.Close()
	simulator, _ := ledger.NewTxSimulator()
	// 构造并提交block1
	simulator.SetState("ns1", "key1", []byte("value1.1"))
	simulator.SetState("ns1", "key2", []byte("value2.1"))
	simulator.SetState("ns1", "key3", []byte("value3.1"))
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	block1 := bg.NextBlock([][]byte{simRes})
	ledger.Commit(block1)
	// 构造block2
	simulator.SetState("ns1", "key1", []byte("value1.2"))
	simulator.SetState("ns1", "key2", []byte("value2.2"))
	simulator.SetState("ns1", "key3", []byte("value3.2"))
	simulator.Done()
	simRes, _ = simulator.GetTxSimulationResults()
	block2 := bg.NextBlock([][]byte{simRes})
	// kvLedger对象的Commit方法只完成了的这部分工作——验证区块/向存储文件添加区块/更新历史数据库
	ledger.(*kvLedger).txtmgmt.ValidateAndPrepare(block2, true)
	err := ledger.(*kvLedger).blockStore.AddBlock(block2)
	if ledgerconfig.IsHistoryDBEnabled() == true {
		err = ledger.(*kvLedger).historyDB.Commit(block2)
	}
	assert.NoError(t, err)
	// 检查世界状态数据库是否更新——未更新
	simulator, _ = ledger.NewTxSimulator()
	value, _ := simulator.GetState("ns1", "key1")
	testhelper.AssertEquals(t, value, []byte("value1.1"))
	value, _ = simulator.GetState("ns1", "key2")
	testhelper.AssertEquals(t, value, []byte("value2.1"))
	value, _ = simulator.GetState("ns1", "key3")
	testhelper.AssertEquals(t, value, []byte("value3.1"))
	stateDBSavepoint, _ := ledger.(*kvLedger).txtmgmt.GetLastSavepoint()
	testhelper.AssertEquals(t, stateDBSavepoint.BlockNum, uint64(1))
	// 检查历史数据库是否更新——已更新
	if ledgerconfig.IsHistoryDBEnabled() == true {
		qhistory, _ := ledger.NewHistoryQueryExecutor()
		itr, _ := qhistory.GetHistoryForKey("ns1", "key1")
		count := 0
		for {
			kmod, err := itr.Next()
			testhelper.AssertNoError(t, err, "Error upon Next()")
			if kmod == nil {
				break
			}
			retrievedValue := kmod.(*queryresult.KeyModification).Value
			count++
			expectedValue := []byte("value1." + strconv.Itoa(count))
			testhelper.AssertEquals(t, retrievedValue, expectedValue)
		}
		testhelper.AssertEquals(t, count, 2)
		historyDBSavepoint, _ := ledger.(*kvLedger).historyDB.GetLastSavepoint()
		testhelper.AssertEquals(t, historyDBSavepoint.BlockNum, uint64(2))
	}
}

// 假设，这个时候peer"恢复"了，会重新调用NewProvider()函数，此时世界状态数据库会根据块存储文件进行recover，即把block2进行应用，
// 因为在NewProvider函数中调用了Provider的recoverUnderConstructionLedger方法
func TestKVLedgerRecoveryRareScenario_afterFail(t *testing.T) {
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	viper.Set("peer.fileSystemPath", "/tmp/rongzer/test/kvledger3")
	defer func() {
		os.RemoveAll("/tmp/rongzer/test/kvledger3")
	}()
	provider, _ := NewProvider()
	defer provider.Close()
	ledger, _ := provider.Open("testLedger")
	defer ledger.Close()
	// 检查世界状态数据库是否更新——已更新
	simulator, _ := ledger.NewTxSimulator()
	value, _ := simulator.GetState("ns1", "key1")
	testhelper.AssertEquals(t, value, []byte("value1.2"))
	value, _ = simulator.GetState("ns1", "key2")
	testhelper.AssertEquals(t, value, []byte("value2.2"))
	value, _ = simulator.GetState("ns1", "key3")
	testhelper.AssertEquals(t, value, []byte("value3.2"))
	stateDBSavepoint, _ := ledger.(*kvLedger).txtmgmt.GetLastSavepoint()
	testhelper.AssertEquals(t, stateDBSavepoint.BlockNum, uint64(2))
	simulator.Done()
}

// 测试peer账本的创建索引(RList)功能
func TestKVLedgerIndexState(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	// 创建测试环境
	testPath := "/tmp/rongzer/test/kvledger4"
	viper.Set("peer.fileSystemPath", testPath)
	defer func() {
		os.RemoveAll(testPath)
	}()
	// 创建一个Provider对象
	provider, _ := NewProvider()
	defer provider.Close()
	// 创建一个"模拟块"生成器
	bg1, gb1 := testhelper.NewBlockGenerator(t, "testLedger", false)
	gbHash := gb1.Header.Hash()
	// 用指定的创世块创建一个kvLedger对象
	ledger, _ := provider.Create(gb1)
	defer ledger.Close()
	// 从账本中获取链信息——当前链高度、当前块哈希、上一个块哈希，进行比对
	bcInfo, _ := ledger.GetBlockchainInfo()
	testhelper.AssertEquals(t, bcInfo, &common.BlockchainInfo{
		Height: 1, CurrentBlockHash: gbHash, PreviousBlockHash: nil})
	// 创建一个交易模拟器
	simulator, _ := ledger.NewTxSimulator()
	// 将一笔提案的模拟执行结果写入读写集
	key1 := "__RLIST_ADD:__IDX___MODELDAT_insurance_ins_applyMedicalDetailst_,0"
	simulator.SetState("ns1", key1, []byte("value1"))
	simulator.Done()
	// 将读写集序列化为[]byte
	simRes, _ := simulator.GetTxSimulationResults()
	// 根据序列化后的读写集，先构造三笔交易，再将这三笔交易塞入envelopes中，最后把envelopes结合header和metadata组装为下一个块
	block1 := bg1.NextBlock([][]byte{simRes})
	// 向账本添加这个块
	ledger.Commit(block1)
	ledger.IndexState()
}
