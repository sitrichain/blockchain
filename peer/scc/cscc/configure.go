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
*/

// Package cscc chaincode configer provides functions to manage
// configuration transactions as the network is being reconfigured. The
// configuration transactions arrive from the ordering service to the committer
// who calls this chaincode. The chaincode also provides peer configuration
// services such as joining a chain or getting configuration data.
package cscc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/chaincode/shim"
	"github.com/rongzer/blockchain/peer/events/producer"
	"github.com/rongzer/blockchain/peer/policy"
	"github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// PeerConfiger implements the configuration handler for the peer. For every
// configuration transaction coming in from the ordering service, the
// committer calls this system chaincode to process the transaction.
type PeerConfiger struct {
	policyChecker policy.PolicyChecker
}

// These are function names from Invoke first parameter
const (
	JoinChain      string = "JoinChain"
	GetConfigBlock string = "GetConfigBlock"
	GetChannels    string = "GetChannels"
)

// Init is called once per chain when the chain is created.
// This allows the chaincode to initialize any variables on the ledger prior
// to any transaction execution on the chain.
func (e *PeerConfiger) Init(_ shim.ChaincodeStubInterface) pb.Response {
	log.Logger.Debug("Init CSCC")

	// Init policy checker for access control
	e.policyChecker = policy.NewPolicyChecker(
		chain.NewChannelPolicyManagerGetter(),
		mgmt.GetLocalMSPOfPeer(),
		mgmt.NewLocalMSPPrincipalGetter(),
	)

	return shim.Success(nil)
}

// Invoke is called for the following:
// # to process joining a chain (called by app as a transaction proposal)
// # to get the current configuration block (called by app)
// # to update the configuration block (called by commmitter)
// Peer calls this function with 2 arguments:
// # args[0] is the function name, which must be JoinChain, GetConfigBlock or
// UpdateConfigBlock
// # args[1] is a configuration Block if args[0] is JoinChain or
// UpdateConfigBlock; otherwise it is the chain id
// TODO: Improve the scc interface to avoid marshal/unmarshal args
func (e *PeerConfiger) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()

	if len(args) < 1 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}

	fname := string(args[0])

	if fname != GetChannels && len(args) < 2 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments, %d", len(args)))
	}

	log.Logger.Warnf("Invoke function: %s", fname)

	// Handle ACL:
	// 1. get the signed proposal
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed getting signed proposal from stub: [%s]", err))
	}

	switch fname {
	case JoinChain:
		if string(args[1]) == "" {
			return shim.Error("channel name cannot be nil")
		}
		// 第一个参数是：通道名
		cid := string(args[1])
		// 第二个参数是：新节点加入通道后是否参与该链的共识
		var participateConsensus bool
		err = json.Unmarshal(args[2], participateConsensus)
		if err != nil {
			log.Logger.Error("cannot unmarshal the second parameter into bool type")
			return shim.Error("cannot unmarshal the second parameter into bool type")
		}
		// 2. check local MSP Admins policy
		if err = e.policyChecker.CheckPolicyNoChannel(mgmt.Admins, sp); err != nil {
			return shim.Error(fmt.Sprintf("\"JoinChain\" request failed authorization check "+
				"for channel [%s]: [%s]", cid, err))
		}
		return joinChain(cid, participateConsensus)
	case GetConfigBlock:
		// 2. check the channel reader policy
		if err = e.policyChecker.CheckPolicy(string(args[1]), policies.ChannelApplicationReaders, sp); err != nil {
			return shim.Error(fmt.Sprintf("\"GetConfigBlock\" request failed authorization check for channel [%s]: [%s]", args[1], err))
		}
		return getConfigBlock(args[1])
	case GetChannels:
		// 2. check local MSP Members policy
		if err = e.policyChecker.CheckPolicyNoChannel(mgmt.Members, sp); err != nil {
			return shim.Error(fmt.Sprintf("\"GetChannels\" request failed authorization check: [%s]", err))
		}

		return getChannels()

	}
	return shim.Error(fmt.Sprintf("Requested function %s not found.", fname))
}

// validateConfigBlock validate configuration block to see whenever it's contains valid config transaction
func validateConfigBlock(block *common.Block) error {
	envelopeConfig, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to %s", err))
	}

	configEnv := &common.ConfigEnvelope{}
	_, err = utils.UnmarshalEnvelopeOfType(envelopeConfig, common.HeaderType_CONFIG, configEnv)
	if err != nil {
		return errors.New(fmt.Sprintf("Bad configuration envelope: %s", err))
	}

	if configEnv.Config == nil {
		return errors.New("Nil config envelope Config")
	}

	if configEnv.Config.ChannelGroup == nil {
		return errors.New("Nil channel group")
	}

	if configEnv.Config.ChannelGroup.Groups == nil {
		return errors.New("No channel configuration groups are available")
	}

	_, exists := configEnv.Config.ChannelGroup.Groups[config.ApplicationGroupKey]
	if !exists {
		return errors.New(fmt.Sprintf("Invalid configuration block, missing %s "+
			"configuration group", config.ApplicationGroupKey))
	}

	return nil
}

// joinChain will join the specified chain in the configuration block.
// Since it is the first block, it is the genesis block containing configuration
// for this chain, so we want to update the Chain object with this info
func joinChain(chainID string, participateConsensus bool) pb.Response {
	if _, created := chain.GetManager().GetChain(chainID); !created {
		replicator := cluster.TheReplicator
		// 拉块
		err := replicator.ReplicateSomeChain(chainID)
		if err != nil {
			return shim.Error("cannot pull blocks from other nodes")
		}
		// 再创建链
		m := chain.GetManager()
		err = m.InitCreateChainAccordingly(chainID, participateConsensus)
		if err != nil {
			log.Logger.Errorf("cannot create chain: %v, err is: %v", chainID, err)
			return shim.Error("cannot join chain for creating chain failed")
		}
		channel, ok := m.GetChain(chainID)
		if ok {
			channel.Start()
		}
	}
	// 初始化该链，即启动系统链码
	chain.InitChain(chainID)
	// 将创世块作为blockEvent发出去
	var block *common.Block
	ledger := chain.GetLedger(chainID)
	if ledger == nil {
		return shim.Error(fmt.Sprintf("cannot get ledger of %v", chainID))
	}
	block, err := ledger.GetBlockByNumber(0)
	if err != nil {
		return shim.Error("cannot get genesis block from ledger")
	}
	if err := producer.SendProducerBlockEvent(block); err != nil {
		return shim.Error("cannot sending block event")
	}

	return shim.Success(nil)
}

// Return the current configuration block for the specified chainID. If the
// peer doesn't belong to the chain, return error
func getConfigBlock(chainID []byte) pb.Response {
	if chainID == nil {
		return shim.Error("ChainID must not be nil.")
	}
	var block *common.Block
	block = chain.GetCurrConfigBlock(string(chainID))

	if block == nil {
		return shim.Error(fmt.Sprintf("Unknown chain ID, %s", string(chainID)))
	}
	blockBytes, err := utils.Marshal(block)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(blockBytes)
}

// getChannels returns information about all channels for this peer
func getChannels() pb.Response {
	var channelInfoArray []*pb.ChannelInfo
	channelInfoArray = chain.GetChannelsInfo()

	// add array with info about all channels for this peer
	cqr := &pb.ChannelQueryResponse{Channels: channelInfoArray}

	cqrbytes, err := cqr.Marshal()
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(cqrbytes)
}
