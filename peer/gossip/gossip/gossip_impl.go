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
	"crypto/tls"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	comm2 "github.com/rongzer/blockchain/peer/gossip/comm"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	discovery2 "github.com/rongzer/blockchain/peer/gossip/discovery"
	filter2 "github.com/rongzer/blockchain/peer/gossip/filter"
	channel2 "github.com/rongzer/blockchain/peer/gossip/gossip/channel"
	msgstore2 "github.com/rongzer/blockchain/peer/gossip/gossip/msgstore"
	pull2 "github.com/rongzer/blockchain/peer/gossip/gossip/pull"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"

	proto "github.com/rongzer/blockchain/protos/gossip"
	"google.golang.org/grpc"
)

const (
	presumedDeadChanSize = 100
	acceptChanSize       = 100
)

var (
	identityExpirationCheckInterval = time.Hour * 24
	identityInactivityCheckInterval = time.Minute * 10
)

type channelRoutingFilterFactory func(channel2.GossipChannel) filter2.RoutingFilter

type gossipServiceImpl struct {
	selfIdentity          api2.PeerIdentityType
	includeIdentityPeriod time.Time
	certStore             *certStore
	idMapper              identity2.Mapper
	presumedDead          chan common2.PKIidType
	disc                  discovery2.Discovery
	comm                  comm2.Comm
	incTime               time.Time
	selfOrg               api2.OrgIdentityType
	*comm2.ChannelDeMultiplexer
	stopSignal        *sync.WaitGroup
	conf              *Config
	toDieChan         chan struct{}
	stopFlag          int32
	emitter           batchingEmitter
	discAdapter       *discoveryAdapter
	secAdvisor        api2.SecurityAdvisor
	chanState         *channelState
	disSecAdap        *discoverySecurityAdapter
	mcs               api2.MessageCryptoService
	stateInfoMsgStore msgstore2.MessageStore
}

// NewGossipService creates a gossip instance attached to a gRPC server
func NewGossipService(conf *Config, s *grpc.Server, secAdvisor api2.SecurityAdvisor,
	mcs api2.MessageCryptoService, idMapper identity2.Mapper, selfIdentity api2.PeerIdentityType,
	secureDialOpts api2.PeerSecureDialOpts) Gossip {

	var c comm2.Comm
	var err error

	if s == nil {
		c, err = createCommWithServer(conf.BindPort, idMapper, selfIdentity, secureDialOpts)
	} else {
		c, err = createCommWithoutServer(s, conf.TLSServerCert, idMapper, selfIdentity, secureDialOpts)
	}

	if err != nil {
		log.Logger.Error("Failed instntiating communication layer:", err)
		return nil
	}

	g := &gossipServiceImpl{
		selfOrg:               secAdvisor.OrgByPeerIdentity(selfIdentity),
		secAdvisor:            secAdvisor,
		selfIdentity:          selfIdentity,
		presumedDead:          make(chan common2.PKIidType, presumedDeadChanSize),
		idMapper:              idMapper,
		disc:                  nil,
		mcs:                   mcs,
		comm:                  c,
		conf:                  conf,
		ChannelDeMultiplexer:  comm2.NewChannelDemultiplexer(),
		toDieChan:             make(chan struct{}, 1),
		stopFlag:              int32(0),
		stopSignal:            &sync.WaitGroup{},
		includeIdentityPeriod: time.Now().Add(conf.PublishCertPeriod),
	}
	g.stateInfoMsgStore = g.newStateInfoMsgStore()

	g.chanState = newChannelState(g)
	g.emitter = newBatchingEmitter(conf.PropagateIterations,
		conf.MaxPropagationBurstSize, conf.MaxPropagationBurstLatency,
		g.sendGossipBatch)

	g.discAdapter = g.newDiscoveryAdapter()
	g.disSecAdap = g.newDiscoverySecurityAdapter()
	g.disc = discovery2.NewDiscoveryService(g.selfNetworkMember(), g.discAdapter, g.disSecAdap, g.disclosurePolicy)
	log.Logger.Info("Creating gossip service with self membership of", g.selfNetworkMember())

	g.certStore = newCertStore(g.createCertStorePuller(), idMapper, selfIdentity, mcs)

	if g.conf.ExternalEndpoint == "" {
		log.Logger.Warn("External endpoint is empty, peer will not be accessible outside of its organization")
	}

	go g.start()
	go g.periodicalIdentityValidationAndExpiration()
	go g.connect2BootstrapPeers()

	return g
}

func (g *gossipServiceImpl) newStateInfoMsgStore() msgstore2.MessageStore {
	pol := proto.NewGossipMessageComparator(0)
	return msgstore2.NewMessageStoreExpirable(pol,
		msgstore2.Noop,
		g.conf.PublishStateInfoInterval*100,
		nil,
		nil,
		msgstore2.Noop)
}

func (g *gossipServiceImpl) selfNetworkMember() discovery2.NetworkMember {
	self := discovery2.NetworkMember{
		Endpoint:         g.conf.ExternalEndpoint,
		PKIid:            g.comm.GetPKIid(),
		Metadata:         []byte{},
		InternalEndpoint: g.conf.InternalEndpoint,
	}
	if g.disc != nil {
		self.Metadata = g.disc.Self().Metadata
	}
	return self
}

func newChannelState(g *gossipServiceImpl) *channelState {
	return &channelState{
		stopping: int32(0),
		channels: make(map[string]channel2.GossipChannel),
		g:        g,
	}
}

func createCommWithoutServer(s *grpc.Server, cert *tls.Certificate, idStore identity2.Mapper,
	identity api2.PeerIdentityType, secureDialOpts api2.PeerSecureDialOpts) (comm2.Comm, error) {
	return comm2.NewCommInstance(s, cert, idStore, identity, secureDialOpts)
}

func createCommWithServer(port int, idStore identity2.Mapper, identity api2.PeerIdentityType,
	secureDialOpts api2.PeerSecureDialOpts) (comm2.Comm, error) {
	return comm2.NewCommInstanceWithServer(port, idStore, identity, secureDialOpts)
}

func (g *gossipServiceImpl) toDie() bool {
	return atomic.LoadInt32(&g.stopFlag) == int32(1)
}

func (g *gossipServiceImpl) JoinChan(joinMsg api2.JoinChannelMessage, chainID common2.ChainID) {
	// joinMsg is supposed to have been already verified
	g.chanState.joinChannel(joinMsg, chainID)

	for _, org := range joinMsg.Members() {
		g.learnAnchorPeers(org, joinMsg.AnchorPeersOf(org))
		log.Logger.Warnf("joinchan org : %s", string(org))
	}
}

// SuspectPeers makes the gossip instance validate identities of suspected peers, and close
// any connections to peers with identities that are found invalid
func (g *gossipServiceImpl) SuspectPeers(isSuspected api2.PeerSuspector) {
	for _, pkiID := range g.certStore.listRevokedPeers(isSuspected) {
		g.comm.CloseConn(&comm2.RemotePeer{PKIID: pkiID})
		log.Logger.Warnf("isSuspected failed close : %s", &comm2.RemotePeer{PKIID: pkiID})
	}
}

func (g *gossipServiceImpl) periodicalIdentityValidationAndExpiration() {
	// We check once every identityExpirationCheckInterval for identities that have been expired
	go g.periodicalIdentityValidation(func(identity api2.PeerIdentityType) bool {
		// We need to validate every identity to check if it has been expired
		return true
	}, identityExpirationCheckInterval)

	// We check once every identityInactivityCheckInterval for identities that have not been used for a long time
	go g.periodicalIdentityValidation(func(identity api2.PeerIdentityType) bool {
		// We don't validate any identity, because we just want to know whether
		// it has not been used for a long time
		return false
	}, identityInactivityCheckInterval)
}

func (g *gossipServiceImpl) periodicalIdentityValidation(suspectFunc api2.PeerSuspector, interval time.Duration) {
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case <-time.After(interval):
			g.SuspectPeers(suspectFunc)
		}
	}
}

func (g *gossipServiceImpl) learnAnchorPeers(orgOfAnchorPeers api2.OrgIdentityType, anchorPeers []api2.AnchorPeer) {
	for _, ap := range anchorPeers {
		if ap.Host == "" {
			log.Logger.Warn("Got empty hostname, skipping connecting to anchor peer", ap)
			continue
		}
		if ap.Port == 0 {
			log.Logger.Warn("Got invalid port (0), skipping connecting to anchor peer", ap)
			continue
		}
		endpoint := fmt.Sprintf("%s:%d", ap.Host, ap.Port)
		// Skip connecting to self
		if g.selfNetworkMember().Endpoint == endpoint || g.selfNetworkMember().InternalEndpoint == endpoint {
			log.Logger.Info("Anchor peer with same endpoint, skipping connecting to myself")
			continue
		}

		inOurOrg := bytes.Equal(g.selfOrg, orgOfAnchorPeers)
		if !inOurOrg && g.selfNetworkMember().Endpoint == "" {
			log.Logger.Infof("Anchor peer %s:%d isn't in our org(%v) and we have no external endpoint, skipping", ap.Host, ap.Port, string(orgOfAnchorPeers))
			continue
		}
		identifier := func() (*discovery2.PeerIdentification, error) {
			remotePeerIdentity, err := g.comm.Handshake(&comm2.RemotePeer{Endpoint: endpoint})
			if err != nil {
				log.Logger.Warn("Deep probe of", endpoint, "failed:", err)
				return nil, err
			}
			isAnchorPeerInMyOrg := bytes.Equal(g.selfOrg, g.secAdvisor.OrgByPeerIdentity(remotePeerIdentity))
			if bytes.Equal(orgOfAnchorPeers, g.selfOrg) && !isAnchorPeerInMyOrg {
				err := fmt.Sprintf("Anchor peer %s isn't in our org, but is claimed to be", endpoint)
				log.Logger.Warn(err)
				return nil, errors.New(err)
			}
			pkiID := g.mcs.GetPKIidOfCert(remotePeerIdentity)
			if len(pkiID) == 0 {
				return nil, fmt.Errorf("Wasn't able to extract PKI-ID of remote peer with identity of %v", remotePeerIdentity)
			}
			return &discovery2.PeerIdentification{
				ID:      pkiID,
				SelfOrg: isAnchorPeerInMyOrg,
			}, nil
		}

		g.disc.Connect(discovery2.NetworkMember{
			InternalEndpoint: endpoint, Endpoint: endpoint}, identifier)
	}
}

func (g *gossipServiceImpl) handlePresumedDead() {
	defer log.Logger.Debug("Exiting")
	g.stopSignal.Add(1)
	defer g.stopSignal.Done()
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case deadEndpoint := <-g.comm.PresumedDead():
			g.presumedDead <- deadEndpoint
		}
	}
}

func (g *gossipServiceImpl) syncDiscovery() {
	log.Logger.Info("Entering discovery sync with interval", g.conf.PullInterval)
	defer log.Logger.Warn("Exiting discovery sync loop")

	t := time.NewTicker(g.conf.PullInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if g.toDie() {
				return
			}
			g.disc.InitiateSync(g.conf.PullPeerNum)
		}
	}
}

func (g *gossipServiceImpl) start() {
	go g.syncDiscovery()
	go g.handlePresumedDead()

	msgSelector := func(msg interface{}) bool {
		gMsg, isGossipMsg := msg.(proto.ReceivedMessage)
		if !isGossipMsg {
			return false
		}

		isConn := gMsg.GetGossipMessage().GetConn() != nil
		isEmpty := gMsg.GetGossipMessage().GetEmpty() != nil

		return !(isConn || isEmpty)
	}

	incMsgs := g.comm.Accept(msgSelector)

	go g.acceptMessages(incMsgs)

	log.Logger.Info("Gossip instance", g.conf.ID, "started")
}

func (g *gossipServiceImpl) acceptMessages(incMsgs <-chan proto.ReceivedMessage) {
	defer log.Logger.Debug("Exiting")
	g.stopSignal.Add(1)
	defer g.stopSignal.Done()
	for {
		select {
		case s := <-g.toDieChan:
			g.toDieChan <- s
			return
		case msg := <-incMsgs:
			g.handleMessage(msg)
		}
	}
}

func (g *gossipServiceImpl) handleMessage(m proto.ReceivedMessage) {
	if g.toDie() {
		return
	}

	if m == nil || m.GetGossipMessage() == nil {
		return
	}

	msg := m.GetGossipMessage()
	log.Logger.Debugf("Receive message from %s: %s", m.GetConnectionInfo(), msg)

	if !g.validateMsg(m) {
		log.Logger.Warnf("Receive invalid message from %s: %s", m.GetConnectionInfo(), msg)
		return
	}

	if msg.IsChannelRestricted() {
		if gc := g.chanState.lookupChannelForMsg(m); gc == nil {
			// If we're not in the channel, we should still forward to peers of our org
			// in case it's a StateInfo message
			if g.isInMyorg(discovery2.NetworkMember{PKIid: m.GetConnectionInfo().ID}) && msg.IsStateInfoMsg() {
				if g.stateInfoMsgStore.Add(msg) {
					g.emitter.Add(msg)
				}
			}
			if !g.toDie() {
				log.Logger.Debug("No such channel", msg.Channel, "discarding message", msg)
			}
		} else {
			if m.GetGossipMessage().IsLeadershipMsg() {
				if err := g.validateLeadershipMessage(m.GetGossipMessage()); err != nil {
					log.Logger.Warn("Failed validating LeaderElection message:", err)
					return
				}
			}
			gc.HandleMessage(m)
		}
		return
	}

	if selectOnlyDiscoveryMessages(m) {
		// It's a membership request, check its self information
		// matches the sender
		if m.GetGossipMessage().GetMemReq() != nil {
			sMsg, err := m.GetGossipMessage().GetMemReq().SelfInformation.ToGossipMessage()
			if err != nil {
				log.Logger.Warn("Got membership request with invalid selfInfo:", err)
				return
			}
			if !sMsg.IsAliveMsg() {
				log.Logger.Warn("Got membership request with selfInfo that isn't an AliveMessage")
				return
			}
			if !bytes.Equal(sMsg.GetAliveMsg().Membership.PkiId, m.GetConnectionInfo().ID) {
				log.Logger.Warn("Got membership request with selfInfo that doesn't match the handshake")
				return
			}
		}
		g.forwardDiscoveryMsg(m)
	}

	if msg.IsPullMsg() && msg.GetPullMsgType() == proto.PullMsgType_IDENTITY_MSG {
		g.certStore.handleMessage(m)
	}
}

func (g *gossipServiceImpl) forwardDiscoveryMsg(msg proto.ReceivedMessage) {
	defer func() { // can be closed while shutting down
		recover()
	}()

	g.discAdapter.incChan <- msg.GetGossipMessage()
}

// validateMsg checks the signature of the message if exists,
// and also checks that the tag matches the message type
func (g *gossipServiceImpl) validateMsg(msg proto.ReceivedMessage) bool {
	if err := msg.GetGossipMessage().IsTagLegal(); err != nil {
		log.Logger.Warn("Tag of", msg.GetGossipMessage(), "isn't legal:", err)
		return false
	}

	if msg.GetGossipMessage().IsAliveMsg() {
		if !g.disSecAdap.ValidateAliveMsg(msg.GetGossipMessage()) {
			return false
		}
	}

	if msg.GetGossipMessage().IsStateInfoMsg() {
		if err := g.validateStateInfoMsg(msg.GetGossipMessage()); err != nil {
			log.Logger.Warn("StateInfo message", msg, "is found invalid:", err)
			return false
		}
	}
	return true
}

func (g *gossipServiceImpl) sendGossipBatch(a []interface{}) {
	msgs2Gossip := make([]*proto.SignedGossipMessage, len(a))
	for i, e := range a {
		msgs2Gossip[i] = e.(*proto.SignedGossipMessage)
	}
	g.gossipBatch(msgs2Gossip)
}

// gossipBatch - This is the method that actually decides to which peers to gossip the message
// batch we possess.
// For efficiency, we first isolate all the messages that have the same routing policy
// and send them together, and only after that move to the next group of messages.
// i.e: we send all blocks of channel C to the same group of peers,
// and send all StateInfo messages to the same group of peers, etc. etc.
// When we send blocks, we send only to peers that advertised themselves in the channel.
// When we send StateInfo messages, we send to peers in the channel.
// When we send messages that are marked to be sent only within the org, we send all of these messages
// to the same set of peers.
// The rest of the messages that have no restrictions on their destinations can be sent
// to any group of peers.
func (g *gossipServiceImpl) gossipBatch(msgs []*proto.SignedGossipMessage) {
	if g.disc == nil {
		log.Logger.Error("Discovery has not been initialized yet, aborting!")
		return
	}

	var blocks []*proto.SignedGossipMessage
	var stateInfoMsgs []*proto.SignedGossipMessage
	var orgMsgs []*proto.SignedGossipMessage
	var leadershipMsgs []*proto.SignedGossipMessage

	isABlock := func(o interface{}) bool {
		return o.(*proto.SignedGossipMessage).IsDataMsg()
	}
	isAStateInfoMsg := func(o interface{}) bool {
		return o.(*proto.SignedGossipMessage).IsStateInfoMsg()
	}
	aliveMsgsWithNoEndpointAndInOurOrg := func(o interface{}) bool {
		msg := o.(*proto.SignedGossipMessage)
		if !msg.IsAliveMsg() {
			return false
		}
		member := msg.GetAliveMsg().Membership
		return member.Endpoint == "" && g.isInMyorg(discovery2.NetworkMember{PKIid: member.PkiId})
	}
	isOrgRestricted := func(o interface{}) bool {
		return aliveMsgsWithNoEndpointAndInOurOrg(o) || o.(*proto.SignedGossipMessage).IsOrgRestricted()
	}
	isLeadershipMsg := func(o interface{}) bool {
		return o.(*proto.SignedGossipMessage).IsLeadershipMsg()
	}

	// Gossip blocks
	blocks, msgs = partitionMessages(isABlock, msgs)
	g.gossipInChan(blocks, func(gc channel2.GossipChannel) filter2.RoutingFilter {
		return filter2.CombineRoutingFilters(gc.EligibleForChannel, gc.IsMemberInChan, g.isInMyorg)
	})

	// Gossip Leadership messages
	leadershipMsgs, msgs = partitionMessages(isLeadershipMsg, msgs)
	g.gossipInChan(leadershipMsgs, func(gc channel2.GossipChannel) filter2.RoutingFilter {
		return filter2.CombineRoutingFilters(gc.EligibleForChannel, gc.IsMemberInChan, g.isInMyorg)
	})

	// Gossip StateInfo messages
	stateInfoMsgs, msgs = partitionMessages(isAStateInfoMsg, msgs)
	for _, stateInfMsg := range stateInfoMsgs {
		peerSelector := g.isInMyorg
		gc := g.chanState.lookupChannelForGossipMsg(stateInfMsg.GossipMessage)
		if gc != nil && g.hasExternalEndpoint(stateInfMsg.GossipMessage.GetStateInfo().PkiId) {
			peerSelector = gc.IsMemberInChan
		}

		peers2Send := filter2.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), peerSelector)
		g.comm.Send(stateInfMsg, peers2Send...)

		//log.Logger.Warnf("send stateinfo msg : %s to peers : %s", stateInfMsg, peers2Send)
	}

	// Gossip messages restricted to our org
	orgMsgs, msgs = partitionMessages(isOrgRestricted, msgs)
	peers2Send := filter2.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), g.isInMyorg)
	for _, msg := range orgMsgs {
		g.comm.Send(msg, peers2Send...)
		//log.Logger.Warnf("gossip org : %s peers2Send : %s", msg.Channel, peers2Send)
	}

	// Finally, gossip the remaining messages
	for _, msg := range msgs {
		if !msg.IsAliveMsg() {
			log.Logger.Error("Unknown message type", msg)
			continue
		}
		selectByOriginOrg := g.peersByOriginOrgPolicy(discovery2.NetworkMember{PKIid: msg.GetAliveMsg().Membership.PkiId})
		peers2Send := filter2.SelectPeers(g.conf.PropagatePeerNum, g.disc.GetMembership(), selectByOriginOrg)
		g.sendAndFilterSecrets(msg, peers2Send...)
	}
}

func (g *gossipServiceImpl) sendAndFilterSecrets(msg *proto.SignedGossipMessage, peers ...*comm2.RemotePeer) {
	for _, peer := range peers {
		// Prevent forwarding alive messages of external organizations
		// to peers that have no external endpoints
		aliveMsgFromDiffOrg := msg.IsAliveMsg() && !g.isInMyorg(discovery2.NetworkMember{PKIid: msg.GetAliveMsg().Membership.PkiId})
		if aliveMsgFromDiffOrg && !g.hasExternalEndpoint(peer.PKIID) {
			continue
		}
		// Don't gossip secrets
		if !g.isInMyorg(discovery2.NetworkMember{PKIid: peer.PKIID}) {
			msg.Envelope.SecretEnvelope = nil
		}

		g.comm.Send(msg, peer)
	}
}

// gossipInChan gossips a given GossipMessage slice according to a channel's routing policy.
func (g *gossipServiceImpl) gossipInChan(messages []*proto.SignedGossipMessage, chanRoutingFactory channelRoutingFilterFactory) {
	if len(messages) == 0 {
		return
	}
	totalChannels := extractChannels(messages)
	var channel common2.ChainID
	var messagesOfChannel []*proto.SignedGossipMessage
	for len(totalChannels) > 0 {
		// Take first channel
		channel, totalChannels = totalChannels[0], totalChannels[1:]
		// Extract all messages of that channel
		grabMsgs := func(o interface{}) bool {
			return bytes.Equal(o.(*proto.SignedGossipMessage).Channel, channel)
		}
		messagesOfChannel, messages = partitionMessages(grabMsgs, messages)
		if len(messagesOfChannel) == 0 {
			continue
		}
		// Grab channel object for that channel
		gc := g.chanState.getGossipChannelByChainID(channel)
		if gc == nil {
			log.Logger.Warn("Channel", channel, "wasn't found")
			continue
		}
		// Select the peers to send the messages to
		// For leadership messages we will select all peers that pass routing factory - e.g. all peers in channel and org
		membership := g.disc.GetMembership()
		var peers2Send []*comm2.RemotePeer
		if messagesOfChannel[0].IsLeadershipMsg() {
			peers2Send = filter2.SelectPeers(len(membership), membership, chanRoutingFactory(gc))
		} else {
			peers2Send = filter2.SelectPeers(g.conf.PropagatePeerNum, membership, chanRoutingFactory(gc))
		}

		// Send the messages to the remote peers
		for _, msg := range messagesOfChannel {
			g.comm.Send(msg, peers2Send...)
		}
	}
}

// Gossip sends a message to other peers to the network
func (g *gossipServiceImpl) Gossip(msg *proto.GossipMessage) {
	// Educate developers to Gossip messages with the right tags.
	// See IsTagLegal() for wanted behavior.
	if err := msg.IsTagLegal(); err != nil {
		panic(err)
	}

	sMsg := &proto.SignedGossipMessage{
		GossipMessage: msg,
	}

	var err error
	if sMsg.IsDataMsg() {
		sMsg, err = sMsg.NoopSign()
	} else {
		_, err = sMsg.Sign(func(msg []byte) ([]byte, error) {
			return g.mcs.Sign(msg)
		})
	}

	if err != nil {
		log.Logger.Warn("Failed signing message:", err)
		return
	}

	if msg.IsChannelRestricted() {
		gc := g.chanState.getGossipChannelByChainID(msg.Channel)
		if gc == nil {
			log.Logger.Warn("Failed obtaining gossipChannel of", msg.Channel, "aborting")
			return
		}
		if msg.IsDataMsg() {
			gc.AddToMsgStore(sMsg)
		}
	}

	if g.conf.PropagateIterations == 0 {
		return
	}
	g.emitter.Add(sMsg)
}

// Send sends a message to remote peers
func (g *gossipServiceImpl) Send(msg *proto.GossipMessage, peers ...*comm2.RemotePeer) {
	m, err := msg.NoopSign()
	if err != nil {
		log.Logger.Warn("Failed creating SignedGossipMessage:", err)
		return
	}
	if msg.IsDataMsg() {
		log.Logger.Debugf("send data message %v", peers)
	}
	g.comm.Send(m, peers...)
}

// GetPeers returns a mapping of endpoint --> []discovery.NetworkMember
func (g *gossipServiceImpl) Peers() []discovery2.NetworkMember {
	var s []discovery2.NetworkMember
	for _, member := range g.disc.GetMembership() {
		s = append(s, member)
	}
	return s

}

// PeersOfChannel returns the NetworkMembers considered alive
//  and also subscribed to the channel given
func (g *gossipServiceImpl) PeersOfChannel(channel common2.ChainID) []discovery2.NetworkMember {
	gc := g.chanState.getGossipChannelByChainID(channel)
	if gc == nil {
		log.Logger.Debug("No such channel", channel)
		return nil
	}

	return gc.GetPeers()
}

// Stop stops the gossip component
func (g *gossipServiceImpl) Stop() {
	if g.toDie() {
		return
	}
	atomic.StoreInt32(&g.stopFlag, int32(1))
	log.Logger.Info("Stopping gossip")
	comWG := sync.WaitGroup{}
	comWG.Add(1)
	go func() {
		defer comWG.Done()
		g.comm.Stop()
	}()
	g.chanState.stop()
	g.discAdapter.close()
	g.disc.Stop()
	g.certStore.stop()
	g.toDieChan <- struct{}{}
	g.emitter.Stop()
	g.ChannelDeMultiplexer.Close()
	g.stateInfoMsgStore.Stop()
	g.stopSignal.Wait()
	comWG.Wait()
}

func (g *gossipServiceImpl) UpdateMetadata(md []byte) {
	g.disc.UpdateMetadata(md)
}

// UpdateChannelMetadata updates the self metadata the peer
// publishes to other peers about its channel-related state
func (g *gossipServiceImpl) UpdateChannelMetadata(md []byte, chainID common2.ChainID) {
	gc := g.chanState.getGossipChannelByChainID(chainID)
	if gc == nil {
		log.Logger.Debug("No such channel", chainID)
		return
	}
	stateInfMsg, err := g.createStateInfoMsg(md, chainID)
	if err != nil {
		log.Logger.Error("Failed creating StateInfo message")
		return
	}
	gc.UpdateStateInfo(stateInfMsg)
}

// Accept returns a dedicated read-only channel for messages sent by other nodes that match a certain predicate.
// If passThrough is false, the messages are processed by the gossip layer beforehand.
// If passThrough is true, the gossip layer doesn't intervene and the messages
// can be used to send a reply back to the sender
func (g *gossipServiceImpl) Accept(acceptor common2.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	if passThrough {
		return nil, g.comm.Accept(acceptor)
	}
	acceptByType := func(o interface{}) bool {
		if o, isGossipMsg := o.(*proto.GossipMessage); isGossipMsg {
			return acceptor(o)
		}
		if o, isSignedMsg := o.(*proto.SignedGossipMessage); isSignedMsg {
			sMsg := o
			return acceptor(sMsg.GossipMessage)
		}
		log.Logger.Warn("Message type:", reflect.TypeOf(o), "cannot be evaluated")
		return false
	}
	inCh := g.AddChannel(acceptByType)
	outCh := make(chan *proto.GossipMessage, acceptChanSize)
	go func() {
		for {
			select {
			case s := <-g.toDieChan:
				g.toDieChan <- s
				return
			case m := <-inCh:
				if m == nil {
					return
				}
				outCh <- m.(*proto.SignedGossipMessage).GossipMessage
				log.Logger.Warnf("receive gossip message : %s", m)
			}
		}
	}()
	return outCh, nil
}

func selectOnlyDiscoveryMessages(m interface{}) bool {
	msg, isGossipMsg := m.(proto.ReceivedMessage)
	if !isGossipMsg {
		return false
	}
	alive := msg.GetGossipMessage().GetAliveMsg()
	memRes := msg.GetGossipMessage().GetMemRes()
	memReq := msg.GetGossipMessage().GetMemReq()

	selected := alive != nil || memReq != nil || memRes != nil

	return selected
}

func (g *gossipServiceImpl) newDiscoveryAdapter() *discoveryAdapter {
	return &discoveryAdapter{
		c:        g.comm,
		stopping: int32(0),
		gossipFunc: func(msg *proto.SignedGossipMessage) {
			if g.conf.PropagateIterations == 0 {
				return
			}
			g.emitter.Add(msg)
		},
		incChan:          make(chan *proto.SignedGossipMessage),
		presumedDead:     g.presumedDead,
		disclosurePolicy: g.disclosurePolicy,
	}
}

// discoveryAdapter is used to supply the discovery module with needed abilities
// that the comm interface in the discovery module declares
type discoveryAdapter struct {
	stopping         int32
	c                comm2.Comm
	presumedDead     chan common2.PKIidType
	incChan          chan *proto.SignedGossipMessage
	gossipFunc       func(message *proto.SignedGossipMessage)
	disclosurePolicy discovery2.DisclosurePolicy
}

func (da *discoveryAdapter) close() {
	atomic.StoreInt32(&da.stopping, int32(1))
	close(da.incChan)
}

func (da *discoveryAdapter) toDie() bool {
	return atomic.LoadInt32(&da.stopping) == int32(1)
}

func (da *discoveryAdapter) Gossip(msg *proto.SignedGossipMessage) {
	if da.toDie() {
		return
	}

	da.gossipFunc(msg)
}

func (da *discoveryAdapter) SendToPeer(peer *discovery2.NetworkMember, msg *proto.SignedGossipMessage) {
	if da.toDie() {
		return
	}
	// Check membership requests for peers that we know of their PKI-ID.
	// The only peers we don't know about their PKI-IDs are bootstrap peers.
	if memReq := msg.GetMemReq(); memReq != nil && len(peer.PKIid) != 0 {
		selfMsg, err := memReq.SelfInformation.ToGossipMessage()
		if err != nil {
			// Shouldn't happen
			panic("Tried to send a membership request with a malformed AliveMessage")
		}
		// Apply the EnvelopeFilter of the disclosure policy
		// on the alive message of the selfInfo field of the membership request
		_, omitConcealedFields := da.disclosurePolicy(peer)
		selfMsg.Envelope = omitConcealedFields(selfMsg)
		// Backup old known field
		oldKnown := memReq.Known
		// Override new SelfInfo message with updated envelope
		memReq = &proto.MembershipRequest{
			SelfInformation: selfMsg.Envelope,
			Known:           oldKnown,
		}
		// Update original message
		msg.Content = &proto.GossipMessage_MemReq{
			MemReq: memReq,
		}
		// Update the envelope of the outer message, no need to sign (point2point)
		msg, err = msg.NoopSign()
		if err != nil {
			return
		}
	}
	da.c.Send(msg, &comm2.RemotePeer{PKIID: peer.PKIid, Endpoint: peer.PreferredEndpoint()})
}

func (da *discoveryAdapter) Ping(peer *discovery2.NetworkMember) bool {
	err := da.c.Probe(&comm2.RemotePeer{Endpoint: peer.PreferredEndpoint(), PKIID: peer.PKIid})
	return err == nil
}

func (da *discoveryAdapter) Accept() <-chan *proto.SignedGossipMessage {
	return da.incChan
}

func (da *discoveryAdapter) PresumedDead() <-chan common2.PKIidType {
	return da.presumedDead
}

func (da *discoveryAdapter) CloseConn(peer *discovery2.NetworkMember) {
	da.c.CloseConn(&comm2.RemotePeer{PKIID: peer.PKIid})
}

type discoverySecurityAdapter struct {
	identity              api2.PeerIdentityType
	includeIdentityPeriod time.Time
	idMapper              identity2.Mapper
	sa                    api2.SecurityAdvisor
	mcs                   api2.MessageCryptoService
	c                     comm2.Comm
}

func (g *gossipServiceImpl) newDiscoverySecurityAdapter() *discoverySecurityAdapter {
	return &discoverySecurityAdapter{
		sa:                    g.secAdvisor,
		idMapper:              g.idMapper,
		mcs:                   g.mcs,
		c:                     g.comm,
		includeIdentityPeriod: g.includeIdentityPeriod,
		identity:              g.selfIdentity,
	}
}

// validateAliveMsg validates that an Alive message is authentic
func (sa *discoverySecurityAdapter) ValidateAliveMsg(m *proto.SignedGossipMessage) bool {
	am := m.GetAliveMsg()
	if am == nil || am.Membership == nil || am.Membership.PkiId == nil || !m.IsSigned() {
		log.Logger.Warn("Invalid alive message:", m)
		return false
	}

	var identity api2.PeerIdentityType

	// If identity is included inside AliveMessage
	if am.Identity != nil {
		identity = am.Identity
		claimedPKIID := am.Membership.PkiId
		err := sa.idMapper.Put(claimedPKIID, identity)
		if err != nil {
			log.Logger.Warn("Failed validating identity of", am, "reason:", err)
			return false
		}
	} else {
		identity, _ = sa.idMapper.Get(am.Membership.PkiId)
		if identity != nil {
			log.Logger.Debug("Fetched identity of", am.Membership.PkiId, "from identity store")
		}
	}

	if identity == nil {
		log.Logger.Debug("Don't have certificate for", am)
		return false
	}

	return sa.validateAliveMsgSignature(m, identity)
}

// SignMessage signs an AliveMessage and updates its signature field
func (sa *discoverySecurityAdapter) SignMessage(m *proto.GossipMessage, internalEndpoint string) *proto.Envelope {
	signer := func(msg []byte) ([]byte, error) {
		return sa.mcs.Sign(msg)
	}
	if m.IsAliveMsg() && time.Now().Before(sa.includeIdentityPeriod) {
		m.GetAliveMsg().Identity = sa.identity
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	e, err := sMsg.Sign(signer)
	if err != nil {
		log.Logger.Warn("Failed signing message:", err)
		return nil
	}

	if internalEndpoint == "" {
		return e
	}
	e.SignSecret(signer, &proto.Secret{
		Content: &proto.Secret_InternalEndpoint{
			InternalEndpoint: internalEndpoint,
		},
	})
	return e
}

func (sa *discoverySecurityAdapter) validateAliveMsgSignature(m *proto.SignedGossipMessage, identity api2.PeerIdentityType) bool {
	am := m.GetAliveMsg()
	// At this point we got the certificate of the peer, proceed to verifying the AliveMessage
	verifier := func(peerIdentity []byte, signature, message []byte) error {
		return sa.mcs.Verify(peerIdentity, signature, message)
	}

	// We verify the signature on the message
	err := m.Verify(identity, verifier)
	if err != nil {
		log.Logger.Warn("Failed verifying:", am, ":", err)
		return false
	}

	return true
}

func (g *gossipServiceImpl) createCertStorePuller() pull2.Mediator {
	conf := pull2.Config{
		MsgType:           proto.PullMsgType_IDENTITY_MSG,
		Channel:           []byte(""),
		ID:                g.conf.InternalEndpoint,
		PeerCountToSelect: g.conf.PullPeerNum,
		PullInterval:      g.conf.PullInterval,
		Tag:               proto.GossipMessage_EMPTY,
	}
	pkiIDFromMsg := func(msg *proto.SignedGossipMessage) string {
		identityMsg := msg.GetPeerIdentity()
		if identityMsg == nil || identityMsg.PkiId == nil {
			return ""
		}
		return fmt.Sprintf("%s", string(identityMsg.PkiId))
	}
	certConsumer := func(msg *proto.SignedGossipMessage) {
		idMsg := msg.GetPeerIdentity()
		if idMsg == nil || idMsg.Cert == nil || idMsg.PkiId == nil {
			log.Logger.Warn("Invalid PeerIdentity:", idMsg)
			return
		}
		err := g.idMapper.Put(idMsg.PkiId, idMsg.Cert)
		if err != nil {
			log.Logger.Warn("Failed associating PKI-ID with certificate:", err)
		}
		log.Logger.Info("Learned of a new certificate:", idMsg.Cert)
	}
	adapter := &pull2.PullAdapter{
		Sndr:        g.comm,
		MemSvc:      g.disc,
		IdExtractor: pkiIDFromMsg,
		MsgCons:     certConsumer,
		DigFilter:   g.sameOrgOrOurOrgPullFilter,
	}
	return pull2.NewPullMediator(conf, adapter)
}

func (g *gossipServiceImpl) sameOrgOrOurOrgPullFilter(msg proto.ReceivedMessage) func(string) bool {
	peersOrg := g.secAdvisor.OrgByPeerIdentity(msg.GetConnectionInfo().Identity)
	if len(peersOrg) == 0 {
		log.Logger.Warn("Failed determining organization of", msg.GetConnectionInfo())
		return func(_ string) bool {
			return false
		}
	}

	// If the peer is from our org, gossip all identities
	if bytes.Equal(g.selfOrg, peersOrg) {
		return func(_ string) bool {
			return true
		}
	}
	// Else, the peer is from a different org
	return func(item string) bool {
		pkiID := common2.PKIidType(item)
		msgsOrg := g.getOrgOfPeer(pkiID)
		if len(msgsOrg) == 0 {
			log.Logger.Warn("Failed determining organization of", pkiID)
			return false
		}
		// Don't gossip identities of dead peers or of peers
		// without external endpoints, to peers of foreign organizations.
		if !g.hasExternalEndpoint(pkiID) {
			return false
		}
		// Peer from our org or identity from our org or identity from peer's org
		return bytes.Equal(msgsOrg, g.selfOrg) || bytes.Equal(msgsOrg, peersOrg)
	}
}

func (g *gossipServiceImpl) connect2BootstrapPeers() {
	for _, endpoint := range g.conf.BootstrapPeers {
		endpoint := endpoint
		identifier := func() (*discovery2.PeerIdentification, error) {
			remotePeerIdentity, err := g.comm.Handshake(&comm2.RemotePeer{Endpoint: endpoint})
			if err != nil {
				return nil, err
			}
			sameOrg := bytes.Equal(g.selfOrg, g.secAdvisor.OrgByPeerIdentity(remotePeerIdentity))
			if !sameOrg {
				return nil, fmt.Errorf("%s isn't in our organization, cannot be a bootstrap peer", endpoint)
			}
			pkiID := g.mcs.GetPKIidOfCert(remotePeerIdentity)
			if len(pkiID) == 0 {
				return nil, fmt.Errorf("Wasn't able to extract PKI-ID of remote peer with identity of %v", remotePeerIdentity)
			}
			return &discovery2.PeerIdentification{ID: pkiID, SelfOrg: sameOrg}, nil
		}
		g.disc.Connect(discovery2.NetworkMember{
			InternalEndpoint: endpoint,
			Endpoint:         endpoint,
		}, identifier)
	}

}

func (g *gossipServiceImpl) createStateInfoMsg(metadata []byte, chainID common2.ChainID) (*proto.SignedGossipMessage, error) {
	pkiID := g.comm.GetPKIid()
	stateInfMsg := &proto.StateInfo{
		Channel_MAC: channel2.GenerateMAC(pkiID, chainID),
		Metadata:    metadata,
		PkiId:       g.comm.GetPKIid(),
		Timestamp: &proto.PeerTime{
			IncNum: uint64(g.incTime.UnixNano()),
			SeqNum: uint64(time.Now().UnixNano()),
		},
	}
	m := &proto.GossipMessage{
		Nonce: 0,
		Tag:   proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_StateInfo{
			StateInfo: stateInfMsg,
		},
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	signer := func(msg []byte) ([]byte, error) {
		return g.mcs.Sign(msg)
	}
	_, err := sMsg.Sign(signer)
	return sMsg, err
}

func (g *gossipServiceImpl) hasExternalEndpoint(PKIID common2.PKIidType) bool {
	if nm := g.disc.Lookup(PKIID); nm != nil {
		return nm.Endpoint != ""
	}
	return false
}

func (g *gossipServiceImpl) isInMyorg(member discovery2.NetworkMember) bool {
	if member.PKIid == nil {
		return false
	}
	if org := g.getOrgOfPeer(member.PKIid); org != nil {
		return bytes.Equal(g.selfOrg, org)
	}
	return false
}

func (g *gossipServiceImpl) getOrgOfPeer(PKIID common2.PKIidType) api2.OrgIdentityType {
	cert, err := g.idMapper.Get(PKIID)
	if err != nil {
		return nil
	}

	return g.secAdvisor.OrgByPeerIdentity(cert)
}

func (g *gossipServiceImpl) validateLeadershipMessage(msg *proto.SignedGossipMessage) error {
	pkiID := msg.GetLeadershipMsg().PkiId
	if len(pkiID) == 0 {
		return errors.New("Empty PKI-ID")
	}
	identity, err := g.idMapper.Get(pkiID)
	if err != nil {
		return fmt.Errorf("Unable to fetch PKI-ID from id-mapper: %v", err)
	}
	return msg.Verify(identity, func(peerIdentity []byte, signature, message []byte) error {
		return g.mcs.Verify(identity, signature, message)
	})
}

func (g *gossipServiceImpl) validateStateInfoMsg(msg *proto.SignedGossipMessage) error {
	verifier := func(identity []byte, signature, message []byte) error {
		pkiID := g.idMapper.GetPKIidOfCert(identity)
		if pkiID == nil {
			return errors.New("PKI-ID not found in identity mapper")
		}
		return g.idMapper.Verify(pkiID, signature, message)
	}
	identity, err := g.idMapper.Get(msg.GetStateInfo().PkiId)
	if err != nil {
		return err
	}
	return msg.Verify(identity, verifier)
}

func (g *gossipServiceImpl) disclosurePolicy(remotePeer *discovery2.NetworkMember) (discovery2.Sieve, discovery2.EnvelopeFilter) {
	remotePeerOrg := g.getOrgOfPeer(remotePeer.PKIid)

	if len(remotePeerOrg) == 0 {
		log.Logger.Warn("Cannot determine organization of", remotePeer)
		return func(msg *proto.SignedGossipMessage) bool {
				return false
			}, func(msg *proto.SignedGossipMessage) *proto.Envelope {
				return msg.Envelope
			}
	}

	return func(msg *proto.SignedGossipMessage) bool {
			if !msg.IsAliveMsg() {
				log.Logger.Panic("Programming error, this should be used only on alive messages")
			}
			org := g.getOrgOfPeer(msg.GetAliveMsg().Membership.PkiId)
			if len(org) == 0 {
				log.Logger.Warn("Unable to determine org of message", msg.GossipMessage)
				// Don't disseminate messages who's origin org is unknown
				return false
			}

			// Target org and the message are from the same org
			fromSameForeignOrg := bytes.Equal(remotePeerOrg, org)
			// The message is from my org
			fromMyOrg := bytes.Equal(g.selfOrg, org)
			// Forward to target org only messages from our org, or from the target org itself.
			if !(fromSameForeignOrg || fromMyOrg) {
				return false
			}

			// Pass the alive message only if the alive message is in the same org as the remote peer
			// or the message has an external endpoint, and the remote peer also has one
			return bytes.Equal(org, remotePeerOrg) || msg.GetAliveMsg().Membership.Endpoint != "" && remotePeer.Endpoint != ""
		}, func(msg *proto.SignedGossipMessage) *proto.Envelope {
			if !bytes.Equal(g.selfOrg, remotePeerOrg) {
				msg.SecretEnvelope = nil
			}
			return msg.Envelope
		}
}

func (g *gossipServiceImpl) peersByOriginOrgPolicy(peer discovery2.NetworkMember) filter2.RoutingFilter {
	peersOrg := g.getOrgOfPeer(peer.PKIid)
	if len(peersOrg) == 0 {
		log.Logger.Warn("Unable to determine organization of peer", peer)
		// Don't disseminate messages who's origin org is undetermined
		return filter2.SelectNonePolicy
	}

	if bytes.Equal(g.selfOrg, peersOrg) {
		// Disseminate messages from our org to all known organizations.
		// IMPORTANT: Currently a peer cannot un-join a channel, so the only way
		// of making gossip stop talking to an organization is by having the MSP
		// refuse validating messages from it.
		return filter2.SelectAllPolicy
	}

	// Else, select peers from the origin's organization,
	// and also peers from our own organization
	return func(member discovery2.NetworkMember) bool {
		memberOrg := g.getOrgOfPeer(member.PKIid)
		if len(memberOrg) == 0 {
			return false
		}
		isFromMyOrg := bytes.Equal(g.selfOrg, memberOrg)
		return isFromMyOrg || bytes.Equal(memberOrg, peersOrg)
	}
}

// partitionMessages receives a predicate and a slice of gossip messages
// and returns a tuple of two slices: the messages that hold for the predicate
// and the rest
func partitionMessages(pred common2.MessageAcceptor, a []*proto.SignedGossipMessage) ([]*proto.SignedGossipMessage, []*proto.SignedGossipMessage) {
	var s1 []*proto.SignedGossipMessage
	var s2 []*proto.SignedGossipMessage
	for _, m := range a {
		if pred(m) {
			s1 = append(s1, m)
		} else {
			s2 = append(s2, m)
		}
	}
	return s1, s2
}

// extractChannels returns a slice with all channels
// of all given GossipMessages
func extractChannels(a []*proto.SignedGossipMessage) []common2.ChainID {
	var channels []common2.ChainID
	for _, m := range a {
		if len(m.Channel) == 0 {
			continue
		}
		sameChan := func(a interface{}, b interface{}) bool {
			return bytes.Equal(a.(common2.ChainID), b.(common2.ChainID))
		}
		if util2.IndexInSlice(channels, common2.ChainID(m.Channel), sameChan) == -1 {
			channels = append(channels, m.Channel)
		}
	}
	return channels
}
