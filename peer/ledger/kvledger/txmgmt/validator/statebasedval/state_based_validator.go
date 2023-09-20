package statebasedval

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/statedb"
	"github.com/rongzer/blockchain/peer/ledger/kvledger/txmgmt/version"
	"github.com/rongzer/blockchain/peer/ledger/util"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/ledger/rwset/kvrwset"
	"github.com/rongzer/blockchain/protos/peer"
	putils "github.com/rongzer/blockchain/protos/utils"
)

// Validator validates a tx against the latest committed state
// and preceding valid transactions with in the same block
type Validator struct {
	db statedb.VersionedDB
}

// NewValidator constructs StateValidator
func NewValidator(db statedb.VersionedDB) *Validator {
	return &Validator{db}
}

//validate endorser transaction
func (v *Validator) ValidateEndorserTX(envBytes []byte, doMVCCValidation bool, updates *statedb.UpdateBatch) (*rwsetutil.TxRwSet, peer.TxValidationCode, error) {
	//  extract actions from the envelope message
	respPayload, err := putils.GetActionFromEnvelope(envBytes)
	if err != nil {
		return nil, peer.TxValidationCode_NIL_TXACTION, nil
	}

	//preparation for extracting RWSet from transaction
	txRWSet := &rwsetutil.TxRwSet{}

	// Get the Result from the Action
	// and then Unmarshal it into a TxReadWriteSet using custom unmarshalling

	if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
		return nil, peer.TxValidationCode_INVALID_OTHER_REASON, nil
	}

	txResult := peer.TxValidationCode_VALID
	//mvccvalidation, may invalidate transaction
	if doMVCCValidation {
		if txResult, err = v.ValidateTx(txRWSet, updates); err != nil {
			return nil, txResult, err
		} else if txResult != peer.TxValidationCode_VALID {
			txRWSet = nil
		}
	}

	return txRWSet, txResult, err
}

// ValidateAndPrepareBatch implements method in Validator interface
func (v *Validator) ValidateAndPrepareBatch(block *common.Block, doMVCCValidation bool) (*statedb.UpdateBatch, error) {
	log.Logger.Debugf("New block arrived for validation:%#v, doMVCCValidation=%t", block, doMVCCValidation)
	readNum, writeNum := v.db.GetReadWriteNum()

	updates := statedb.NewUpdateBatch()
	log.Logger.Debugf("Validating a block with [%d] transactions", len(block.Data.Data))

	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])
	if len(txsFilter) == 0 {
		txsFilter = util.NewTxValidationFlags(len(block.Data.Data))
	}
	var counter = struct {
		sync.RWMutex // gard m
		m map[int]*rwsetutil.TxRwSet
	}{m: make(map[int]*rwsetutil.TxRwSet, 100)}
	// 多协程并行验证交易
	var wg sync.WaitGroup
	for txIndex, envBytes := range block.Data.Data {
		wg.Add(1)
		go func(txIndex int, envBytes []byte) {
			defer wg.Done()
			if txsFilter.IsInvalid(txIndex) {
				return
			}
			// mvcc验证
			txRWSet, txResult, err := v.ValidateEndorserTX(envBytes, doMVCCValidation, updates)
			if err != nil {
				txsFilter.SetFlag(txIndex, txResult)
				return
			}
			//txRWSet != nil => t is valid
			if txRWSet != nil {
				counter.Lock()
				counter.m[txIndex] = txRWSet
				counter.Unlock()
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_VALID)
			}
		}(txIndex, envBytes)
	}
	wg.Wait()

	for txIndex := range block.Data.Data {
		txRWSet := counter.m[txIndex]
		if txRWSet == nil {
			continue
		}
		if txResult, err := v.validateCalTx(txRWSet, updates); err != nil {
			return nil, err
		} else if txResult != peer.TxValidationCode_VALID {
			txsFilter.SetFlag(txIndex, txResult)
			continue
		}

		committingTxHeight := version.NewHeight(block.Header.Number, uint64(txIndex))
		addWriteSetToBatch(txRWSet, committingTxHeight, updates)
		delete(counter.m, txIndex)
	}
	counter.m = nil

	v.moveCalsToUpdate(updates)

	readNum1, writeNum1 := v.db.GetReadWriteNum()
	if readNum1 >= readNum {
		readNum1 = readNum1 - readNum
	}
	if writeNum1 >= writeNum {
		writeNum1 = writeNum1 - writeNum
	}
	log.Logger.Debugf("Validating [%d] block with [%d] transactions with %d updates,%d calupdat,%d read", block.GetHeader().Number, len(block.Data.Data), updates.UpdateNum, updates.CalNum, readNum1)
	//验证结束后，将txsFilter写入block的元数据BlockMetadataIndex_TRANSACTIONS_FILTER对应的位置
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter

	return updates, nil
}

func addWriteSetToBatch(txRWSet *rwsetutil.TxRwSet, txHeight *version.Height, batch *statedb.UpdateBatch) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace
		for _, kvWrite := range nsRWSet.KvRwSet.Writes {
			if kvWrite.IsDelete {
				batch.Delete(ns, kvWrite.Key, txHeight)
			} else {
				vStr := kvWrite.Key

				// 对值作增加计算
				if !strings.HasPrefix(vStr, "__CAL_") && !strings.HasPrefix(vStr, "__RLIST_") {
					if strings.HasPrefix(kvWrite.Key, "__RBCMODEL_") { //兼容老的数据模型
						key := "__RBC_MODEL_" + kvWrite.Key[11:]
						batch.Put(ns, key, kvWrite.Value, txHeight)
					} else {
						batch.Put(ns, kvWrite.Key, kvWrite.Value, txHeight)
					}
				}
			}
		}
	}
}

func (v *Validator) ValidateTx(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch) (peer.TxValidationCode, error) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace

		if valid, err := v.validateReadSet(ns, nsRWSet.KvRwSet.Reads, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
		if valid, err := v.validateRangeQueries(ns, nsRWSet.KvRwSet.RangeQueriesInfo, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}
	}
	return peer.TxValidationCode_VALID, nil
}

func (v *Validator) validateCalTx(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch) (peer.TxValidationCode, error) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace

		for _, kvRead := range nsRWSet.KvRwSet.Reads {
			if updates.Exists(ns, kvRead.Key) {
				return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
			}
		}

		// 验证交易前，先处理计算值
		if valid, err := v.calWriteSet(ns, nsRWSet.KvRwSet.Writes, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}

	}
	return peer.TxValidationCode_VALID, nil
}

type CalItem struct {
	Key int64
	Val *kvrwset.KVWrite
}

type CalSorter []CalItem

func (ms CalSorter) Len() int {
	return len(ms)
}

func (ms CalSorter) Less(i, j int) bool {
	return ms[i].Key < ms[j].Key // 按键排序
}

func (ms CalSorter) Swap(i, j int) {
	ms[i], ms[j] = ms[j], ms[i]
}

func (v *Validator) calWriteSet(ns string, kvWrites []*kvrwset.KVWrite, updates *statedb.UpdateBatch) (bool, error) {

	updates.GetOrCreateNsUpdates(ns)
	updates.GetOrCreateNsCals(ns)
	updates.GetOrCreateNsRLists(ns)
	mapUpdates := updates.GetUpdates(ns)
	mapCals := updates.GetCals(ns) // map[string]*VersionedValue
	//	mapRLists := updates.GetRLists(ns)
	vh, _ := v.db.GetLatestSavePoint()

	// 根据序列号进行排序。

	sortedAry := make(CalSorter, 0, len(kvWrites))

	// "__CAL_ADD:"+rListName+","+getSeq(),id

	for _, kvWrite := range kvWrites {
		vStr := kvWrite.Key

		// 对值作增加计算
		if strings.HasPrefix(vStr, "__CAL_") {
			vStr = vStr[6:]

			calParams := strings.Split(vStr[4:], ",")
			sSeq := calParams[1]

			nSeq, _ := strconv.ParseInt(sSeq, 10, 64)
			sortedAry = append(sortedAry, CalItem{nSeq, kvWrite})
		}
	}

	sort.Sort(sortedAry)

	for _, sortedKV := range sortedAry {
		kvWrite := sortedKV.Val // 按次序获取kvWrite元素。
		vStr := kvWrite.Key

		// 对值作增加计算
		if strings.HasPrefix(vStr, "__CAL_") {
			vStr = vStr[6:]
			calMethod := vStr[:3]

			calParams := strings.Split(vStr[4:], ",")
			rListName := calParams[0]
			sSeq := calParams[1]

			lParam, _ := strconv.ParseInt(string(kvWrite.Value), 10, 64)
			nSeq, _ := strconv.ParseInt(sSeq, 10, 64)

			existValue := mapCals[rListName]
			//string到int64

			if existValue == nil {
				existValue = mapUpdates[rListName]
				if existValue == nil {
					//读己有的值
					existValue, _ = v.db.GetState(ns, rListName)
				}
			}

			if existValue == nil {
				existValue = &statedb.VersionedValue{[]byte("0"), vh}
			}

			//计算
			lValue, err := strconv.ParseInt(string(existValue.Value), 10, 64)
			if err != nil {
				lValue = 0
			}

			if strings.EqualFold(calMethod, "ADD") {
				lValue = lValue + lParam
			}

			if strings.EqualFold(calMethod, "MUL") {
				lValue = lValue * lParam
			}

			log.Logger.Debugf("  CAL kvWrite seqence=%d %s value method:%s param:%s old:%s result:%d ns:%s rListName:%s",
				nSeq, kvWrite.Key, calMethod, lParam, string(existValue.Value), lValue, ns, rListName)

			//int64到string
			existValue.Value = []byte(strconv.FormatInt(lValue, 10))
			mapCals[rListName] = existValue
			delete(mapUpdates, rListName)
		}

	}

	for _, kvWrite := range kvWrites {
		//处理值的变理
		valueStr := string(kvWrite.Value)

		if strings.Contains(valueStr, "##RCOUNT##") {
			replaceStr := valueStr

			for {
				bIndex := strings.Index(replaceStr, "##RCOUNT##")
				if bIndex < 0 {
					break
				}
				//log.Logger.Infof("replaceStr %d %s", bIndex, replaceStr)

				replaceStr = replaceStr[bIndex+10:]
				eIndex := strings.Index(replaceStr, "##RCOUNT##")
				if eIndex <= 0 || eIndex > 64 {
					break
				}

				replaceVal := replaceStr[0:eIndex]

				//查找replaceVal对应的值，先从计算参数中找
				existValue := mapCals[replaceVal]
				if existValue == nil {
					//读己有的值
					existValue, _ = v.db.GetState(ns, replaceVal)
					if existValue == nil {
						existValue = &statedb.VersionedValue{[]byte("0"), vh}
					}
					mapCals[replaceVal] = existValue
				}

				//计算
				lValue, err := strconv.ParseInt(string(existValue.Value), 10, 64)
				if err != nil {
					lValue = 0
				}

				replaceStr = replaceStr[eIndex+10:]

				//替换valueStr中的值
				valueStr = strings.Replace(valueStr, "##RCOUNT##"+replaceVal+"##RCOUNT##", strconv.FormatInt(lValue, 10), -1)

				log.Logger.Infof("replaceStr Result %s", replaceStr)
				log.Logger.Infof("replaceValue %s", valueStr)

			}
			kvWrite.Value = []byte(valueStr)
		}

	}

	return true, nil
}

func (v *Validator) moveCalsToUpdate(updates *statedb.UpdateBatch) {
	namespaces := updates.GetCaledNamespaces()
	updates.CalNum = 0
	updates.RListNum = 0
	for _, ns := range namespaces {
		updates.GetOrCreateNsUpdates(ns)
		updates.GetOrCreateNsCals(ns)
		mapUpdates := updates.GetUpdates(ns)
		mapCals := updates.GetCals(ns)

		for k, vv := range mapCals {
			if mapUpdates[k] == nil {
				mapUpdates[k] = vv
				updates.CalNum++
			}
			delete(mapCals, k)
		}
	}

	updates.UpdateNum = updates.GetUpdateSize()
}

func (v *Validator) validateReadSet(ns string, kvReads []*kvrwset.KVRead, updates *statedb.UpdateBatch) (bool, error) {
	for _, kvRead := range kvReads {
		if valid, err := v.validateKVRead(ns, kvRead, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVRead performs mvcc check for a key read during transaction simulation.
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *Validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, _ *statedb.UpdateBatch) (bool, error) {
	key := kvRead.Key
	if strings.HasPrefix(key, "__RBCMODEL_") { //兼容老的数据模型
		key = "__RBC_MODEL_" + key[11:]
	}

	versionedValue, err := v.db.GetState(ns, key)
	if err != nil {
		log.Logger.Warnf("Validate KV read error because get key [%s:%s] from state error. %s", ns, key, err)
		return false, nil
	}
	var committedVersion *version.Height
	if versionedValue != nil {
		committedVersion = versionedValue.Version
	}

	newversion := rwsetutil.NewVersion(kvRead.Version)
	if !version.AreSame(committedVersion, newversion) {
		log.Logger.Warnf("Validate KV read error because version mismatch for key [%s:%s]. Committed version = [%s], Version in readSet [%s]",
			ns, key, committedVersion, kvRead.Version)
		return false, nil
	}
	return true, nil
}

func (v *Validator) validateKVReadWriteIsError(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch) (*common.ValidateResultCode, error) {
	if true {
		return nil, nil
	}
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace
		kvReads := nsRWSet.KvRwSet.Reads

		for _, kvRead := range kvReads {
			versionedValue, err := v.db.GetState(ns, kvRead.Key)
			if err != nil {
				return nil, err
			}
			committedVersion := &version.Height{}
			if versionedValue != nil {
				committedVersion = versionedValue.Version
			}

			newversion := rwsetutil.NewVersion(kvRead.Version)
			if !version.AreSame(committedVersion, newversion) {
				log.Logger.Debugf("Version mismatch for key [%s:%s]. Committed version = [%s], Version in readSet [%s]",
					ns, kvRead.Key, committedVersion, kvRead.Version)

				oldHeight := &common.Height{committedVersion.BlockNum, committedVersion.TxNum}
				newHeight := &common.Height{newversion.BlockNum, newversion.TxNum}

				oldValidateCode := &common.VersionValue{versionedValue.Value, oldHeight}
				newValidateCode := &common.VersionValue{nil, newHeight}

				validateCode := &common.ValidateResultCode{oldValidateCode, newValidateCode}

				return validateCode, nil
			}
		}

		for _, kvRead := range nsRWSet.KvRwSet.Reads {
			if updates.Exists(ns, kvRead.Key) {
				updateValue := updates.Get(ns, kvRead.Key)

				oldHeight := &common.Height{kvRead.Version.BlockNum, kvRead.Version.TxNum}
				newHeight := &common.Height{updateValue.Version.BlockNum, updateValue.Version.TxNum}

				oldValidateCode := &common.VersionValue{nil, oldHeight}
				newValidateCode := &common.VersionValue{updateValue.Value, newHeight}

				validateCode := &common.ValidateResultCode{oldValidateCode, newValidateCode}

				return validateCode, nil
			}
		}
	}

	return nil, nil
}

func (v *Validator) validateRangeQueries(ns string, rangeQueriesInfo []*kvrwset.RangeQueryInfo, updates *statedb.UpdateBatch) (bool, error) {
	for _, rqi := range rangeQueriesInfo {
		if valid, err := v.validateRangeQuery(ns, rqi, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateRangeQuery performs a phatom read check i.e., it
// checks whether the results of the range query are still the same when executed on the
// statedb (latest state as of last committed block) + updates (prepared by the writes of preceding valid transactions
// in the current block and yet to be committed as part of group commit at the end of the validation of the block)
func (v *Validator) validateRangeQuery(ns string, rangeQueryInfo *kvrwset.RangeQueryInfo, updates *statedb.UpdateBatch) (bool, error) {
	log.Logger.Debugf("validateRangeQuery: ns=%s, rangeQueryInfo=%s", ns, rangeQueryInfo)

	// If during simulation, the caller had not exhausted the iterator so
	// rangeQueryInfo.EndKey is not actual endKey given by the caller in the range query
	// but rather it is the last key seen by the caller and hence the combinedItr should include the endKey in the results.
	includeEndKey := !rangeQueryInfo.ItrExhausted

	combinedItr, err := newCombinedIterator(v.db, updates,
		ns, rangeQueryInfo.StartKey, rangeQueryInfo.EndKey, includeEndKey)
	if err != nil {
		return false, err
	}
	defer combinedItr.Close()
	var validator rangeQueryValidator
	if rangeQueryInfo.GetReadsMerkleHashes() != nil {
		log.Logger.Debug(`Hashing results are present in the range query info hence, initiating hashing based validation`)
		validator = &rangeQueryHashValidator{}
	} else {
		log.Logger.Debug(`Hashing results are not present in the range query info hence, initiating raw KVReads based validation`)
		validator = &rangeQueryResultsValidator{}
	}
	if err := validator.init(rangeQueryInfo, combinedItr); err != nil {
		return false, err
	}
	return validator.validate()
}
