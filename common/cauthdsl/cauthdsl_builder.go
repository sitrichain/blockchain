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

package cauthdsl

import (
	"github.com/golang/protobuf/proto"
	"sort"

	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/msp"
	"github.com/rongzer/blockchain/protos/utils"
)

// AcceptAllPolicy always evaluates to true
var AcceptAllPolicy *cb.SignaturePolicyEnvelope

// MarshaledAcceptAllPolicy is the Marshaled version of AcceptAllPolicy
var MarshaledAcceptAllPolicy []byte

// RejectAllPolicy always evaluates to false
var RejectAllPolicy *cb.SignaturePolicyEnvelope

// MarshaledRejectAllPolicy is the Marshaled version of RejectAllPolicy
var MarshaledRejectAllPolicy []byte

func init() {
	var err error

	AcceptAllPolicy = Envelope(NOutOf(0, []*cb.SignaturePolicy{}), [][]byte{})
	MarshaledAcceptAllPolicy, err = proto.Marshal(AcceptAllPolicy)
	_, err = AcceptAllPolicy.Marshal()
	if err != nil {
		panic("Error marshaling trueEnvelope")
	}

	RejectAllPolicy = Envelope(NOutOf(1, []*cb.SignaturePolicy{}), [][]byte{})
	MarshaledRejectAllPolicy, err = proto.Marshal(RejectAllPolicy)
	_, err = RejectAllPolicy.Marshal()
	if err != nil {
		panic("Error marshaling falseEnvelope")
	}
}

// Envelope builds an envelope message embedding a SignaturePolicy
func Envelope(policy *cb.SignaturePolicy, identities [][]byte) *cb.SignaturePolicyEnvelope {
	ids := make([]*msp.MSPPrincipal, len(identities))
	for i := range ids {
		ids[i] = &msp.MSPPrincipal{PrincipalClassification: msp.MSPPrincipal_IDENTITY, Principal: identities[i]}
	}

	return &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       policy,
		Identities: ids,
	}
}

// SignedBy creates a SignaturePolicy requiring a given signer's signature
func SignedBy(index int32) *cb.SignaturePolicy {
	return &cb.SignaturePolicy{
		Type: &cb.SignaturePolicy_SignedBy{
			SignedBy: index,
		},
	}
}

// SignedByMspMember creates a SignaturePolicyEnvelope
// requiring 1 signature from any member of the specified MSP
func SignedByMspMember(mspId string) *cb.SignaturePolicyEnvelope {
	// specify the principal: it's a member of the msp we just found
	principal := &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: mspId})}

	// create the policy: it requires exactly 1 signature from the first (and only) principal
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*cb.SignaturePolicy{SignedBy(0)}),
		Identities: []*msp.MSPPrincipal{principal},
	}

	return p
}

// SignedByMspAdmin creates a SignaturePolicyEnvelope
// requiring 1 signature from any admin of the specified MSP
func SignedByMspAdmin(mspId string) *cb.SignaturePolicyEnvelope {
	// specify the principal: it's a member of the msp we just found
	principal := &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_ADMIN, MspIdentifier: mspId})}

	// create the policy: it requires exactly 1 signature from the first (and only) principal
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*cb.SignaturePolicy{SignedBy(0)}),
		Identities: []*msp.MSPPrincipal{principal},
	}

	return p
}

//wrapper for generating "any of a given role" type policies
func signedByAnyOfGivenRole(role msp.MSPRole_MSPRoleType, ids []string) *cb.SignaturePolicyEnvelope {
	// we create an array of principals, one principal
	// per application MSP defined on this chain
	sort.Strings(ids)
	principals := make([]*msp.MSPPrincipal, len(ids))
	sigspolicy := make([]*cb.SignaturePolicy, len(ids))
	for i, id := range ids {
		principals[i] = &msp.MSPPrincipal{
			PrincipalClassification: msp.MSPPrincipal_ROLE,
			Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: role, MspIdentifier: id})}
		sigspolicy[i] = SignedBy(int32(i))
	}

	// create the policy: it requires exactly 1 signature from any of the principals
	p := &cb.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, sigspolicy),
		Identities: principals,
	}

	return p
}

// SignedByAnyMember returns a policy that requires one valid
// signature from a member of any of the orgs whose ids are
// listed in the supplied string array
func SignedByAnyMember(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_MEMBER, ids)
}

// SignedByAnyAdmin returns a policy that requires one valid
// signature from a admin of any of the orgs whose ids are
// listed in the supplied string array
func SignedByAnyAdmin(ids []string) *cb.SignaturePolicyEnvelope {
	return signedByAnyOfGivenRole(msp.MSPRole_ADMIN, ids)
}

// NOutOf creates a policy which requires N out of the slice of policies to evaluate to true
func NOutOf(n int32, policies []*cb.SignaturePolicy) *cb.SignaturePolicy {
	return &cb.SignaturePolicy{
		Type: &cb.SignaturePolicy_NOutOf_{
			NOutOf: &cb.SignaturePolicy_NOutOf{
				N:     n,
				Rules: policies,
			},
		},
	}
}
