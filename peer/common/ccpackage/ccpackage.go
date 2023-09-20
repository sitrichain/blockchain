/*
Copyright IBM Corp. 2016-2017 All Rights Reserved.

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

package ccpackage

import (
	"fmt"
	"github.com/rongzer/blockchain/common/msp"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

// ExtractSignedCCDepSpec extracts the messages from the envelope
func ExtractSignedCCDepSpec(env *common.Envelope) (*common.ChannelHeader, *peer.SignedChaincodeDeploymentSpec, error) {
	p := &common.Payload{}
	if err := p.Unmarshal(env.Payload); err != nil {
		return nil, nil, err
	}
	ch := &common.ChannelHeader{}
	if err := ch.Unmarshal(p.Header.ChannelHeader); err != nil {
		return nil, nil, err
	}
	sp := &peer.SignedChaincodeDeploymentSpec{}
	if err := sp.Unmarshal(p.Data); err != nil {
		return nil, nil, err
	}

	return ch, sp, nil
}

func createSignedCCDepSpec(cdsbytes []byte, instpolicybytes []byte, endorsements []*peer.Endorsement) (*common.Envelope, error) {
	if cdsbytes == nil {
		return nil, fmt.Errorf("nil chaincode deployment spec")
	}

	if instpolicybytes == nil {
		return nil, fmt.Errorf("nil instantiation policy")
	}

	// create SignedChaincodeDeploymentSpec...
	cip := &peer.SignedChaincodeDeploymentSpec{ChaincodeDeploymentSpec: cdsbytes, InstantiationPolicy: instpolicybytes, OwnerEndorsements: endorsements}

	//...and marshal it
	cipbytes := utils.MarshalOrPanic(cip)

	//use defaults (this is definitely ok for install package)
	msgVersion := int32(0)
	epoch := uint64(0)
	chdr := utils.MakeChannelHeader(common.HeaderType_CHAINCODE_PACKAGE, msgVersion, "", epoch)

	// create the payload
	payl := &common.Payload{Header: &common.Header{ChannelHeader: utils.MarshalOrPanic(chdr)}, Data: cipbytes}
	paylBytes, err := utils.GetBytesPayload(payl)
	if err != nil {
		return nil, err
	}

	// here's the unsigned  envelope. The install package is endorsed if signingEntity != nil
	return &common.Envelope{Payload: paylBytes}, nil
}

// OwnerCreateSignedCCDepSpec creates a package from a ChaincodeDeploymentSpec and
// optionally endorses it
func OwnerCreateSignedCCDepSpec(cds *peer.ChaincodeDeploymentSpec, instPolicy *common.SignaturePolicyEnvelope, owner msp.SigningIdentity) (*common.Envelope, error) {
	if cds == nil {
		return nil, fmt.Errorf("invalid chaincode deployment spec")
	}

	if instPolicy == nil {
		return nil, fmt.Errorf("must provide an instantiation policy")
	}

	cdsbytes := utils.MarshalOrPanic(cds)

	instpolicybytes := utils.MarshalOrPanic(instPolicy)

	var endorsements []*peer.Endorsement
	//it is not mandatory (at this utils level) to have a signature
	//this is especially convenient during dev/test
	//it may be necessary to enforce it via a policy at a higher level
	if owner != nil {
		// serialize the signing identity
		endorser, err := owner.Serialize()
		if err != nil {
			return nil, fmt.Errorf("Could not serialize the signing identity for %s, err %s", owner.GetIdentifier(), err)
		}

		// sign the concatenation of cds, instpolicy and the serialized endorser identity with this endorser's key
		signature, err := owner.Sign(append(cdsbytes, append(instpolicybytes, endorser...)...))
		if err != nil {
			return nil, fmt.Errorf("Could not sign the ccpackage, err %s", err)
		}

		// each owner starts off the endorsements with one element. All such endorsed
		// packages will be collected in a final package by CreateSignedCCDepSpecForInstall
		// when endorsements will have all the entries
		endorsements = make([]*peer.Endorsement, 1)

		endorsements[0] = &peer.Endorsement{Signature: signature, Endorser: endorser}
	}

	return createSignedCCDepSpec(cdsbytes, instpolicybytes, endorsements)
}

