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
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	comm2 "github.com/rongzer/blockchain/peer/gossip/comm"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	discovery2 "github.com/rongzer/blockchain/peer/gossip/discovery"
	channel2 "github.com/rongzer/blockchain/peer/gossip/gossip/channel"
	"sync"
	"sync/atomic"

	proto "github.com/rongzer/blockchain/protos/gossip"
)

type channelState struct {
	stopping int32
	sync.RWMutex
	channels map[string]channel2.GossipChannel
	g        *gossipServiceImpl
}

func (cs *channelState) stop() {
	if cs.isStopping() {
		return
	}
	atomic.StoreInt32(&cs.stopping, int32(1))
	cs.Lock()
	defer cs.Unlock()
	for _, gc := range cs.channels {
		gc.Stop()
	}
}

func (cs *channelState) isStopping() bool {
	return atomic.LoadInt32(&cs.stopping) == int32(1)
}

func (cs *channelState) lookupChannelForMsg(msg proto.ReceivedMessage) channel2.GossipChannel {
	if msg.GetGossipMessage().IsStateInfoPullRequestMsg() {
		sipr := msg.GetGossipMessage().GetStateInfoPullReq()
		mac := sipr.Channel_MAC
		pkiID := msg.GetConnectionInfo().ID
		return cs.getGossipChannelByMAC(mac, pkiID)
	}
	return cs.lookupChannelForGossipMsg(msg.GetGossipMessage().GossipMessage)
}

func (cs *channelState) lookupChannelForGossipMsg(msg *proto.GossipMessage) channel2.GossipChannel {
	if !msg.IsStateInfoMsg() {
		// If we reached here then the message isn't:
		// 1) StateInfoPullRequest
		// 2) StateInfo
		// Hence, it was already sent to a peer (us) that has proved it knows the channel name, by
		// sending StateInfo messages in the past.
		// Therefore- we use the channel name from the message itself.
		return cs.getGossipChannelByChainID(msg.Channel)
	}

	// Else, it's a StateInfo message.
	stateInfMsg := msg.GetStateInfo()
	return cs.getGossipChannelByMAC(stateInfMsg.Channel_MAC, stateInfMsg.PkiId)
}

func (cs *channelState) getGossipChannelByMAC(receivedMAC []byte, pkiID common2.PKIidType) channel2.GossipChannel {
	// Iterate over the channels, and try to find a channel that the computation
	// of the MAC is equal to the MAC on the message.
	// If it is, then the peer that signed the message knows the name of the channel
	// because its PKI-ID was checked when the message was verified.
	cs.RLock()
	defer cs.RUnlock()
	for chanName, gc := range cs.channels {
		mac := channel2.GenerateMAC(pkiID, common2.ChainID(chanName))
		if bytes.Equal(mac, receivedMAC) {
			return gc
		}
	}
	return nil
}

func (cs *channelState) getGossipChannelByChainID(chainID common2.ChainID) channel2.GossipChannel {
	if cs.isStopping() {
		return nil
	}
	cs.RLock()
	defer cs.RUnlock()
	return cs.channels[string(chainID)]
}

func (cs *channelState) joinChannel(joinMsg api2.JoinChannelMessage, chainID common2.ChainID) {
	if cs.isStopping() {
		return
	}
	cs.Lock()
	defer cs.Unlock()
	if gc, exists := cs.channels[string(chainID)]; !exists {
		pkiID := cs.g.comm.GetPKIid()
		ga := &gossipAdapterImpl{gossipServiceImpl: cs.g, Discovery: cs.g.disc}
		gc := channel2.NewGossipChannel(pkiID, cs.g.selfOrg, cs.g.mcs, chainID, ga, joinMsg)
		cs.channels[string(chainID)] = gc
	} else {
		gc.ConfigureChannel(joinMsg)
	}
}

type gossipAdapterImpl struct {
	*gossipServiceImpl
	discovery2.Discovery
}

func (ga *gossipAdapterImpl) GetConf() channel2.Config {
	return channel2.Config{
		ID:                          ga.conf.ID,
		MaxBlockCountToStore:        ga.conf.MaxBlockCountToStore,
		PublishStateInfoInterval:    ga.conf.PublishStateInfoInterval,
		PullInterval:                ga.conf.PullInterval,
		PullPeerNum:                 ga.conf.PullPeerNum,
		RequestStateInfoInterval:    ga.conf.RequestStateInfoInterval,
		BlockExpirationInterval:     ga.conf.PullInterval * 100,
		StateInfoCacheSweepInterval: ga.conf.PullInterval * 5,
	}
}

// Gossip gossips a message
func (ga *gossipAdapterImpl) Gossip(msg *proto.SignedGossipMessage) {
	ga.gossipServiceImpl.emitter.Add(msg)
}

func (ga *gossipAdapterImpl) Send(msg *proto.SignedGossipMessage, peers ...*comm2.RemotePeer) {
	ga.gossipServiceImpl.comm.Send(msg, peers...)
}

// ValidateStateInfoMessage returns error if a message isn't valid
// nil otherwise
func (ga *gossipAdapterImpl) ValidateStateInfoMessage(msg *proto.SignedGossipMessage) error {
	return ga.gossipServiceImpl.validateStateInfoMsg(msg)
}

// GetOrgOfPeer returns the organization identifier of a certain peer
func (ga *gossipAdapterImpl) GetOrgOfPeer(PKIID common2.PKIidType) api2.OrgIdentityType {
	return ga.gossipServiceImpl.getOrgOfPeer(PKIID)
}

// GetIdentityByPKIID returns an identity of a peer with a certain
// pkiID, or nil if not found
func (ga *gossipAdapterImpl) GetIdentityByPKIID(pkiID common2.PKIidType) api2.PeerIdentityType {
	identity, err := ga.idMapper.Get(pkiID)
	if err != nil {
		return nil
	}
	return identity
}
