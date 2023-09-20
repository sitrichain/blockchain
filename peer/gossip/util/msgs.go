package util

import (
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	"sync"

	"github.com/rongzer/blockchain/common/util"
	proto "github.com/rongzer/blockchain/protos/gossip"
)

// MembershipStore struct which encapsulates
// membership message store abstraction
type MembershipStore struct {
	m sync.Map // map[string]*proto.SignedGossipMessage
}

// NewMembershipStore creates new membership store instance
func NewMembershipStore() *MembershipStore {
	return &MembershipStore{}
}

// MsgByID returns a message stored by a certain ID, or nil
// if such an ID isn't found
func (m *MembershipStore) MsgByID(pkiID common2.PKIidType) *proto.SignedGossipMessage {
	if msg, exists := m.m.Load(util.BytesToString(pkiID)); exists {
		return msg.(*proto.SignedGossipMessage)
	}
	return nil
}

// Size of the membership store
func (m *MembershipStore) Size() int {
	l := 0
	m.m.Range(func(key, value interface{}) bool {
		l += 1
		return true
	})
	return l
}

// Put associates msg with the given pkiID
func (m *MembershipStore) Put(pkiID common2.PKIidType, msg *proto.SignedGossipMessage) {
	m.m.Store(util.BytesToString(pkiID), msg)
}

// Remove removes a message with a given pkiID
func (m *MembershipStore) Remove(pkiID common2.PKIidType) {
	m.m.Delete(util.BytesToString(pkiID))
}

// ToSlice returns a slice backed by the elements
// of the MembershipStore
func (m *MembershipStore) ToSlice() []*proto.SignedGossipMessage {
	var members []*proto.SignedGossipMessage
	m.m.Range(func(key, value interface{}) bool {
		members = append(members, value.(*proto.SignedGossipMessage))
		return true
	})
	return members
}
