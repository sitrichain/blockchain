/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package qscc

import (
	"fmt"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/dispatcher"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/policy"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// LedgerQuerier implements the ledger query functions, including:
// - GetChainInfo returns BlockchainInfo
// - GetBlockByNumber returns a block
// - GetBlockByHash returns a block
// - GetTransactionByID returns a transaction
type LedgerQuerier struct {
	policyChecker policy.PolicyChecker
}

type ValidateResult struct {
	//result *rwsetutil.TxRwSet
	code pb.TxValidationCode
	err  error
}

// These are function names from Invoke first parameter
const (
	GetChainInfo       string = "GetChainInfo"
	GetBlockByNumber   string = "GetBlockByNumber"
	GetBlockByHash     string = "GetBlockByHash"
	GetTransactionByID string = "GetTransactionByID"
	GetBlockByTxID     string = "GetBlockByTxID"
	GetAttachById      string = "GetAttachById"
)

// Init is called once per chain when the chain is created.
// This allows the chaincode to initialize any variables on the ledger prior
// to any transaction execution on the chain.
func (e *LedgerQuerier) Init(_ shim.ChaincodeStubInterface) pb.Response {
	log.Logger.Debug("Init QSCC")

	// Init policy checker for access control
	e.policyChecker = policy.NewPolicyChecker(
		chain.NewChannelPolicyManagerGetter(),
		mgmt.GetLocalMSPOfPeer(),
		mgmt.NewLocalMSPPrincipalGetter(),
	)

	return shim.Success(nil)
}

// Invoke is called with args[0] contains the query function name, args[1]
// contains the chain ID, which is temporary for now until it is part of stub.
// Each function requires additional parameters as described below:
// # GetChainInfo: Return a BlockchainInfo object marshalled in bytes
// # GetBlockByNumber: Return the block specified by block number in args[2]
// # GetBlockByHash: Return the block specified by block hash in args[2]
// #  GetTransactionByID: Return the transaction specified by ID in args[2]
func (e *LedgerQuerier) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()

	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}
	fname := string(args[0])
	cid := string(args[1])

	//log.Logger.Infof("fname : %s cid : %s", fname, cid)

	if fname == "GetPeerList" {
		buf, err := dispatcher.GetRaftDistributor().MarshalPeerList(cid)
		if err != nil {
			log.Logger.Errorf("Rejecting getPeerList message because %s", err)
			return shim.Error("get peer List error")
		}
		return shim.Success(buf)
	}

	if fname != GetChainInfo && len(args) < 3 {
		return shim.Error(fmt.Sprintf("missing 3rd argument for %s", fname))
	}
	var targetLedger ledger.PeerLedger
	targetLedger = chain.GetLedger(cid)
	if targetLedger == nil {
		return shim.Error(fmt.Sprintf("Invalid chain ID, %s", cid))
	}

	//log.Logger.Infof("Invoke function: %s on chain: %s", fname, cid)

	// Handle ACL:
	// 1. get the signed proposal
	_, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed getting signed proposal from stub, %s: %s", cid, err))
	}

	/*查询不需要权限判断
	// 2. check the channel reader policy
	if err = e.policyChecker.CheckPolicy(cid, policies.ChannelApplicationReaders, sp); err != nil {
		return shim.Error(fmt.Sprintf("Authorization request failed %s: %s", cid, err))
	}
	*/

	if fname == "invoke" {
		fname = "Invoke"
	}

	switch fname {
	case GetTransactionByID:
		return getTransactionByID(targetLedger, args[2], cid)
	case GetBlockByNumber:
		return getBlockByNumber(targetLedger, args[2], cid)
	case GetBlockByHash:
		return getBlockByHash(targetLedger, args[2], cid)
	case GetChainInfo:
		return getChainInfo(targetLedger, cid)
	case GetBlockByTxID:
		return getBlockByTxID(targetLedger, args[2], cid)
	case GetAttachById:
		return getAttachById(targetLedger, string(args[2]))
	case "GetBlockStatistics":
		blockIndex, err := strconv.ParseInt(string(args[2]), 10, 64)
		if err != nil {
			blockIndex = -1
		}

		curBlockStatisticsBuf := []byte("")
		if blockIndex >= 0 {
			curBlockStatisticsBuf, _ = stub.GetState("BlockStatistics_" + string(blockIndex))
		} else {
			curBlockStatisticsBuf, _ = stub.GetState("curBlockStatistics")
		}
		if curBlockStatisticsBuf == nil {
			curBlockStatisticsBuf = []byte("")
		}
		return shim.Success(curBlockStatisticsBuf)
	case "QueryStatisticsTransaction":
		if len(args) < 4 {
			return shim.Error(fmt.Sprintf("missing 4rd argument for %s", fname))
		}

		if len(args[2]) < 32 {
			return shim.Error(fmt.Sprintf("txId params is error %s", fname))
		}

		return queryStatisticsTransaction(stub, string(args[2]), string(args[3]))

	case "Invoke":
		if string(args[2]) == "GetTxValidateReturn" {
			pr := getTransactionByID(targetLedger, args[3], cid)

			if pr.Status != shim.OK {
				log.Logger.Error("get transaction err")
				return pr
			}

			//rwset, code, err := targetLedger.GetTxValidationResult(pr.Payload, true, nil)
			//result := ValidateResult{result: rwset, code: code, err: err}
			//
			//resultBytes, err := jsoniter.Marshal(result)
			//if err != nil {
			//	log.Logger.Errorf("%s", err)
			//}

			return shim.Success(nil)
		} else if string(args[2]) == "getErrorTxChain" {
			errChain, err := getErrorTxChain(targetLedger, string(args[3])+","+string(args[4]))
			if err != nil {
				log.Logger.Errorf("getErrorTxChain err : %s", err)
				return shim.Error(fmt.Sprintf("getErrorTxChain err : %s", err))
			}

			errChainBytes, err := jsoniter.Marshal(errChain)
			if err != nil {
				log.Logger.Errorf(" json.Marshal(errChain) err : %s", err)
				return shim.Error(fmt.Sprintf(" json.Marshal(errChain) err : %s", err))
			}

			return shim.Success(errChainBytes)
		} else if string(args[2]) == "getLatestErrorTxChain" {
			errLatestId, err := getLatestErrorTxChain(targetLedger)
			if err != nil {
				log.Logger.Errorf("getErrorTxChain err : %s", err)
				return shim.Error(fmt.Sprintf("getErrorTxChain err : %s", err))
			}

			log.Logger.Infof("errLatestId : %s", errLatestId)
			return shim.Success([]byte(errLatestId))
		} else if string(args[2]) == "getErrorTxChainList" {

			pts, err := getErrorTxChainList(targetLedger, cid)
			if err != nil {
				return shim.Error(fmt.Sprintf("getErrorTxChainList err : %s", err))
			}

			ptsBuf, err := jsoniter.Marshal(pts)
			if err != nil {
				return shim.Error(fmt.Sprintf("json.Marshal(pts) err : %s", err))
			}

			return shim.Success(ptsBuf)
		} else if string(args[2]) == GetTransactionByID {
			return getTransactionByID(targetLedger, args[3], cid)
		}
	}

	return shim.Error(fmt.Sprintf("Requested function %s not found.", fname))
}

func queryStatisticsTransaction(stub shim.ChaincodeStubInterface, txId, transStaticKey string) pb.Response {
	lisTxId := txId
	rNum := 0
	for {
		preTxBuf, _ := stub.GetState(txId + "-" + transStaticKey)
		if preTxBuf != nil {
			txId = string(preTxBuf)
			lisTxId += "," + txId
		}
		rNum++
		if rNum >= 100 {
			break
		}
	}
	return shim.Success([]byte(lisTxId))
}

func getTransactionByID(vledger ledger.PeerLedger, tid []byte, _ string) pb.Response {
	if tid == nil {
		return shim.Error("Transaction ID must not be nil.")
	}

	tidStr := strings.TrimSpace(string(tid))
	processedTran, err := vledger.GetTransactionByID(tidStr)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get proccess from orderer %v, error %s", processedTran, err))
	}

	bytes, err := utils.Marshal(processedTran)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByNumber(vledger ledger.PeerLedger, number []byte, chanId string) pb.Response {
	if number == nil {
		return shim.Error("Block number must not be nil.")
	}
	bnum, err := strconv.ParseUint(string(number), 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to parse block number with error %s", err))
	}
	block, err := vledger.GetBlockByNumber(bnum)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block by number: %v, error: %v", bnum, err))
	}

	bytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByHash(vledger ledger.PeerLedger, hash []byte, cid string) pb.Response {
	if hash == nil {
		return shim.Error("Block hash must not be nil.")
	}
	block, err := vledger.GetBlockByHash(hash)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block by hash: %v, error: %v", string(hash), err))
	}

	bytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getChainInfo(vledger ledger.PeerLedger, cid string) pb.Response {
	binfo, err := vledger.GetBlockchainInfo()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get chain info, error %s", err))
	}
	bytes, err := utils.Marshal(binfo)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getBlockByTxID(vledger ledger.PeerLedger, rawTxID []byte, cid string) pb.Response {
	txID := string(rawTxID)
	block, err := vledger.GetBlockByTxID(txID)

	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get block for txID %s, error %s", txID, err))
	}

	bytes, err := utils.Marshal(block)

	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytes)
}

func getAttachById(vledger ledger.PeerLedger, key string) pb.Response {
	attach, err := vledger.GetAttachById(key)
	if err != nil {
		return shim.Error(err.Error())
	}

	log.Logger.Info("qscc key: ", key, " attach :", attach)

	return shim.Success([]byte(attach))
}

// 获取错误交易链表
func getErrorTxChain(vledger ledger.PeerLedger, txId string) (*pb.ErrorTxChain, error) {
	return vledger.GetErrorTxChain(txId)
}

func getLatestErrorTxChain(vledger ledger.PeerLedger) (string, error) {
	return vledger.GetLatestErrorTxChain()
}

func getErrorTxChainList(vledger ledger.PeerLedger, _ string) ([]*pb.ProcessedTransaction, error) {

	txId, err := getLatestErrorTxChain(vledger)
	if err != nil {
		return nil, err
	}

	//log.Logger.Infof("txid : %s err : %s", txId, err)

	pts := make([]*pb.ProcessedTransaction, 0)

	for {
		errTxChan, err := getErrorTxChain(vledger, txId)
		if err != nil {
			return nil, err
		}

		//log.Logger.Infof("errTxChan current : %s errTxChan previous : %s", errTxChan.CurrentTx, errTxChan.PreviousTx)

		tran, err := vledger.GetTransactionByID(errTxChan.CurrentTx)
		if err != nil {
			return nil, err
		}

		//tran.TransactionEnvelope = nil
		pts = append(pts, tran)

		if errTxChan.PreviousTx == "" || errTxChan.PreviousTx == errTxChan.CurrentTx {

			break
		}

		txId = errTxChan.PreviousTx
	}

	return pts, nil
}
