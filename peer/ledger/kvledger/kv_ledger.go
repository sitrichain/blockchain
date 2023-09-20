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
	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/cauthdsl"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/events/producer"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	commonledger "github.com/rongzer/blockchain/common/ledger"
	"github.com/rongzer/blockchain/common/ledger/blkstorage"
	"github.com/rongzer/blockchain/common/log"
	coreUtil "github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/common/sysccprovider"
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
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/ledger/rwset/kvrwset"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/peer"
	putils "github.com/rongzer/blockchain/protos/utils"
)

var newestSeekPosition = &ab.SeekPosition{
	Type: &ab.SeekPosition_Newest{
		&ab.SeekNewest{},
	},
}

var IsTheChaincodeSupportInitialized bool

// KVLedger provides an implementation of `ledger.PeerLedger`.
// This implementation provides a key-value based data model
type KvLedger struct {
	ledgerID   string
	BlockStore blkstorage.BlockStore
	txtmgmt    txmgr.TxMgr
	historyDB  historydb.HistoryDB
	db         statedb.VersionedDB
	indexPoint uint64
	vsccVal    *vscctxValidator
}

// NewKVLedger constructs new `KVLedger`
func newKVLedger(ledgerID string, blockStore blkstorage.BlockStore,
	versionedDB statedb.VersionedDB, historyDB historydb.HistoryDB) (*KvLedger, error) {

	log.Logger.Debugf("Creating KVLedger ledgerID=%s: ", ledgerID)

	//Initialize transaction manager using state database
	txmgmt, err := lockbasedtxmgr.NewLockBasedTxMgr(versionedDB)
	if err != nil {
		return nil, err
	}
	vsccVal := &vscctxValidator{ccprovider: ccprovider.GetChaincodeProvider(), sccprovider: sysccprovider.GetSystemChaincodeProvider()}
	// Create a KvLedger for this chain/ledger, which encasulates the underlying
	// id store, blockstore, txmgr (state database), history database
	l := &KvLedger{ledgerID: ledgerID, BlockStore: blockStore, txtmgmt: txmgmt, historyDB: historyDB, db: versionedDB, vsccVal: vsccVal}
	l.indexPoint = 0

	//Recover both state DB and history DB if they are out of sync with block storage
	if err := l.recoverDBs(); err != nil {
		return nil, fmt.Errorf(`Error during state DB recovery:%s`, err)
	}

	return l, nil
}

// GetLedgerID 获取账本ID
func (l *KvLedger) GetLedgerID() string {
	return l.ledgerID
}

//
func (l *KvLedger) GetLedgerDB() statedb.VersionedDB {
	return l.db
}

//Recover the state database and history database (if exist)
//by recommitting last valid blocks
func (l *KvLedger) recoverDBs() error {
	log.Logger.Debugf("Entering recoverDB()")
	//If there is no block in blockstorage, nothing to recover.
	info, _ := l.BlockStore.GetBlockchainInfo()
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
func (l *KvLedger) recommitLostBlocks(firstBlockNum uint64, lastBlockNum uint64, recoverables ...recoverable) error {
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
func (l *KvLedger) GetTransactionByID(txID string) (*peer.ProcessedTransaction, error) {
	tranEnv, err := l.BlockStore.RetrieveTxByID(txID)
	if err != nil {
		return nil, err
	}

	txVResult, err := l.BlockStore.RetrieveTxValidationCodeByTxID(txID)

	if err != nil {
		return nil, err
	}

	processedTran := &peer.ProcessedTransaction{TransactionEnvelope: tranEnv, ValidationCode: int32(txVResult)}
	return processedTran, nil
}

// GetBlockchainInfo returns basic info about blockchain
func (l *KvLedger) GetBlockchainInfo() (*common.BlockchainInfo, error) {
	return l.BlockStore.GetBlockchainInfo()
}

// GetBlockByNumber returns block at a given height
// blockNumber of  math.MaxUint64 will return last block
func (l *KvLedger) GetBlockByNumber(blockNumber uint64) (*common.Block, error) {
	return l.BlockStore.RetrieveBlockByNumber(blockNumber)
}

func (l *KvLedger) GetAttachById(key string) (string, error) {
	return l.BlockStore.GetAttach(key), nil
}

// GetBlocksIterator returns an iterator that starts from `startBlockNumber`(inclusive).
// The iterator is a blocking iterator i.e., it blocks till the next block gets available in the ledger
// ResultsIterator contains type BlockHolder
func (l *KvLedger) GetBlocksIterator(startBlockNumber uint64) (commonledger.ResultsIterator, error) {
	return l.BlockStore.RetrieveBlocks(startBlockNumber)
}

// GetBlockByHash returns a block given it's hash
func (l *KvLedger) GetBlockByHash(blockHash []byte) (*common.Block, error) {
	return l.BlockStore.RetrieveBlockByHash(blockHash)
}

// GetBlockByTxID returns a block which contains a transaction
func (l *KvLedger) GetBlockByTxID(txID string) (*common.Block, error) {
	return l.BlockStore.RetrieveBlockByTxID(txID)
}

func (l *KvLedger) GetTxValidationCodeByTxID(txID string) (peer.TxValidationCode, error) {
	return l.BlockStore.RetrieveTxValidationCodeByTxID(txID)
}

func (l *KvLedger) GetTxValidationResult(envBytes []byte, doMVCCValidation bool, updates *statedb.UpdateBatch) (*rwsetutil.TxRwSet, peer.TxValidationCode, error) {
	return l.txtmgmt.GetTxValidateResult(envBytes, doMVCCValidation, updates)
}

func (l *KvLedger) GetErrorTxChain(txID string) (*peer.ErrorTxChain, error) {
	return l.BlockStore.RetrieveErrorTxChain(txID)
}

func (l *KvLedger) GetLatestErrorTxChain() (string, error) {
	return l.BlockStore.RetrieveLatestErrorTxChain()
}

//Prune prunes the blocks/transactions that satisfy the given policy
func (l *KvLedger) Prune(_ commonledger.PrunePolicy) error {
	return errors.New("Not yet implemented")
}

// NewTxSimulator returns new `ledger.TxSimulator`
func (l *KvLedger) NewTxSimulator() (ledger.TxSimulator, error) {
	return l.txtmgmt.NewTxSimulator()
}

// NewQueryExecutor gives handle to a query executor.
// A client can obtain more than one 'QueryExecutor's for parallel execution.
// Any synchronization should be performed at the implementation level if required
func (l *KvLedger) NewQueryExecutor() (ledger.QueryExecutor, error) {
	return l.txtmgmt.NewQueryExecutor()
}

// NewHistoryQueryExecutor gives handle to a history query executor.
// A client can obtain more than one 'HistoryQueryExecutor's for parallel execution.
// Any synchronization should be performed at the implementation level if required
// Pass the ledger blockstore so that historical values can be looked up from the chain
func (l *KvLedger) NewHistoryQueryExecutor() (ledger.HistoryQueryExecutor, error) {
	return l.historyDB.NewHistoryQueryExecutor(l.BlockStore)
}

func (l *KvLedger) VSCCValidateTx(chaincodeAction *peer.ChaincodeAction, chdr *common.ChannelHeader, envBytes []byte) (error, peer.TxValidationCode) {
	/* obtain the list of namespaces we're writing stuff to;
	   at first, we establish a few facts about this invocation:
	   1) which namespaces does it write to?
	   2) does it write to LSCC's namespace?
	   3) does it write to any cc that cannot be invoked? */
	var wrNamespace []string
	writesToLSCC := false
	writesToNonInvokableSCC := false
	respPayload := chaincodeAction

	txRWSet := &rwsetutil.TxRwSet{}
	if err := txRWSet.FromProtoBytes(respPayload.Results); err != nil {
		return fmt.Errorf("txRWSet.FromProtoBytes failed, error %s", err), peer.TxValidationCode_BAD_RWSET
	}
	for _, ns := range txRWSet.NsRwSets {
		if len(ns.KvRwSet.Writes) > 0 {
			wrNamespace = append(wrNamespace, ns.NameSpace)

			if !writesToLSCC && ns.NameSpace == "lscc" {
				writesToLSCC = true
			}

			if !writesToNonInvokableSCC && l.vsccVal.sccprovider.IsSysCCAndNotInvokableCC2CC(ns.NameSpace) {
				writesToNonInvokableSCC = true
			}

			if !writesToNonInvokableSCC && l.vsccVal.sccprovider.IsSysCCAndNotInvokableExternal(ns.NameSpace) {
				writesToNonInvokableSCC = true
			}
		}
	}

	// get name and version of the cc we invoked
	ccID := respPayload.ChaincodeId.Name
	ccVer := respPayload.ChaincodeId.Version
	// sanity check on ccver
	if ccVer == "" {
		err := fmt.Errorf("invalid chaincode version")
		log.Logger.Errorf("%s", err)
		return err, peer.TxValidationCode_INVALID_OTHER_REASON
	}

	// we've gathered all the info required to proceed to validation;
	// validation will behave differently depending on the type of
	// chaincode (system vs. application)

	if !l.vsccVal.sccprovider.IsSysCC(ccID) {
		// if we're here, we know this is an invocation of an application chaincode;
		// first of all, we make sure that:
		// 1) we don't write to LSCC - an application chaincode is free to invoke LSCC
		//    for instance to get information about itself or another chaincode; however
		//    these legitimate invocations only ready from LSCC's namespace; currently
		//    only two functions of LSCC write to its namespace: deploy and upgrade and
		//    neither should be used by an application chaincode
		if writesToLSCC {
			return fmt.Errorf("Chaincode %s attempted to write to the namespace of LSCC", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}
		// 2) we don't write to the namespace of a chaincode that we cannot invoke - if
		//    the chaincode cannot be invoked in the first place, there's no legitimate
		//    way in which a transaction has a write set that writes to it; additionally
		//    we don't have any means of verifying whether the transaction had the rights
		//    to perform that write operation because in v1, system chaincodes do not have
		//    any endorsement policies to speak of. So if the chaincode can't be invoked
		//    it can't be written to by an invocation of an application chaincode
		if writesToNonInvokableSCC {
			return fmt.Errorf("Chaincode %s attempted to write to the namespace of a system chaincode that cannot be invoked", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}

		// validate *EACH* read write set according to its chaincode's endorsement policy
		for _, ns := range wrNamespace {
			// Get latest chaincode version, vscc and validate policy
			txcc, vscc, policy, err := l.GetInfoForValidate(chdr.TxId, chdr.ChannelId, ns)
			if err != nil {
				log.Logger.Errorf("GetInfoForValidate for txId = %s returned error %s", chdr.TxId, err)
				return err, peer.TxValidationCode_INVALID_OTHER_REASON
			}

			// if the namespace corresponds to the cc that was originally
			// invoked, we check that the version of the cc that was
			// invoked corresponds to the version that lscc has returned
			if ns == ccID && txcc.ChaincodeVersion != ccVer {
				err := fmt.Errorf("Chaincode %s:%s/%s didn't match %s:%s/%s in lscc", ccID, ccVer, chdr.ChannelId, txcc.ChaincodeName, txcc.ChaincodeVersion, chdr.ChannelId)
				log.Logger.Errorf(err.Error())
				return err, peer.TxValidationCode_EXPIRED_CHAINCODE
			}

			// do VSCC validation
			if err = l.VSCCValidateTxForCC(envBytes, chdr.TxId, chdr.ChannelId, vscc.ChaincodeName, vscc.ChaincodeVersion, policy); err != nil {
				return fmt.Errorf("VSCCValidateTxForCC failed for cc %s, error %s", ccID, err),
					peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE
			}
		}
	} else {
		// make sure that we can invoke this system chaincode - if the chaincode
		// cannot be invoked through a proposal to this peer, we have to drop the
		// transaction; if we didn't, we wouldn't know how to decide whether it's
		// valid or not because in v1, system chaincodes have no endorsement policy
		if l.vsccVal.sccprovider.IsSysCCAndNotInvokableExternal(ccID) {
			return fmt.Errorf("Committing an invocation of cc %s is illegal", ccID),
				peer.TxValidationCode_ILLEGAL_WRITESET
		}

		// Get latest chaincode version, vscc and validate policy
		_, vscc, policy, err := l.GetInfoForValidate(chdr.TxId, chdr.ChannelId, ccID)
		if err != nil {
			log.Logger.Errorf("GetInfoForValidate for txId = %s returned error %s", chdr.TxId, err)
			return err, peer.TxValidationCode_INVALID_OTHER_REASON
		}

		// validate the transaction as an invocation of this system chaincode;
		// vscc will have to do custom validation for this system chaincode
		// currently, VSCC does custom validation for LSCC only; if an hlf
		// user creates a new system chaincode which is invokable from the outside
		// they have to modify VSCC to provide appropriate validation
		if err = l.VSCCValidateTxForCC(envBytes, chdr.TxId, vscc.ChainID, vscc.ChaincodeName, vscc.ChaincodeVersion, policy); err != nil {
			return fmt.Errorf("VSCCValidateTxForCC failed for cc %s, error %s", ccID, err),
				peer.TxValidationCode_ENDORSEMENT_POLICY_FAILURE
		}
	}

	return nil, peer.TxValidationCode_VALID
}

// GetInfoForValidate gets the ChaincodeInstance(with latest version) of tx, vscc and policy from lscc
func (l *KvLedger) GetInfoForValidate(txid, chID, ccID string) (*sysccprovider.ChaincodeInstance, *sysccprovider.ChaincodeInstance, []byte, error) {
	cc := &sysccprovider.ChaincodeInstance{ChainID: chID}
	vscc := &sysccprovider.ChaincodeInstance{ChainID: chID}
	var policy []byte
	var err error
	if ccID != "lscc" && ccID != "rbccustomer" && ccID != "rbcapproval" && ccID != "rbcmodel" &&
		ccID != "rbctoken" && ccID != "simple" {
		// when we are validating any chaincode other than
		// LSCC, we need to ask LSCC to give us the name
		// of VSCC and of the policy that should be used

		// obtain name of the VSCC and the policy from LSCC
		cd, err := l.getCDataForCC(ccID)
		if err != nil {
			log.Logger.Errorf("Unable to get chaincode data from ledger for txid %s, due to %s", txid, err)
			return nil, nil, nil, err
		}
		cc.ChaincodeName = cd.Name
		cc.ChaincodeVersion = cd.Version
		vscc.ChaincodeName = cd.Vscc
		policy = cd.Policy
	} else {
		// when we are validating LSCC, we use the default
		// VSCC and a default policy that requires one signature
		// from any of the members of the channel
		cc.ChaincodeName = ccID
		cc.ChaincodeVersion = coreUtil.GetSysCCVersion()
		vscc.ChaincodeName = "vscc"
		p := cauthdsl.SignedByAnyMember([]string{conf.V.Peer.MSPID})
		policy, err = putils.Marshal(p)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Get vscc version
	vscc.ChaincodeVersion = coreUtil.GetSysCCVersion()

	return cc, vscc, policy, nil
}

func (l *KvLedger) VSCCValidateTxForCC(envBytes []byte, txid, chid, vsccName, vsccVer string, policy []byte) error {
	l.vsccVal.ccprovider.LockTxsim()
	ctxt, err := l.vsccVal.ccprovider.GetContext(l)
	txsim := l.vsccVal.ccprovider.GetTxsim()
	defer l.vsccVal.ccprovider.TxsimDone(txsim)
	l.vsccVal.ccprovider.UnLockTxsim()

	if err != nil {
		log.Logger.Errorf("Cannot obtain context for txid=%s, err %s", txid, err)
		return err
	}
	//defer v.ccprovider.ReleaseContext()

	// build arguments for VSCC invocation
	// args[0] - function name (not used now)
	// args[1] - serialized Envelope
	// args[2] - serialized policy
	args := [][]byte{[]byte(""), envBytes, policy}

	// get context to invoke VSCC
	vscctxid := coreUtil.GenerateUUID()
	cccid := l.vsccVal.ccprovider.GetCCContext(chid, vsccName, vsccVer, vscctxid, true, nil, nil)

	// invoke VSCC
	log.Logger.Debug("Invoking VSCC txid", txid, "chaindID", chid)
	res, _, err := l.vsccVal.ccprovider.ExecuteChaincode(ctxt, cccid, args)
	if err != nil {
		log.Logger.Errorf("Invoke VSCC failed for transaction txid=%s, error %s", txid, err)
		return err
	}
	if res.Status != shim.OK {
		log.Logger.Errorf("VSCC check failed for transaction txid=%s, error %s", txid, res.Message)
		return fmt.Errorf("%s", res.Message)
	}

	return nil
}

func (l *KvLedger) getCDataForCC(ccid string) (*ccprovider.ChaincodeData, error) {
	if l == nil {
		return nil, fmt.Errorf("nil ledger instance")
	}

	qe, err := l.NewQueryExecutor()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve QueryExecutor, error %s", err)
	}
	defer qe.Done()

	bytes, err := qe.GetState("lscc", ccid)
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve state for chaincode %s, error %s", ccid, err)
	}

	if bytes == nil {
		return nil, fmt.Errorf("lscc's state for [%s] not found.", ccid)
	}

	cd := &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(bytes, cd)
	if err != nil {
		return nil, fmt.Errorf("Unmarshalling ChaincodeQueryResponse failed, error %s", err)
	}

	if cd.Vscc == "" {
		return nil, fmt.Errorf("lscc's state for [%s] is invalid, vscc field must be set.", ccid)
	}

	if len(cd.Policy) == 0 {
		return nil, fmt.Errorf("lscc's state for [%s] is invalid, policy field must be set.", ccid)
	}

	return cd, err
}

// ValidateVscc 使用vscc验证块中交易
func (l *KvLedger) ValidateVscc(block *common.Block) {
	txsFilter := util.NewTxValidationFlags(len(block.Data.Data))
	var mapLock = struct{ sync.RWMutex }{}
	// txsChaincodeNames records all the invoked chaincodes by tx in a block
	txsChaincodeNames := make(map[int]*sysccprovider.ChaincodeInstance)
	// upgradedChaincodes records all the chaincodes that are upgrded in a block
	txsUpgradedChaincodes := make(map[int]*sysccprovider.ChaincodeInstance)
	// 多协程并行验证交易
	var wg sync.WaitGroup
	for txIndex, envBytes := range block.Data.Data {
		wg.Add(1)
		go func(txIndex int, envBytes []byte) {
			defer wg.Done()
			// 反序列化出chdr
			env, err := putils.GetEnvelopeFromBlock(envBytes)
			if err != nil {
				log.Logger.Errorf("Error getting Envelope from block(%s)", err)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				return
			}
			payload, err := putils.GetPayload(env)
			if err != nil {
				log.Logger.Errorf("Error getting tx from Envelope(%s)", err)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				return
			}
			chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
			if err != nil {
				log.Logger.Errorf("Error getting channel header from tx(%s)", err)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				return
			}
			// HeaderType_CONFIG类型和HeaderType_ORDERER_TRANSACTION类型的交易不进行验证
			txType := common.HeaderType(chdr.Type)
			if txType != common.HeaderType_ENDORSER_TRANSACTION {
				log.Logger.Infof("Skipping vscc and mvcc validation for Block [%d] Transaction index [%d] because, the transaction type is [%s]",
					block.Header.Number, txIndex, txType)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_NO_NEED_TO_VALID)
				return
			}
			// 验证背书交易的基本面是否通过，得到该笔交易的chaincodeAction，用于接下来的vscc验证
			chaincodeAction, err := validateEndorserTransaction(payload.Data, payload.Header)
			if err != nil {
				log.Logger.Errorf("validateEndorserTransaction returns err %s", err)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_ENDORSER_TRANSACTION)
				return
			}
			// 在账本中检查该笔背书交易是否重复
			txID := chdr.TxId
			if _, err := l.GetTransactionByID(txID); err == nil {
				log.Logger.Error("Duplicate transaction found, ", txID, ", skipping")
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_DUPLICATE_TXID)
				return
			}
			// vscc验证
			log.Logger.Debug("Validating transaction vscc tx validate")
			err, cde := l.VSCCValidateTx(chaincodeAction, chdr, envBytes)
			if err != nil {
				txID := txID
				log.Logger.Errorf("VSCCValidateTx for transaction txId = %s returned error %s", txID, err)
				txsFilter.SetFlag(txIndex, cde)
				return
			}
			invokeCC, upgradeCC, err := l.vsccVal.getTxCCInstance(payload)
			if err != nil {
				log.Logger.Errorf("Get chaincode instance from transaction txId = %s returned error %s", txID, err)
				txsFilter.SetFlag(txIndex, peer.TxValidationCode_INVALID_OTHER_REASON)
				return
			}
			mapLock.Lock()
			txsChaincodeNames[txIndex] = invokeCC
			if upgradeCC != nil {
				log.Logger.Infof("Find chaincode upgrade transaction for chaincode %s on chain %s with new version %s", upgradeCC.ChaincodeName, upgradeCC.ChainID, upgradeCC.ChaincodeVersion)
				txsUpgradedChaincodes[txIndex] = upgradeCC
			}
			mapLock.Unlock()
			txsFilter.SetFlag(txIndex, peer.TxValidationCode_VALID)
		}(txIndex, envBytes)
	}
	wg.Wait()
	txsFilter = l.vsccVal.invalidTXsForUpgradeCC(txsChaincodeNames, txsUpgradedChaincodes, txsFilter)
	//验证结束后，将txsFilter写入block的元数据BlockMetadataIndex_TRANSACTIONS_FILTER对应的位置
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
}

// Commit commits the valid block (returned in the method RemoveInvalidTransactionsAndPrepare) and related state changes
func (l *KvLedger) Commit(block *common.Block) error {
	var err error
	blockNo := block.Header.Number
	log.Logger.Debugf("Validate And Prepare block [%d]", blockNo)
	// 若theChaincodeSupport尚未初始化，则不进行交易的vscc验证——这种场景适用于新加入节点从bootstrap节点拉块时
	if IsTheChaincodeSupportInitialized {
		l.ValidateVscc(block)
	}
	// 再进行交易的mvcc验证
	err = l.txtmgmt.ValidateAndPrepare(block, true)
	if err != nil {
		return err
	}
	chainSize := 3
	ch := make(chan int, chainSize)
	go func() {
		log.Logger.Debugf("Channel [%s]: Committing block [%d] to storage", l.ledgerID, blockNo)
		if err = l.BlockStore.AddBlock(block); err != nil {
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

func (l *KvLedger) IndexState() int {
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
func (l *KvLedger) indexStateBlock(block *common.Block, updates *statedb.UpdateBatch) {

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
		if err != nil {
			return
		}

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

func (l *KvLedger) indexStateTX(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch, vh *version.Height) {
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

func (l *KvLedger) calWriteSet(ns string, kvWrites []*kvrwset.KVWrite, updates *statedb.UpdateBatch, vh *version.Height) {

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

func (l *KvLedger) moveCalsToUpdate(updates *statedb.UpdateBatch, vh *version.Height) {
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
func (l *KvLedger) Close() {
	l.BlockStore.Shutdown()
	l.txtmgmt.Shutdown()
}

// 带有signal的账本，用于orderer
type SignaledLedger struct {
	*KvLedger
	Signal chan struct{}
}

// GetBlock is a utility method for retrieving a single block
func (l *SignaledLedger) GetBlock(index uint64) *cb.Block {
	iterator, _ := l.Iterator(&ab.SeekPosition{
		Type: &ab.SeekPosition_Specified{
			Specified: &ab.SeekSpecified{Number: index},
		},
	})
	if iterator == nil {
		return nil
	}
	defer iterator.Close()
	block, status, err := iterator.Next()
	if status != cb.Status_SUCCESS || err != nil {
		log.Logger.Errorf("cannot get block with index %v, err is: %v", index, err)
		return nil
	}
	return block
}

// Iterator 获取指定起始位置的迭代器, 同时返回开始的块序号
func (l *SignaledLedger) Iterator(startPosition *ab.SeekPosition) (Iterator, uint64) {
	switch start := startPosition.Type.(type) {
	case *ab.SeekPosition_Oldest:
		return &ledgerIterator{ledger: l, blockNumber: 0}, 0
	case *ab.SeekPosition_Newest:
		info, err := l.KvLedger.GetBlockchainInfo()
		if err != nil {
			log.Logger.Panic(err)
		}
		newestBlockNumber := info.Height - 1
		return &ledgerIterator{ledger: l, blockNumber: newestBlockNumber}, newestBlockNumber
	case *ab.SeekPosition_Specified:
		height := l.Height()
		if start.Specified.Number > height {
			return &notFoundErrorIterator{}, 0
		}
		return &ledgerIterator{ledger: l, blockNumber: start.Specified.Number}, start.Specified.Number
	default:
		return &notFoundErrorIterator{}, 0
	}
}

// Height 获取账本的块高度
func (l *SignaledLedger) Height() uint64 {
	info, err := l.KvLedger.GetBlockchainInfo()
	if err != nil {
		log.Logger.Panic(err)
	}
	return info.Height
}

// Append 先提交block（写入文件账本+更新世界状态），再发布事件通知
func (l *SignaledLedger) Append(block *cb.Block) error {
	// 提交block
	err := l.Commit(block)
	if err != nil {
		return err
	}
	// send block event *after* the block has been committed
	if err := producer.SendProducerBlockEvent(block); err != nil {
		log.Logger.Errorf("Error publishing block %d, because: %v", block.Header.Number, err)
	}
	close(l.Signal)
	l.Signal = make(chan struct{})

	return nil
}

// GetBlockchainInfo 获取账本的基本信息
func (l *SignaledLedger) GetBlockchainInfo() (*cb.BlockchainInfo, error) {
	return l.BlockStore.GetBlockchainInfo()
}

// GetBlockByNumber 获取指定高度的块, 传入math.MaxUint64可获取最新的块
func (l *SignaledLedger) GetBlockByNumber(blockNumber uint64) (*cb.Block, error) {
	return l.BlockStore.RetrieveBlockByNumber(blockNumber)
}

// GetBlockByHash 获取指定哈希的块
func (l *SignaledLedger) GetBlockByHash(blockHash []byte) (*cb.Block, error) {
	return l.BlockStore.RetrieveBlockByHash(blockHash)
}

// GetBlockByTxID 获取包含指定交易的块
func (l *SignaledLedger) GetBlockByTxID(txID string) (*cb.Block, error) {
	return l.BlockStore.RetrieveBlockByTxID(txID)
}

// GetTxByID 获取指定交易ID的交易
func (l *SignaledLedger) GetTxByID(txID string) (*cb.Envelope, error) {
	return l.BlockStore.RetrieveTxByID(txID)
}

// GetAttach 获取附件数据
func (l *SignaledLedger) GetAttach(attachKey string) string {
	return l.BlockStore.GetAttach(attachKey)
}

// CreateNextBlock 使用传入的交易列表创建新块
func (l *SignaledLedger) CreateGenesisBlock(messages []*cb.Envelope, metadata *cb.Metadata) (*cb.Block, error) {
	var nextBlockNumber uint64
	var previousBlockHash []byte

	data := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	var err error
	for i, msg := range messages {
		data.Data[i], err = msg.Marshal()
		if err != nil {
			return nil, err
		}
	}
	// Attachs不参与Hash
	data1 := &cb.BlockData{
		Data: make([][]byte, len(messages)),
	}

	for i, msg := range messages {
		msg1 := &cb.Envelope{Payload: msg.Payload, Signature: msg.Signature}
		data1.Data[i], err = msg1.Marshal()
		if err != nil {
			return nil, err
		}
	}

	block := cb.NewBlock(nextBlockNumber, previousBlockHash)
	block.Header.DataHash = data1.Hash()
	block.Data = data
	block.Metadata.Metadata[cb.BlockMetadataIndex_ORDERER] = putils.MarshalOrPanic(metadata)

	return block, nil
}
