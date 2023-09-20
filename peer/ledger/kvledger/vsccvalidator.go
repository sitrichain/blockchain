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

package kvledger

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/common/sysccprovider"
	ledgerUtil "github.com/rongzer/blockchain/peer/ledger/util"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// implementation of Validator interface, keeps
// reference to the ledger to enable tx simulation
// and execution of vscc
type vscctxValidator struct {
	ccprovider  ccprovider.ChaincodeProvider
	sccprovider sysccprovider.SystemChaincodeProvider
}

// generateCCKey generates a unique identifier for chaincode in specific chain
func (v *vscctxValidator) generateCCKey(ccName, chainID string) string {
	return fmt.Sprintf("%s/%s", ccName, chainID)
}

// invalidTXsForUpgradeCC invalid all txs that should be invalided because of chaincode upgrade txs
func (v *vscctxValidator) invalidTXsForUpgradeCC(txsChaincodeNames map[int]*sysccprovider.ChaincodeInstance, txsUpgradedChaincodes map[int]*sysccprovider.ChaincodeInstance, txsfltr ledgerUtil.TxValidationFlags) ledgerUtil.TxValidationFlags {
	if len(txsUpgradedChaincodes) == 0 {
		return txsfltr
	}

	// Invalid former cc upgrade txs if there're two or more txs upgrade the same cc
	finalValidUpgradeTXs := make(map[string]int)
	upgradedChaincodes := make(map[string]*sysccprovider.ChaincodeInstance)
	for tIdx, cc := range txsUpgradedChaincodes {
		if cc == nil {
			continue
		}
		upgradedCCKey := v.generateCCKey(cc.ChaincodeName, cc.ChainID)

		if finalIdx, exist := finalValidUpgradeTXs[upgradedCCKey]; !exist {
			finalValidUpgradeTXs[upgradedCCKey] = tIdx
			upgradedChaincodes[upgradedCCKey] = cc
		} else if finalIdx < tIdx {
			log.Logger.Infof("Invalid transaction with index %d: chaincode was upgraded by latter tx", finalIdx)
			txsfltr.SetFlag(finalIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)

			// record latter cc upgrade tx info
			finalValidUpgradeTXs[upgradedCCKey] = tIdx
			upgradedChaincodes[upgradedCCKey] = cc
		} else {
			log.Logger.Infof("Invalid transaction with index %d: chaincode was upgraded by latter tx", tIdx)
			txsfltr.SetFlag(tIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
		}
	}

	// invalid txs which invoke the upgraded chaincodes
	for tIdx, cc := range txsChaincodeNames {
		if cc == nil {
			continue
		}
		ccKey := v.generateCCKey(cc.ChaincodeName, cc.ChainID)
		if _, exist := upgradedChaincodes[ccKey]; exist {
			if txsfltr.IsValid(tIdx) {
				log.Logger.Infof("Invalid transaction with index %d: chaincode was upgraded in the same block", tIdx)
				txsfltr.SetFlag(tIdx, peer.TxValidationCode_CHAINCODE_VERSION_CONFLICT)
			}
		}
	}

	return txsfltr
}

func (v *vscctxValidator) getTxCCInstance(payload *common.Payload) (invokeCCIns, upgradeCCIns *sysccprovider.ChaincodeInstance, err error) {
	// This is duplicated unpacking work, but make test easier.
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return nil, nil, err
	}

	// Chain ID
	chainID := chdr.ChannelId // it is guaranteed to be an existing channel by now

	// ChaincodeID
	hdrExt, err := utils.GetChaincodeHeaderExtension(payload.Header)
	if err != nil {
		return nil, nil, err
	}
	invokeCC := hdrExt.ChaincodeId
	invokeIns := &sysccprovider.ChaincodeInstance{ChainID: chainID, ChaincodeName: invokeCC.Name, ChaincodeVersion: invokeCC.Version}

	// Transaction
	tx, err := utils.GetTransaction(payload.Data)
	if err != nil {
		log.Logger.Errorf("GetTransaction failed: %s", err)
		return invokeIns, nil, nil
	}

	// ChaincodeActionPayload
	p, err := utils.GetChaincodeActionPayload(tx.Actions[0].Payload)
	if err != nil {
		log.Logger.Errorf("GetChaincodeActionPayload failed: %s", err)
		return invokeIns, nil, nil
	}

	// ChaincodeProposalPayload
	cpp, err := utils.GetChaincodeProposalPayload(p.ChaincodeProposalPayload)
	if err != nil {
		log.Logger.Errorf("GetChaincodeProposalPayload failed: %s", err)
		return invokeIns, nil, nil
	}

	// ChaincodeInvocationSpec
	cis := &peer.ChaincodeInvocationSpec{}
	if err := cis.Unmarshal(cpp.Input); err != nil {
		log.Logger.Errorf("GetChaincodeInvokeSpec failed: %s", err)
		return invokeIns, nil, nil
	}

	if invokeCC.Name == "lscc" {
		if string(cis.ChaincodeSpec.Input.Args[0]) == "upgrade" {
			upgradeIns, err := v.getUpgradeTxInstance(chainID, cis.ChaincodeSpec.Input.Args[2])
			if err != nil {
				return invokeIns, nil, nil
			}
			return invokeIns, upgradeIns, nil
		}
	}

	return invokeIns, nil, nil
}

func (v *vscctxValidator) getUpgradeTxInstance(chainID string, cdsBytes []byte) (*sysccprovider.ChaincodeInstance, error) {
	cds, err := utils.GetChaincodeDeploymentSpec(cdsBytes)
	if err != nil {
		return nil, err
	}

	return &sysccprovider.ChaincodeInstance{
		ChainID:          chainID,
		ChaincodeName:    cds.ChaincodeSpec.ChaincodeId.Name,
		ChaincodeVersion: cds.ChaincodeSpec.ChaincodeId.Version,
	}, nil
}

// validateEndorserTransaction validates the payload of a
// transaction assuming its type is ENDORSER_TRANSACTION
func validateEndorserTransaction(data []byte, hdr *common.Header) (*peer.ChaincodeAction, error) {
	log.Logger.Debugf("validateEndorserTransaction starts for data %p, header %s", data, hdr)

	// check for nil argument
	if data == nil || hdr == nil {
		return nil, fmt.Errorf("Nil arguments")
	}

	tx, err := utils.GetTransaction(data)
	if err != nil {
		return nil, err
	}

	// check for nil argument
	if tx == nil {
		return nil, fmt.Errorf("Nil transaction")
	}

	// hlf version 1 only supports a single action per transaction
	if len(tx.Actions) != 1 {
		return nil, fmt.Errorf("Only one action per transaction is supported (tx contains %d)", len(tx.Actions))
	}

	log.Logger.Debugf("validateEndorserTransaction info: there are %d actions", len(tx.Actions))
	act := tx.Actions[0]
	if act == nil {
		return nil, fmt.Errorf("Nil action")
	}

	// if the type is ENDORSER_TRANSACTION we unmarshal a SignatureHeader
	sHdr, err := utils.GetSignatureHeader(act.Header)
	if err != nil {
		return nil, err
	}

	// validate the SignatureHeader - here we actually only
	// care about the nonce since the creator is in the outer header
	err = validateSignatureHeader(sHdr)
	if err != nil {
		return nil, err
	}

	log.Logger.Debugf("validateEndorserTransaction info: signature header is valid")

	// if the type is ENDORSER_TRANSACTION we unmarshal a ChaincodeActionPayload
	ccActionPayload, err := utils.GetChaincodeActionPayload(act.Payload)
	if err != nil {
		return nil, err
	}

	// extract the proposal response payload
	prp, err := utils.GetProposalResponsePayload(ccActionPayload.Action.ProposalResponsePayload)
	if err != nil {
		return nil, err
	}

	// build the original header by stitching together
	// the common ChannelHeader and the per-action SignatureHeader
	hdrOrig := &common.Header{ChannelHeader: hdr.ChannelHeader, SignatureHeader: act.Header}

	// compute proposalHash
	pHash, err := utils.GetProposalHash2(hdrOrig, ccActionPayload.ChaincodeProposalPayload)
	if err != nil {
		return nil, err
	}

	// ensure that the proposal hash matches
	if bytes.Compare(pHash, prp.ProposalHash) != 0 {
		return nil, fmt.Errorf("proposal hash does not match")
	}
	respPayload, err := utils.GetChaincodeAction(prp.Extension)
	if err != nil {
		return nil, fmt.Errorf("GetChaincodeAction error %s", err)
	}
	return respPayload, nil
}

// checks for a valid SignatureHeader
func validateSignatureHeader(sHdr *common.SignatureHeader) error {
	// check for nil argument
	if sHdr == nil {
		return errors.New("Nil SignatureHeader provided")
	}

	// ensure that there is a nonce
	if sHdr.Nonce == nil || len(sHdr.Nonce) == 0 {
		return errors.New("Invalid nonce specified in the header")
	}

	// ensure that there is a creator
	if sHdr.Creator == nil || len(sHdr.Creator) == 0 {
		return errors.New("Invalid creator specified in the header")
	}

	return nil
}
