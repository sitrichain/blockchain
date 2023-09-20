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

package utils

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/util"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/peer/chaincode/platforms"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
)

// GetChaincodeInvocationSpec get the ChaincodeInvocationSpec from the proposal
func GetChaincodeInvocationSpec(prop *peer.Proposal) (*peer.ChaincodeInvocationSpec, error) {
	if prop == nil {
		return nil, fmt.Errorf("Proposal is nil")
	}
	_, err := GetHeader(prop.Header)
	if err != nil {
		return nil, err
	}
	ccPropPayload := &peer.ChaincodeProposalPayload{}
	err = ccPropPayload.Unmarshal(prop.Payload)
	if err != nil {
		return nil, err
	}
	cis := &peer.ChaincodeInvocationSpec{}
	err = cis.Unmarshal(ccPropPayload.Input)
	return cis, err
}

// GetChaincodeProposalContext returns creator and transient
func GetChaincodeProposalContext(prop *peer.Proposal) ([]byte, map[string][]byte, error) {
	if prop == nil {
		return nil, nil, fmt.Errorf("Proposal is nil")
	}
	if len(prop.Header) == 0 {
		return nil, nil, fmt.Errorf("Proposal's header is nil")
	}
	if len(prop.Payload) == 0 {
		return nil, nil, fmt.Errorf("Proposal's payload is nil")
	}

	//// get back the header
	hdr, err := GetHeader(prop.Header)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not extract the header from the proposal: %s", err)
	}
	if hdr == nil {
		return nil, nil, fmt.Errorf("Unmarshalled header is nil")
	}

	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not extract the channel header from the proposal: %s", err)
	}

	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION &&
		common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		return nil, nil, fmt.Errorf("Invalid proposal type expected ENDORSER_TRANSACTION or CONFIG. Was: %d", chdr.Type)
	}

	shdr, err := GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not extract the signature header from the proposal: %s", err)
	}

	ccPropPayload := &peer.ChaincodeProposalPayload{}
	err = ccPropPayload.Unmarshal(prop.Payload)
	if err != nil {
		return nil, nil, err
	}

	return shdr.Creator, ccPropPayload.TransientMap, nil
}

// GetHeader Get Header from bytes
func GetHeader(bytes []byte) (*common.Header, error) {
	hdr := &common.Header{}
	err := hdr.Unmarshal(bytes)
	return hdr, err
}

// GetNonce returns the nonce used in Proposal
func GetNonce(prop *peer.Proposal) ([]byte, error) {
	if prop == nil {
		return nil, fmt.Errorf("Proposal is nil")
	}
	// get back the header
	hdr, err := GetHeader(prop.Header)
	if err != nil {
		return nil, fmt.Errorf("Could not extract the header from the proposal: %s", err)
	}

	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, fmt.Errorf("Could not extract the channel header from the proposal: %s", err)
	}

	if common.HeaderType(chdr.Type) != common.HeaderType_ENDORSER_TRANSACTION &&
		common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		return nil, fmt.Errorf("Invalid proposal type expected ENDORSER_TRANSACTION or CONFIG. Was: %d", chdr.Type)
	}

	shdr, err := GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, fmt.Errorf("Could not extract the signature header from the proposal: %s", err)
	}

	if hdr.SignatureHeader == nil {
		return nil, errors.New("Invalid signature header. It must be different from nil.")
	}

	return shdr.Nonce, nil
}


// GetChaincodeHeaderExtension get chaincode header extension given header
func GetChaincodeHeaderExtension(hdr *common.Header) (*peer.ChaincodeHeaderExtension, error) {
	chdr, err := UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, err
	}

	chaincodeHdrExt := &peer.ChaincodeHeaderExtension{}
	err = chaincodeHdrExt.Unmarshal(chdr.Extension)
	return chaincodeHdrExt, err
}

// GetProposalResponse given proposal in bytes
func GetProposalResponse(prBytes []byte) (*peer.ProposalResponse, error) {
	proposalResponse := &peer.ProposalResponse{}
	err := proposalResponse.Unmarshal(prBytes)
	return proposalResponse, err
}

// GetChaincodeDeploymentSpec returns a ChaincodeDeploymentSpec given args
func GetChaincodeDeploymentSpec(code []byte) (*peer.ChaincodeDeploymentSpec, error) {
	cds := &peer.ChaincodeDeploymentSpec{}
	err := cds.Unmarshal(code)
	if err != nil {
		return nil, err
	}

	// FAB-2122: Validate the CDS according to platform specific requirements
	platform, err := platforms.Find(cds.ChaincodeSpec.Type)
	if err != nil {
		return nil, err
	}

	err = platform.ValidateDeploymentSpec(cds)
	return cds, err
}

// GetChaincodeAction gets the ChaincodeAction given chaicnode action bytes
func GetChaincodeAction(caBytes []byte) (*peer.ChaincodeAction, error) {
	chaincodeAction := &peer.ChaincodeAction{}
	err := chaincodeAction.Unmarshal(caBytes)
	return chaincodeAction, err
}

// GetResponse gets the Response given response bytes
func GetResponse(resBytes []byte) (*peer.Response, error) {
	response := &peer.Response{}
	err := response.Unmarshal(resBytes)
	return response, err
}

// GetProposalResponsePayload gets the proposal response payload
func GetProposalResponsePayload(prpBytes []byte) (*peer.ProposalResponsePayload, error) {
	prp := &peer.ProposalResponsePayload{}
	err := prp.Unmarshal(prpBytes)
	return prp, err
}

// GetProposal returns a Proposal message from its bytes
func GetProposal(propBytes []byte) (*peer.Proposal, error) {
	prop := &peer.Proposal{}
	err := prop.Unmarshal(propBytes)
	return prop, err
}

// GetPayload Get Payload from Envelope message
func GetPayload(e *common.Envelope) (*common.Payload, error) {
	payload := &common.Payload{}
	err := payload.Unmarshal(e.Payload)
	return payload, err
}

// GetTransaction Get Transaction from bytes
func GetTransaction(txBytes []byte) (*peer.Transaction, error) {
	tx := &peer.Transaction{}
	err := tx.Unmarshal(txBytes)
	return tx, err
}

// GetChaincodeActionPayload Get ChaincodeActionPayload from bytes
func GetChaincodeActionPayload(capBytes []byte) (*peer.ChaincodeActionPayload, error) {
	p := &peer.ChaincodeActionPayload{}
	err := p.Unmarshal(capBytes)
	return p, err
}

// GetChaincodeProposalPayload Get ChaincodeProposalPayload from bytes
func GetChaincodeProposalPayload(bytes []byte) (*peer.ChaincodeProposalPayload, error) {
	cpp := &peer.ChaincodeProposalPayload{}
	err := cpp.Unmarshal(bytes)
	return cpp, err
}

// GetSignatureHeader Get SignatureHeader from bytes
func GetSignatureHeader(bytes []byte) (*common.SignatureHeader, error) {
	sh := &common.SignatureHeader{}
	err := sh.Unmarshal(bytes)
	return sh, err
}

// GetBytesProposalResponsePayload gets proposal response payload
func GetBytesProposalResponsePayload(hash []byte, response *peer.Response, result []byte, event []byte, ccid *peer.ChaincodeID) ([]byte, error) {
	cAct := &peer.ChaincodeAction{Events: event, Results: result, Response: response, ChaincodeId: ccid}
	cActBytes, err := cAct.Marshal()
	if err != nil {
		return nil, err
	}

	prp := &peer.ProposalResponsePayload{Extension: cActBytes, ProposalHash: hash}
	prpBytes, err := prp.Marshal()
	return prpBytes, err
}

// GetBytesChaincodeProposalPayload gets the chaincode proposal payload
func GetBytesChaincodeProposalPayload(cpp *peer.ChaincodeProposalPayload) ([]byte, error) {
	cppBytes, err := cpp.Marshal()
	return cppBytes, err
}

// GetBytesResponse gets the bytes of Response
func GetBytesResponse(res *peer.Response) ([]byte, error) {
	resBytes, err := res.Marshal()
	return resBytes, err
}

// GetBytesChaincodeEvent gets the bytes of ChaincodeEvent
func GetBytesChaincodeEvent(event *peer.ChaincodeEvent) ([]byte, error) {
	eventBytes, err := event.Marshal()
	return eventBytes, err
}

// GetBytesChaincodeActionPayload get the bytes of ChaincodeActionPayload from the message
func GetBytesChaincodeActionPayload(cap *peer.ChaincodeActionPayload) ([]byte, error) {
	capBytes, err := cap.Marshal()
	return capBytes, err
}

// GetBytesProposalResponse gets propoal bytes response
func GetBytesProposalResponse(pr *peer.ProposalResponse) ([]byte, error) {
	respBytes, err := pr.Marshal()
	return respBytes, err
}

// GetBytesProposal returns the bytes of a proposal message
func GetBytesProposal(prop *peer.Proposal) ([]byte, error) {
	propBytes, err := proto.Marshal(prop)
	return propBytes, err
}

// GetBytesSignatureHeader get the bytes of SignatureHeader from the message
func GetBytesSignatureHeader(hdr *common.SignatureHeader) ([]byte, error) {
	bytes, err := hdr.Marshal()
	return bytes, err
}

// GetBytesTransaction get the bytes of Transaction from the message
func GetBytesTransaction(tx *peer.Transaction) ([]byte, error) {
	bytes, err := tx.Marshal()
	return bytes, err
}

// GetBytesPayload get the bytes of Payload from the message
func GetBytesPayload(payl *common.Payload) ([]byte, error) {
	bytes, err := payl.Marshal()
	return bytes, err
}

// GetBytesEnvelope get the bytes of Envelope from the message
func GetBytesEnvelope(env *common.Envelope) ([]byte, error) {
	bytes, err := env.Marshal()
	return bytes, err
}

// GetActionFromEnvelope extracts a ChaincodeAction message from a serialized Envelope
func GetActionFromEnvelope(envBytes []byte) (*peer.ChaincodeAction, error) {
	env, err := GetEnvelopeFromBlock(envBytes)
	if err != nil {
		return nil, err
	}

	payl, err := GetPayload(env)
	if err != nil {
		return nil, err
	}

	tx, err := GetTransaction(payl.Data)
	if err != nil {
		return nil, err
	}

	if len(tx.Actions) == 0 {
		return nil, fmt.Errorf("At least one TransactionAction is required")
	}

	_, respPayload, err := GetPayloads(tx.Actions[0])
	return respPayload, err
}

// ComputeProposalTxID computes TxID as the Hash computed
// over the concatenation of nonce and creator.
func ComputeProposalTxID(nonce, creator []byte) (string, error) {
	// TODO: Get the Hash function to be used from
	// channel configuration
	digest, err := factory.GetDefault().Hash(
		append(nonce, creator...),
		&bccsp.SHA256Opts{})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

// CheckProposalTxID checks that txid is equal to the Hash computed
// over the concatenation of nonce and creator.
func CheckProposalTxID(txid string, nonce, creator []byte) error {
	computedTxID, err := ComputeProposalTxID(nonce, creator)
	if err != nil {
		return fmt.Errorf("Failed computing target TXID for comparison [%s]", err)
	}

	if txid != computedTxID {
		return fmt.Errorf("Transaction is not valid. Got [%s], expected [%s]", txid, computedTxID)
	}

	return nil
}

// ComputeProposalBinding computes the binding of a proposal
func ComputeProposalBinding(proposal *peer.Proposal) ([]byte, error) {
	if proposal == nil {
		return nil, fmt.Errorf("Porposal is nil")
	}
	if len(proposal.Header) == 0 {
		return nil, fmt.Errorf("Proposal's Header is nil")
	}

	h, err := GetHeader(proposal.Header)
	if err != nil {
		return nil, err
	}

	chdr, err := UnmarshalChannelHeader(h.ChannelHeader)
	if err != nil {
		return nil, err
	}
	shdr, err := GetSignatureHeader(h.SignatureHeader)
	if err != nil {
		return nil, err
	}

	return computeProposalBindingInternal(shdr.Nonce, shdr.Creator, chdr.Epoch)
}

func computeProposalBindingInternal(nonce, creator []byte, epoch uint64) ([]byte, error) {
	epochBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochBytes, epoch)

	// TODO: add to genesis block the hash function used for the binding computation.
	return factory.GetDefault().Hash(
		append(append(nonce, creator...), epochBytes...),
		&bccsp.SHA256Opts{})
}

// CreateChaincodeProposal creates a proposal from given input.
// It returns the proposal and the transaction id associated to the proposal
func CreateChaincodeProposal(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte) (*peer.Proposal, string, error) {
	return CreateChaincodeProposalWithTransient(typ, chainID, cis, creator, nil)
}

// CreateProposalFromCIS returns a proposal given a serialized identity and a ChaincodeInvocationSpec
func CreateProposalFromCIS(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte) (*peer.Proposal, string, error) {
	return CreateChaincodeProposal(typ, chainID, cis, creator)
}

// CreateChaincodeProposalWithTransient creates a proposal from given input
// It returns the proposal and the transaction id associated to the proposal
func CreateChaincodeProposalWithTransient(typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, creator []byte, transientMap map[string][]byte) (*peer.Proposal, string, error) {
	// generate a random nonce
	nonce, err := crypto.GetRandomNonce()
	if err != nil {
		return nil, "", err
	}

	// compute txid
	txid, err := ComputeProposalTxID(nonce, creator)
	if err != nil {
		return nil, "", err
	}

	return CreateChaincodeProposalWithTxIDNonceAndTransient(txid, typ, chainID, cis, nonce, creator, transientMap)
}

// CreateChaincodeProposalWithTxIDNonceAndTransient creates a proposal from given input
func CreateChaincodeProposalWithTxIDNonceAndTransient(txid string, typ common.HeaderType, chainID string, cis *peer.ChaincodeInvocationSpec, nonce, creator []byte, transientMap map[string][]byte) (*peer.Proposal, string, error) {
	ccHdrExt := &peer.ChaincodeHeaderExtension{ChaincodeId: cis.ChaincodeSpec.ChaincodeId}
	ccHdrExtBytes, err := ccHdrExt.Marshal()
	if err != nil {
		return nil, "", err
	}

	cisBytes, err := cis.Marshal()
	if err != nil {
		return nil, "", err
	}

	ccPropPayload := &peer.ChaincodeProposalPayload{Input: cisBytes, TransientMap: transientMap}
	ccPropPayloadBytes, err := ccPropPayload.Marshal()
	if err != nil {
		return nil, "", err
	}

	// TODO: epoch is now set to zero. This must be changed once we
	// get a more appropriate mechanism to handle it in.
	var epoch uint64 = 0

	timestamp := util.CreateUtcTimestamp()

	hdr := &common.Header{ChannelHeader: MarshalOrPanic(&common.ChannelHeader{
		Type:      int32(typ),
		TxId:      txid,
		Timestamp: timestamp,
		ChannelId: chainID,
		Extension: ccHdrExtBytes,
		Epoch:     epoch}),
		SignatureHeader: MarshalOrPanic(&common.SignatureHeader{Nonce: nonce, Creator: creator})}

	hdrBytes, err := hdr.Marshal()
	if err != nil {
		return nil, "", err
	}

	return &peer.Proposal{Header: hdrBytes, Payload: ccPropPayloadBytes}, txid, nil
}
