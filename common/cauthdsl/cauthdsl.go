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
	"fmt"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	cb "github.com/rongzer/blockchain/protos/common"
	mb "github.com/rongzer/blockchain/protos/msp"
)

// compile recursively builds a go evaluatable function corresponding to the policy specified
func compile(policy *cb.SignaturePolicy, identities []*mb.MSPPrincipal, deserializer msp.IdentityDeserializer) (func([]*cb.SignedData, []bool) bool, error) {
	if policy == nil {
		return nil, fmt.Errorf("Empty policy element")
	}

	switch t := policy.Type.(type) {
	case *cb.SignaturePolicy_NOutOf_:
		policies := make([]func([]*cb.SignedData, []bool) bool, len(t.NOutOf.Rules))
		for i, policy := range t.NOutOf.Rules {
			compiledPolicy, err := compile(policy, identities, deserializer)
			if err != nil {
				return nil, err
			}
			policies[i] = compiledPolicy

		}
		return func(signedData []*cb.SignedData, used []bool) bool {
			grepKey := time.Now().UnixNano()
			log.Logger.Debugf("%p gate %d evaluation starts", signedData, grepKey)
			verified := int32(0)
			_used := make([]bool, len(used))
			for _, policy := range policies {
				copy(_used, used)
				if policy(signedData, _used) {
					verified++
					copy(used, _used)
				}
			}

			if verified >= t.NOutOf.N {
				log.Logger.Debugf("%p gate %d evaluation succeeds", signedData, grepKey)
			} else {
				log.Logger.Debugf("%p gate %d evaluation fails", signedData, grepKey)
			}

			return verified >= t.NOutOf.N
		}, nil
	case *cb.SignaturePolicy_SignedBy:
		if t.SignedBy < 0 || t.SignedBy >= int32(len(identities)) {
			return nil, fmt.Errorf("identity index out of range, requested %v, but identies length is %d", t.SignedBy, len(identities))
		}
		signedByID := identities[t.SignedBy]
		return func(signedData []*cb.SignedData, used []bool) bool {
			log.Logger.Debugf("%p signed by %d principal evaluation starts (used %v)", signedData, t.SignedBy, used)
			for i, sd := range signedData {
				if used[i] {
					log.Logger.Debugf("%p skipping identity %d because it has already been used", signedData, i)
					continue
				}
				log.Logger.Debugf("%p processing identity %d with bytes of %x", signedData, i, sd.Identity)
				identity, err := deserializer.DeserializeIdentity(sd.Identity)
				if err != nil {
					log.Logger.Errorf("Principal deserialization failure (%s) for identity %x", err, sd.Identity)
					continue
				}
				err = identity.SatisfiesPrincipal(signedByID)
				if err != nil {
					log.Logger.Debugf("%p identity %d does not satisfy principal: %s", signedData, i, err)
					continue
				}
				log.Logger.Debugf("%p principal matched by identity %d", signedData, i)
				err = identity.Verify(sd.Data, sd.Signature)
				if err != nil {
					log.Logger.Debugf("%p signature for identity %d is invalid: %s", signedData, i, err)
					continue
				}
				log.Logger.Debugf("%p principal evaluation succeeds for identity %d", signedData, i)
				used[i] = true
				return true
			}
			log.Logger.Debugf("%p principal evaluation fails", signedData)
			return false
		}, nil
	default:
		return nil, fmt.Errorf("Unknown type: %T:%v", t, t)
	}
}
