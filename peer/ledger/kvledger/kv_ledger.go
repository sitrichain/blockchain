/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

xxx

*/

package kvledger

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	commonledger "github.com/rongzer/blockchain/common/ledger"
	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/history/historydb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/txmgr"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/txmgr/lockbasedtxmgr"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
	"github.com/rongzer/blockchain/peer/ledger/ledgerconfig"
	"github.com/rongzer/blockchain/peer/ledger/util"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/ledger/rwset/kvrwset"
	"github.com/rongzer/blockchain/protos/peer"
	putils "github.com/rongzer/blockchain/protos/utils"
)

// KVLedger provides an implementation of `ledger.PeerLedger`.
// This implementation provides a key-value based data model
type kvLedger struct {
	ledgerID   string
	blockStore blkstorage.BlockStore
	txtmgmt    txmgr.TxMgr
	historyDB  historydb.HistoryDB
	db         statedb.VersionedDB
	indexPoint uint64
}

// NewKVLedger constructs new `KVLedger`
func newKVLedger(ledgerID string, blockStore blkstorage.BlockStore,
	versionedDB statedb.VersionedDB, historyDB historydb.HistoryDB) (*kvLedger, error) {

	log.Logger.Debugf("Creating KVLedger ledgerID=%s: ", ledgerID)

	//Initialize transaction manager using state database
	var txmgmt txmgr.TxMgr
	txmgmt = lockbasedtxmgr.NewLockBasedTxMgr(versionedDB)

	// Create a kvLedger for this chain/ledger, which encasulates the underlying
	// id store, blockstore, txmgr (state database), history database
	l := &kvLedger{ledgerID: ledgerID, blockStore: blockStore, txtmgmt: txmgmt, historyDB: historyDB, db: versionedDB}
	l.indexPoint = 0

	//Recover both state DB and history DB if they are out of sync with block storage
	if err := l.recoverDBs(); err != nil {
		panic(fmt.Errorf(`Error during state DB recovery:%s`, err))
	}

	return l, nil
}

//
func (l *kvLedger) GetLedgerID() string {
	return l.ledgerID
}

//
func (l *kvLedger) GetLedgerDB() statedb.VersionedDB {
	return l.db
}

//Recover the state database and history database (if exist)
//by recommitting last valid blocks
func (l *kvLedger) recoverDBs() error {
	log.Logger.Debugf("Entering recoverDB()")
	//If there is no block in blockstorage, nothing to recover.
	info, _ := l.blockStore.GetBlockchainInfo()
	if info.Height == 0 {
		log.Logger.Debug("Block storage is empty.")
		return nil
	}
	lastAvailableBlockNum := info.Height - 1
	recoverables := []recoverable{l.txtmgmt, l.historyDB}
	var recoverers []*recoverer
	for _, recoverable := range recoverables {
		recoverFlag, firstBlockNum, err := recoverable.ShouldRecover(lastAvailableBlockNum)
		if err != nil {
			return err
		}
		if recoverFlag {
			recoverers = append(recoverers, &recoverer{firstBlockNum, recoverable})
		}
	}
	if len(recoverers) == 0 {
		return nil
	}
	if len(recoverers) == 1 {
		return l.recommitLostBlocks(recoverers[0].firstBlockNum, lastAvailableBlockNum, recoverers[0].recoverable)
	}

	// both dbs need to be recovered
	if recoverers[0].firstBlockNum > recoverers[1].firstBlockNum {
		// swap (put the lagger db at 0 index)
		recoverers[0], recoverers[1] = recoverers[1], recoverers[0]
	}
	if recoverers[0].firstBlockNum != recoverers[1].firstBlockNum {
		// bring the lagger db equal to the other db
		if err := l.recommitLostBlocks(recoverers[0].firstBlockNum, recoverers[1].firstBlockNum-1,
			recoverers[0].recoverable); err != nil {
			return err
		}
	}
	// get both the db upto block storage
	return l.recommitLostBlocks(recoverers[1].firstBlockNum, lastAvailableBlockNum,
		recoverers[0].recoverable, recoverers[1].recoverable)

}

//recommitLostBlocks retrieves blocks in specified range and commit the write set to either
//state DB or history DB or both
func (l *kvLedger) recommitLostBlocks(firstBlockNum uint64, lastBlockNum uint64, recoverables ...recoverable) error {
	var err error
	var block *common.Block
	for blockNumber := firstBlockNum; blockNumber <= lastBlockNum; blockNumber++ {
		if block, err = l.GetBlockByNumber(blockNumber); err != nil {
			return err
		}
		for _, r := range recoverables {
			if err := r.CommitLostBlock(block); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetTransactionByID retrieves a transaction by id
func (l *kvLedger) GetTransactionByID(txID string) (*peer.ProcessedTransaction, error) {
	tranEnv, err := l.blockStore.RetrieveTxByID(txID)
	if err != nil {
		return nil, err
	}

	txVResult, err := l.blockStore.RetrieveTxValidationCodeByTxID(txID)

	if err != nil {
		return nil, err
	}

	processedTran := &peer.ProcessedTransaction{TransactionEnvelope: tranEnv, ValidationCode: int32(txVResult)}
	return processedTran, nil
}

// GetBlockchainInfo returns basic info about blockchain
func (l *kvLedger) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return l.blockStore.GetBlockchainInfo()
}

// GetBlockByNumber returns block at a given height
// blockNumber of  math.MaxUint64 will return last block
func (l *kvLedger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	return l.blockStore.RetrieveBlockByNumber(blockNumber)
}

// GetBlocksIterator returns an iterator that starts from `startBlockNumber`(inclusive).
// The iterator is a blocking iterator i.e., it blocks till the next block gets available in the ledger
// ResultsIterator contains type BlockHolder
func (l *kvLedger) GetBlocksIterator(startBlockNumber uint64) (commonledger.ResultsIterator, error) {
	return l.blockStore.RetrieveBlocks(startBlockNumber)
}

// GetBlockByHash returns a block given it's hash
func (l *kvLedger) GetBlockByHash(blockHash []byte) (*common.Block, error) {
	return l.blockStore.RetrieveBlockByHash(blockHash)
}

// GetBlockByTxID returns a block which contains a transaction
func (l *kvLedger) GetBlockByTxID(txID string) (*common.Block, error) {
	return l.blockStore.RetrieveBlockByTxID(txID)
}

func (l *kvLedger) GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	return l.blockStore.RetrieveTxValidationCodeByTxID(txID)
}

func (l *kvLedger) GetTxValidationResult(envBytes []byte, doMVCCValidation bool, updates *statedb.UpdateBatch) (*rwsetutil.TxRwSet, peer.TxValidationCode, error) {
	return l.txtmgmt.GetTxValidateResult(envBytes, doMVCCValidation, updates)
}

func (l *kvLedger) GetErrorTxChain(txID string) (*peer.ErrorTxChain, error) {
	return l.blockStore.RetrieveErrorTxChain(txID)
}

func (l *kvLedger) GetLatestErrorTxChain() (string, error) {
	return l.blockStore.RetrieveLatestErrorTxChain()
}

//Prune prunes the blocks/transactions that satisfy the given policy
func (l *kvLedger) Prune(_ commonledger.PrunePolicy) error {
	return errors.New("Not yet implemented")
}

// NewTxSimulator returns new `ledger.TxSimulator`
func (l *kvLedger) NewTxSimulator() (ledger.TxSimulator, error) {
	return l.txtmgmt.NewTxSimulator()
}

// NewQueryExecutor gives handle to a query executor.
// A client can obtain more than one 'QueryExecutor's for parallel execution.
// Any synchronization should be performed at the implementation level if required
func (l *kvLedger) NewQueryExecutor() (ledger.QueryExecutor, error) {
	return l.txtmgmt.NewQueryExecutor()
}

// NewHistoryQueryExecutor gives handle to a history query executor.
// A client can obtain more than one 'HistoryQueryExecutor's for parallel execution.
// Any synchronization should be performed at the implementation level if required
// Pass the ledger blockstore so that historical values can be looked up from the chain
func (l *kvLedger) NewHistoryQueryExecutor() (ledger.HistoryQueryExecutor, error) {
	return l.historyDB.NewHistoryQueryExecutor(l.blockStore)
}

// Commit commits the valid block (returned in the method RemoveInvalidTransactionsAndPrepare) and related state changes
func (l *kvLedger) Commit(block *common.Block) error {

	var err error
	blockNo := block.Header.Number

	log.Logger.Infof("Validate And Prepare block [%d]", blockNo)
	err = l.txtmgmt.ValidateAndPrepare(block, true)
	if err != nil {
		return err
	}

	chainSize := 3
	ch := make(chan int, chainSize)
	go func() {
		log.Logger.Debugf("Channel [%s]: Committing block [%d] to storage", l.ledgerID, blockNo)
		if err = l.blockStore.AddBlock(block); err != nil {
			ch <- -1
		} else {
			ch <- 1
		}
	}()

	go func() {
		log.Logger.Debugf("Channel [%s]: Committing block [%d] transactions to state database", l.ledgerID, blockNo)
		if err = l.txtmgmt.Commit(); err != nil {
			ch <- -1
			panic(fmt.Errorf(`Error during commit to txmgr:%s`, err))
		}
		ch <- 2
	}()

	go func() {
		// History database could be written in parallel with state and/or async as a future optimization
		if ledgerconfig.IsHistoryDBEnabled() {
			log.Logger.Debugf("Channel [%s]: Committing block [%d] transactions to history database", l.ledgerID, blockNo)
			if err := l.historyDB.Commit(block); err != nil {
				ch <- -1
				panic(fmt.Errorf(`Error during commit to history db:%s`, err))
			}
		}
		ch <- 3

	}()

	for i := 0; i < chainSize; i++ {
		out := <-ch
		if out == -1 {
			return fmt.Errorf("write block data has exception")
		}
	}
	log.Logger.Infof("Channel [%s]: Created block [%d] with %d transaction(s)", l.ledgerID, block.Header.Number, len(block.Data.Data))

	return nil
}

func (l *kvLedger) IndexState() int {
	if l.indexPoint < 1 {
		//从DB中查询
		existValue, _ := l.db.GetState("qscc", "__RLIST_INDEXPOINT")
		if existValue != nil && len(existValue.Value) > 1 {
			l.indexPoint = binary.BigEndian.Uint64(existValue.Value)
		}
	}

	info, _ := l.GetBlockchainInfo()
	beginNum := l.indexPoint
	//循环处理BlockIndex,最多处理100块
	endNum := info.Height
	endNum0 := info.Height

	if endNum <= beginNum {
		return 0
	}

	readNum, writeNum := l.db.GetReadWriteNum()

	pTime1 := time.Now().UnixNano() / 1000000

	updates := statedb.NewUpdateBatch()
	updates.GetOrCreateNsUpdates("qscc")
	var bNum uint64 = 5
	if endNum > beginNum+bNum {
		endNum = beginNum + bNum
	}

	if endNum > beginNum+100 {
		endNum = beginNum + 100
	}

	var tranNo uint64
	for i := beginNum; i < endNum; i++ {
		block, _ := l.GetBlockByNumber(i)
		if block == nil {
			log.Logger.Infof("Channel [%s]:Index %d block [%d-%d] has error", l.ledgerID, i, beginNum, endNum)
			continue
		}
		l.indexStateBlock(block, updates)
		tranNo = uint64(len(block.Data.Data) - 1)
	}

	vh := version.NewHeight(endNum, tranNo)
	pTime2 := time.Now().UnixNano() / 1000000

	l.moveCalsToUpdate(updates, vh)

	readNum1, writeNum1 := l.db.GetReadWriteNum()
	if readNum1 >= readNum {
		readNum1 = readNum1 - readNum
	} else {
		readNum1 = readNum1 + (1000000 - readNum)
	}

	if writeNum1 >= writeNum {
		writeNum1 = writeNum1 - writeNum
	} else {
		writeNum1 = writeNum1 + (1000000 - writeNum)
	}

	pTime3 := time.Now().UnixNano() / 1000000

	err := l.db.ApplyUpdates(updates, nil)
	if err != nil {
		log.Logger.Errorf("Channel [%s]:Index State has error %s", l.ledgerID, err)
	}
	pTime4 := time.Now().UnixNano() / 1000000
	//最后写增量标记
	updates1 := statedb.NewUpdateBatch()
	updates1.GetOrCreateNsUpdates("qscc")

	mapUpdates1 := updates1.GetUpdates("qscc")
	bEndNum := make([]byte, 8)
	binary.BigEndian.PutUint64(bEndNum, endNum)

	updateValue1 := &statedb.VersionedValue{bEndNum, vh}
	mapUpdates1["__RLIST_INDEXPOINT"] = updateValue1

	err = l.db.ApplyUpdates(updates1, nil)
	if err != nil {
		log.Logger.Errorf("Channel [%s]:Index State has error %s", l.ledgerID, err)
	}
	l.indexPoint = endNum
	log.Logger.Infof("[%s]:Index [%d-%d] with %d(%d) update %d read %d rTime %d wTime", l.ledgerID, beginNum, endNum-1, updates.UpdateNum, updates.RListNum, readNum1, pTime2-pTime1, pTime4-pTime3)
	info1, _ := l.GetBlockchainInfo()
	endNum1 := info1.Height

	return int(endNum1 - endNum0)
}

//计算
func (l *kvLedger) indexStateBlock(block *common.Block, updates *statedb.UpdateBatch) {

	if block.Metadata == nil || block.Metadata.Metadata == nil {
		return
	}
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

	blockNo := block.Header.Number
	var tranNo uint64
	for txIndex, envBytes := range block.Data.Data {

		//判断数据的验证状态
		tranNo++
		vh := version.NewHeight(blockNo, tranNo)

		env, err := putils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			return
		}

		payload, err := putils.GetPayload(env)
		if err != nil {
			return
		}

		chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return
		}

		txType := common.HeaderType(chdr.Type)

		if txType != common.HeaderType_ENDORSER_TRANSACTION {
			log.Logger.Debugf("Skipping mvcc validation for Block [%d] Transaction index [%d] because, the transaction type is [%s]",
				block.Header.Number, txIndex, txType)
			continue
		}
		respPayload, err := putils.GetActionFromEnvelope(envBytes)
		txRWSet := &rwsetutil.TxRwSet{}

		// Get the Result from the Action
		// and then Unmarshal it into a TxReadWriteSet using custom unmarshalling
		if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
			continue
		}

		if txsFilter.IsInvalid(txIndex) {

			// Skiping invalid transaction
			log.Logger.Warnf("Block [%d] Transaction index [%d] marked as invalid by committer. Reason code [%d]", block.Header.Number, txIndex, txsFilter.Flag(txIndex))

			continue
		}

		l.indexStateTX(txRWSet, updates, vh)

	}

}

func (l *kvLedger) indexStateTX(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch, vh *version.Height) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace

		// 验证交易前，先处理计算值
		l.calWriteSet(ns, nsRWSet.KvRwSet.Writes, updates, vh)
	}
}

type RListOptorItem struct {
	Key int64
	Val *kvrwset.KVWrite
}

type RListOptorSorter []RListOptorItem

func (ms RListOptorSorter) Len() int {
	return len(ms)
}

func (ms RListOptorSorter) Less(i, j int) bool {
	return ms[i].Key < ms[j].Key // 按键排序
}

func (ms RListOptorSorter) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}

func (l *kvLedger) calWriteSet(ns string, kvWrites []*kvrwset.KVWrite, updates *statedb.UpdateBatch, vh *version.Height) {

	updates.GetOrCreateNsRLists(ns)
	mapRLists := updates.GetRLists(ns)

	// "__RLIST_ADD:"+rListName+","+getSeq() <= id

	// 根据序列号进行排序。不能使用map，可能有多个RList，存在同一个seq的多个RList。

	sortedAry := make(RListOptorSorter, 0, len(kvWrites))

	for _, kvWrite := range kvWrites {
		vStr := kvWrite.Key

		if strings.HasPrefix(vStr, "__RLIST_") {
			vStr = vStr[8:]
			lisParams := strings.Split(vStr[4:], ",")
			sSeq := lisParams[1]

			nSeq, _ := strconv.ParseInt(sSeq, 10, 64)

			sortedAry = append(sortedAry, RListOptorItem{nSeq, kvWrite})
		}
	}

	sort.Sort(sortedAry)

	for _, sortedKV := range sortedAry {

		kvWrite := sortedKV.Val // 按次序获取kvWrite元素。

		vStr := kvWrite.Key

		//维护RList
		if strings.HasPrefix(vStr, "__RLIST_") {
			vStr = vStr[8:]
			lisMethod := vStr[:3]
			lisParams := strings.Split(vStr[4:], ",")
			rListName := lisParams[0]
			if strings.HasPrefix(rListName, "__RBCMODEL_") { //兼容老的数据模型
				rListName = "__RBC_MODEL_" + rListName[11:]
			}

			rList := mapRLists[rListName]

			if rList == nil {
				rList = statedb.NewRList(l.db, ns, rListName)
				mapRLists[rListName] = rList
				//cache.RListCache.Set(ns+"__RLIST__"+lisParams[0], rList, 0)
			}

			rList.SetRootDB(l.db)

			if strings.EqualFold(lisMethod, "ADD") {
				lisValues := strings.Split(string(kvWrite.Value), ",")

				//log.Logger.Infof("vStr : %s kvWrite.Value : %s", vStr, string(kvWrite.Value))

				if len(lisValues) == 2 {
					b, err := strconv.Atoi(lisValues[0])
					if err != nil {
						rList.AddId(vh, lisValues[1])
					} else {
						rList.AddIndexId(vh, b, lisValues[1])
					}

				} else {
					rValue := lisValues[0]
					if strings.HasPrefix(rValue, "__RBCMODEL_") { //兼容老的数据模型
						rValue = "__RBC_MODEL_" + rValue[11:]
					}
					rList.AddId(vh, rValue)
				}
			}

			//rList.Print(0)

			if strings.EqualFold(lisMethod, "DEL") {
				rValue := string(kvWrite.Value)
				if strings.HasPrefix(rValue, "__RBCMODEL_") { //兼容老的数据模型
					rValue = "__RBC_MODEL_" + rValue[11:]
				}

				rList.RemoveId(rValue)
			}

		}
	}
}

func (l *kvLedger) moveCalsToUpdate(updates *statedb.UpdateBatch, vh *version.Height) {
	updates.CalNum = 0
	updates.RListNum = 0

	//更新RList
	namespaces := updates.GetRListsNamespaces()
	for _, ns := range namespaces {
		updates.GetOrCreateNsUpdates(ns)
		updates.GetOrCreateNsRLists(ns)
		mapUpdates := updates.GetUpdates(ns)
		mapRlists := updates.GetRLists(ns)

		for k, vv := range mapRlists {
			rList := vv

			rList.SaveState(vh)

			putStub := rList.GetPutStub()

			for j, value := range putStub {
				if value != nil {
					mapUpdates[j] = value
				} else {
					updateValue := &statedb.VersionedValue{nil, vh}
					mapUpdates[j] = updateValue
				}

				delete(putStub, j)
				updates.RListNum++
			}

			rList.Reset()

			delete(mapRlists, k)
		}
	}

	updates.UpdateNum = updates.GetUpdateSize()
}

// Close closes `KVLedger`
func (l *kvLedger) Close() {
	l.blockStore.Shutdown()
	l.txtmgmt.Shutdown()
}
