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

package validator

import (
	"errors"
	"fmt"

	"github.com/rongzer/blockchain/common/log"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// validateChaincodeProposalMessage checks the validity of a Proposal message of type CHAINCODE
func validateChaincodeProposalMessage(prop *pb.Proposal, hdr *common.Header) (*pb.ChaincodeHeaderExtension, error) {
	if prop == nil || hdr == nil {
		return nil, errors.New("Nil arguments")
	}

	log.Logger.Debugf("validateChaincodeProposalMessage starts for proposal %p, header %p", prop, hdr)

	// 4) based on the header type (assuming it's CHAINCODE), look at the extensions
	chaincodeHdrExt, err := utils.GetChaincodeHeaderExtension(hdr)
	if err != nil {
		return nil, errors.New("Invalid header extension for type CHAINCODE")
	}

	log.Logger.Debugf("validateChaincodeProposalMessage info: header extension references chaincode %s", chaincodeHdrExt.ChaincodeId)

	//    - ensure that the chaincodeID is correct (?)
	// TODO: should we even do this? If so, using which interface?

	//    - ensure that the visibility field has some value we understand
	// currently the blockchain only supports full visibility: this means that
	// there are no restrictions on which parts of the proposal payload will
	// be visible in the final transaction; this default approach requires
	// no additional instructions in the PayloadVisibility field which is
	// therefore expected to be nil; however the blockchain may be extended to
	// encode more elaborate visibility mechanisms that shall be encoded in
	// this field (and handled appropriately by the peer)
	if chaincodeHdrExt.PayloadVisibility != nil {
		return nil, errors.New("Invalid payload visibility field")
	}

	return chaincodeHdrExt, nil
}

// ValidateProposalMessage checks the validity of a SignedProposal message
// this function returns Header and ChaincodeHeaderExtension messages since they
// have been unmarshalled and validated
func ValidateProposalMessage(signedProp *pb.SignedProposal) (*pb.Proposal, *common.ChannelHeader, *common.SignatureHeader, *pb.ChaincodeHeaderExtension, error) {
	if signedProp == nil {
		return nil, nil, nil, nil, errors.New("Nil arguments")
	}

	log.Logger.Debugf("ValidateProposalMessage starts for signed proposal %p", signedProp)

	// extract the Proposal message from signedProp
	prop, err := utils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// 1) look at the ProposalHeader
	hdr, err := utils.GetHeader(prop.Header)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// validate the header
	chdr, shdr, err := validateCommonHeader(hdr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// validate the signature
	err = CheckSignatureFromCreator(shdr.Creator, signedProp.Signature, signedProp.ProposalBytes, chdr.ChannelId)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Verify that the transaction ID has been computed properly.
	// This check is needed to ensure that the lookup into the ledger
	// for the same TxID catches duplicates.
	err = utils.CheckProposalTxID(
		chdr.TxId,
		shdr.Nonce,
		shdr.Creator)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// continue the validation in a way that depends on the type specified in the header
	switch common.HeaderType(chdr.Type) {
	case common.HeaderType_CONFIG:
		//which the types are different the validation is the same
		//viz, validate a proposal to a chaincode. If we need other
		//special validation for confguration, we would have to implement
		//special validation
		fallthrough
	case common.HeaderType_ENDORSER_TRANSACTION:
		// validation of the proposal message knowing it's of type CHAINCODE
		chaincodeHdrExt, err := validateChaincodeProposalMessage(prop, hdr)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		return prop, chdr, shdr, chaincodeHdrExt, err
	default:
		//NOTE : we proably need a case
		return nil, nil, nil, nil, fmt.Errorf("Unsupported proposal type %d", common.HeaderType(chdr.Type))
	}
}

// given a creator, a message and a signature,
// this function returns nil if the creator
// is a valid cert and the signature is valid
func CheckSignatureFromCreator(creatorBytes []byte, sig []byte, msg []byte, ChainID string) error {
	log.Logger.Debugf("CheckSignatureFromCreator starts")

	// check for nil argument
	if creatorBytes == nil || sig == nil || msg == nil {
		return errors.New("Nil arguments")
	}

	mspObj := mspmgmt.GetIdentityDeserializer(ChainID)
	if mspObj == nil {
		return fmt.Errorf("could not get msp for chain [%s]", ChainID)
	}

	// get the identity of the creator
	creator, err := mspObj.DeserializeIdentity(creatorBytes)
	if err != nil {
		return fmt.Errorf("Failed to deserialize creator identity, err %s", err)
	}

	log.Logger.Debugf("CheckSignatureFromCreator info: creator is %s", creator.GetIdentifier())

	// ensure that creator is a valid certificate
	err = creator.Validate()
	if err != nil {
		return fmt.Errorf("The creator certificate is not valid, err %s", err)
	}

	log.Logger.Debugf("CheckSignatureFromCreator info: creator is valid")

	// validate the signature
	err = creator.Verify(msg, sig)
	if err != nil {
		return fmt.Errorf("The creator's signature over the proposal is not valid, err %s", err)
	}

	log.Logger.Debugf("CheckSignatureFromCreator exists successfully")

	return nil
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

// checks for a valid ChannelHeader
func validateChannelHeader(cHdr *common.ChannelHeader) error {
	// check for nil argument
	if cHdr == nil {
		return errors.New("Nil ChannelHeader provided")
	}

	// validate the header type
	if common.HeaderType(cHdr.Type) != common.HeaderType_ENDORSER_TRANSACTION &&
		common.HeaderType(cHdr.Type) != common.HeaderType_CONFIG_UPDATE &&
		common.HeaderType(cHdr.Type) != common.HeaderType_CONFIG {
		return fmt.Errorf("invalid header type %s", common.HeaderType(cHdr.Type))
	}

	log.Logger.Debugf("validateChannelHeader info: header type %d", common.HeaderType(cHdr.Type))

	// TODO: validate chainID in cHdr.ChainID

	// Validate epoch in cHdr.Epoch
	// Currently we enforce that Epoch is 0.
	// TODO: This check will be modified once the Epoch management
	// will be in place.
	if cHdr.Epoch != 0 {
		return fmt.Errorf("Invalid Epoch in ChannelHeader. It must be 0. It was [%d]", cHdr.Epoch)
	}

	// TODO: Validate version in cHdr.Version

	return nil
}

// checks for a valid Header
func validateCommonHeader(hdr *common.Header) (*common.ChannelHeader, *common.SignatureHeader, error) {
	if hdr == nil {
		return nil, nil, errors.New("Nil header")
	}

	chdr, err := utils.UnmarshalChannelHeader(hdr.ChannelHeader)
	if err != nil {
		return nil, nil, err
	}

	shdr, err := utils.GetSignatureHeader(hdr.SignatureHeader)
	if err != nil {
		return nil, nil, err
	}

	err = validateChannelHeader(chdr)
	if err != nil {
		return nil, nil, err
	}

	err = validateSignatureHeader(shdr)
	if err != nil {
		return nil, nil, err
	}

	return chdr, shdr, nil
}


