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

package gossip

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	pull2 "github.com/rongzer/blockchain/peer/gossip/gossip/pull"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	proto "github.com/rongzer/blockchain/protos/gossip"
)

// certStore supports pull dissemination of identity messages
type certStore struct {
	sync.RWMutex
	selfIdentity api2.PeerIdentityType
	idMapper     identity2.Mapper
	pull         pull2.Mediator
	mcs          api2.MessageCryptoService
}

func newCertStore(puller pull2.Mediator, idMapper identity2.Mapper, selfIdentity api2.PeerIdentityType, mcs api2.MessageCryptoService) *certStore {
	selfPKIID := idMapper.GetPKIidOfCert(selfIdentity)

	certStore := &certStore{
		mcs:          mcs,
		pull:         puller,
		idMapper:     idMapper,
		selfIdentity: selfIdentity,
	}

	if err := certStore.idMapper.Put(selfPKIID, selfIdentity); err != nil {
		log.Logger.Panic("Failed associating self PKIID to cert:", err)
	}

	selfIdMsg, err := certStore.createIdentityMessage()
	if err != nil {
		log.Logger.Panic("Failed creating self identity message:", err)
	}
	puller.Add(selfIdMsg)
	puller.RegisterMsgHook(pull2.RequestMsgType, func(_ []string, msgs []*proto.SignedGossipMessage, _ proto.ReceivedMessage) {
		for _, msg := range msgs {
			pkiID := common2.PKIidType(msg.GetPeerIdentity().PkiId)
			cert := api2.PeerIdentityType(msg.GetPeerIdentity().Cert)
			if err := certStore.idMapper.Put(pkiID, cert); err != nil {
				log.Logger.Warn("Failed adding identity", cert, ", reason:", err)
			}
		}
	})
	return certStore
}

func (cs *certStore) handleMessage(msg proto.ReceivedMessage) {
	if update := msg.GetGossipMessage().GetDataUpdate(); update != nil {
		for _, env := range update.Data {
			m, err := env.ToGossipMessage()
			if err != nil {
				log.Logger.Warn("Data update contains an invalid message:", err)
				return
			}
			if !m.IsIdentityMsg() {
				log.Logger.Warn("Got a non-identity message:", m, "aborting")
				return
			}
			if err := cs.validateIdentityMsg(m); err != nil {
				log.Logger.Warn("Failed validating identity message:", err)
				return
			}
		}
	}
	cs.pull.HandleMessage(msg)
}

func (cs *certStore) validateIdentityMsg(msg *proto.SignedGossipMessage) error {
	idMsg := msg.GetPeerIdentity()
	if idMsg == nil {
		return fmt.Errorf("Identity empty: %+v", msg)
	}
	pkiID := idMsg.PkiId
	cert := idMsg.Cert
	calculatedPKIID := cs.mcs.GetPKIidOfCert(cert)
	claimedPKIID := common2.PKIidType(pkiID)
	if !bytes.Equal(calculatedPKIID, claimedPKIID) {
		return fmt.Errorf("Calculated pkiID doesn't match identity: calculated: %v, claimedPKI-ID: %v", calculatedPKIID, claimedPKIID)
	}

	verifier := func(peerIdentity []byte, signature, message []byte) error {
		return cs.mcs.Verify(peerIdentity, signature, message)
	}

	err := msg.Verify(cert, verifier)
	if err != nil {
		return fmt.Errorf("Failed verifying message: %v", err)
	}

	return cs.mcs.ValidateIdentity(idMsg.Cert)
}

func (cs *certStore) createIdentityMessage() (*proto.SignedGossipMessage, error) {
	identity := &proto.PeerIdentity{
		Cert:     cs.selfIdentity,
		Metadata: nil,
		PkiId:    cs.idMapper.GetPKIidOfCert(cs.selfIdentity),
	}
	m := &proto.GossipMessage{
		Channel: nil,
		Nonce:   0,
		Tag:     proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_PeerIdentity{
			PeerIdentity: identity,
		},
	}
	signer := func(msg []byte) ([]byte, error) {
		return cs.idMapper.Sign(msg)
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	_, err := sMsg.Sign(signer)
	return sMsg, err
}

func (cs *certStore) listRevokedPeers(isSuspected api2.PeerSuspector) []common2.PKIidType {
	revokedPeers := cs.idMapper.ListInvalidIdentities(isSuspected)
	for _, pkiID := range revokedPeers {
		cs.pull.Remove(string(pkiID))
	}
	return revokedPeers
}

func (cs *certStore) stop() {
	cs.pull.Stop()
}
