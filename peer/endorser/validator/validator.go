package validator

import (
	"errors"
	"fmt"

	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/common/validation"
	"github.com/rongzer/blockchain/peer/policy"
	"github.com/rongzer/blockchain/peer/scc"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
)

type Validator struct {
	policyChecker policy.PolicyChecker
}

func NewValidator() *Validator {
	v := new(Validator)
	v.policyChecker = policy.NewPolicyChecker(
		chain.NewChannelPolicyManagerGetter(),
		mgmt.GetLocalMSP(),
		mgmt.NewLocalMSPPrincipalGetter(),
	)

	return v
}

func (v *Validator) checkACL(signedProp *peer.SignedProposal, chdr *common.ChannelHeader, _ *common.SignatureHeader, _ *peer.ChaincodeHeaderExtension) error {
	return v.policyChecker.CheckPolicy(chdr.ChannelId, policies.ChannelApplicationWriters, signedProp)
}

func (v *Validator) ValidateEndorserProposal(signedProp *peer.SignedProposal) (*peer.Proposal, *common.ChannelHeader, *common.SignatureHeader, *peer.ChaincodeHeaderExtension, error) {

	// at first, we check whether the message is valid
	prop, chdr, shdr, hdrExt, err := validation.ValidateProposalMessage(signedProp)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// block invocations to security-sensitive system chaincodes
	if scc.IsSysCCAndNotInvokableExternal(hdrExt.ChaincodeId.Name) {
		err = fmt.Errorf("Chaincode %s cannot be invoked through a proposal", hdrExt.ChaincodeId.Name)
		return nil, nil, nil, nil, err
	}

	chainID := chdr.ChannelId

	// Check for uniqueness of prop.TxID with ledger
	// Notice that ValidateProposalMessage has already verified
	// that TxID is computed properly
	txid := chdr.TxId
	if txid == "" {
		err = errors.New("Invalid txID. It must be different from the empty string.")
	}

	if chainID != "" {
		// here we handle uniqueness check and ACLs for proposals targeting a chain
		lgr := chain.GetLedger(chainID)
		if lgr == nil {
			return nil, nil, nil, nil, err
		}
		if _, err := lgr.GetTransactionByID(txid); err == nil {
			return nil, nil, nil, nil, err
		}

		// check ACL only for application chaincodes; ACLs
		// for system chaincodes are checked elsewhere
		if !scc.IsSysCC(hdrExt.ChaincodeId.Name) {
			// check that the proposal complies with the channel's writers
			if err = v.checkACL(signedProp, chdr, shdr, hdrExt); err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	return prop, chdr, shdr, hdrExt, nil
}
