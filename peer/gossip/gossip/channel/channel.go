package channel

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	common_utils "github.com/rongzer/blockchain/common/util"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	comm2 "github.com/rongzer/blockchain/peer/gossip/comm"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	discovery2 "github.com/rongzer/blockchain/peer/gossip/discovery"
	election2 "github.com/rongzer/blockchain/peer/gossip/election"
	filter2 "github.com/rongzer/blockchain/peer/gossip/filter"
	msgstore2 "github.com/rongzer/blockchain/peer/gossip/gossip/msgstore"
	pull2 "github.com/rongzer/blockchain/peer/gossip/gossip/pull"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
	proto "github.com/rongzer/blockchain/protos/gossip"
)

// Config is a configuration item
// of the channel store
type Config struct {
	ID                          string
	PublishStateInfoInterval    time.Duration
	MaxBlockCountToStore        int
	PullPeerNum                 int
	PullInterval                time.Duration
	RequestStateInfoInterval    time.Duration
	BlockExpirationInterval     time.Duration
	StateInfoCacheSweepInterval time.Duration
}

// GossipChannel defines an object that deals with all channel-related messages
type GossipChannel interface {

	// GetPeers returns a list of peers with metadata as published by them
	GetPeers() []discovery2.NetworkMember

	// IsMemberInChan checks whether the given member is eligible to be in the channel
	IsMemberInChan(member discovery2.NetworkMember) bool

	// UpdateStateInfo updates this channel's StateInfo message
	// that is periodically published
	UpdateStateInfo(msg *proto.SignedGossipMessage)

	// IsOrgInChannel returns whether the given organization is in the channel
	IsOrgInChannel(membersOrg api2.OrgIdentityType) bool

	// EligibleForChannel returns whether the given member should get blocks
	// for this channel
	EligibleForChannel(member discovery2.NetworkMember) bool

	// HandleMessage processes a message sent by a remote peer
	HandleMessage(proto.ReceivedMessage)

	// AddToMsgStore adds a given GossipMessage to the message store
	AddToMsgStore(msg *proto.SignedGossipMessage)

	// ConfigureChannel (re)configures the list of organizations
	// that are eligible to be in the channel
	ConfigureChannel(joinMsg api2.JoinChannelMessage)

	// Stop stops the channel's activity
	Stop()
}

// Adapter enables the gossipChannel
// to communicate with gossipServiceImpl.
type Adapter interface {
	// GetConf returns the configuration that this GossipChannel will posses
	GetConf() Config

	// Gossip gossips a message in the channel
	Gossip(message *proto.SignedGossipMessage)

	// DeMultiplex de-multiplexes an item to subscribers
	DeMultiplex(interface{})

	// GetMembership returns the known alive peers and their information
	GetMembership() []discovery2.NetworkMember

	// Lookup returns a network member, or nil if not found
	Lookup(PKIID common2.PKIidType) *discovery2.NetworkMember

	// Send sends a message to a list of peers
	Send(msg *proto.SignedGossipMessage, peers ...*comm2.RemotePeer)

	// ValidateStateInfoMessage returns an error if a message
	// hasn't been signed correctly, nil otherwise.
	ValidateStateInfoMessage(message *proto.SignedGossipMessage) error

	// GetOrgOfPeer returns the organization ID of a given peer PKI-ID
	GetOrgOfPeer(pkiID common2.PKIidType) api2.OrgIdentityType

	// GetIdentityByPKIID returns an identity of a peer with a certain
	// pkiID, or nil if not found
	GetIdentityByPKIID(pkiID common2.PKIidType) api2.PeerIdentityType
}

type gossipChannel struct {
	Adapter
	sync.RWMutex
	shouldGossipStateInfo     int32
	mcs                       api2.MessageCryptoService
	pkiID                     common2.PKIidType
	selfOrg                   api2.OrgIdentityType
	stopChan                  chan struct{}
	stateInfoMsg              *proto.SignedGossipMessage
	orgs                      []api2.OrgIdentityType
	joinMsg                   api2.JoinChannelMessage
	blockMsgStore             msgstore2.MessageStore
	stateInfoMsgStore         *stateInfoCache
	leaderMsgStore            msgstore2.MessageStore
	chainID                   common2.ChainID
	blocksPuller              pull2.Mediator
	stateInfoPublishScheduler *time.Ticker
	stateInfoRequestScheduler *time.Ticker
	memFilter                 *membershipFilter
}

type membershipFilter struct {
	adapter Adapter
	*gossipChannel
}

// GetMembership returns the known alive peers and their information
func (mf *membershipFilter) GetMembership() []discovery2.NetworkMember {
	var members []discovery2.NetworkMember
	for _, mem := range mf.adapter.GetMembership() {
		if mf.eligibleForChannelAndSameOrg(mem) {
			members = append(members, mem)
		}
	}
	return members
}

// NewGossipChannel creates a new GossipChannel
func NewGossipChannel(pkiID common2.PKIidType, org api2.OrgIdentityType, mcs api2.MessageCryptoService,
	chainID common2.ChainID, adapter Adapter, joinMsg api2.JoinChannelMessage) GossipChannel {
	gc := &gossipChannel{
		selfOrg:                   org,
		pkiID:                     pkiID,
		mcs:                       mcs,
		Adapter:                   adapter,
		stopChan:                  make(chan struct{}, 1),
		shouldGossipStateInfo:     int32(0),
		stateInfoPublishScheduler: time.NewTicker(adapter.GetConf().PublishStateInfoInterval),
		stateInfoRequestScheduler: time.NewTicker(adapter.GetConf().RequestStateInfoInterval),
		orgs:                      []api2.OrgIdentityType{},
		chainID:                   chainID,
	}

	gc.memFilter = &membershipFilter{adapter: gc.Adapter, gossipChannel: gc}

	comparator := proto.NewGossipMessageComparator(adapter.GetConf().MaxBlockCountToStore)

	gc.blocksPuller = gc.createBlockPuller()

	seqNumFromMsg := func(m interface{}) string {
		return fmt.Sprintf("%d", m.(*proto.SignedGossipMessage).GetDataMsg().Payload.SeqNum)
	}
	gc.blockMsgStore = msgstore2.NewMessageStoreExpirable(comparator, func(m interface{}) {
		gc.blocksPuller.Remove(seqNumFromMsg(m))
	}, gc.GetConf().BlockExpirationInterval, nil, nil, func(m interface{}) {
		gc.blocksPuller.Remove(seqNumFromMsg(m))
	})

	hashPeerExpiredInMembership := func(o interface{}) bool {
		pkiID := o.(*proto.SignedGossipMessage).GetStateInfo().PkiId
		return gc.Lookup(pkiID) == nil
	}
	gc.stateInfoMsgStore = newStateInfoCache(gc.GetConf().StateInfoCacheSweepInterval, hashPeerExpiredInMembership)

	ttl := election2.GetMsgExpirationTimeout()
	pol := proto.NewGossipMessageComparator(0)

	gc.leaderMsgStore = msgstore2.NewMessageStoreExpirable(pol, msgstore2.Noop, ttl, nil, nil, nil)

	gc.ConfigureChannel(joinMsg)

	// Periodically publish state info
	go gc.periodicalInvocation(gc.publishStateInfo, gc.stateInfoPublishScheduler.C)
	// Periodically request state info
	go gc.periodicalInvocation(gc.requestStateInfo, gc.stateInfoRequestScheduler.C)
	return gc
}

// Stop stop the channel operations
func (gc *gossipChannel) Stop() {
	gc.stopChan <- struct{}{}
	gc.blocksPuller.Stop()
	gc.stateInfoPublishScheduler.Stop()
	gc.stateInfoRequestScheduler.Stop()
	gc.leaderMsgStore.Stop()
	gc.stateInfoMsgStore.Stop()
	gc.blockMsgStore.Stop()
}

func (gc *gossipChannel) periodicalInvocation(fn func(), c <-chan time.Time) {
	for {
		select {
		case <-c:
			fn()
		case <-gc.stopChan:
			gc.stopChan <- struct{}{}
			return
		}
	}
}

// GetPeers returns a list of peers with metadata as published by them
func (gc *gossipChannel) GetPeers() []discovery2.NetworkMember {
	var members []discovery2.NetworkMember

	for _, member := range gc.GetMembership() {
		if !gc.EligibleForChannel(member) {
			continue
		}
		stateInf := gc.stateInfoMsgStore.MsgByID(member.PKIid)
		if stateInf == nil {
			continue
		}
		member.Metadata = stateInf.GetStateInfo().Metadata
		members = append(members, member)
	}
	return members
}

func (gc *gossipChannel) requestStateInfo() {
	req, err := gc.createStateInfoRequest()
	if err != nil {
		log.Logger.Warn("Failed creating SignedGossipMessage:", err)
		return
	}
	endpoints := filter2.SelectPeers(gc.GetConf().PullPeerNum, gc.GetMembership(), gc.IsMemberInChan)
	gc.Send(req, endpoints...)
}

func (gc *gossipChannel) eligibleForChannelAndSameOrg(member discovery2.NetworkMember) bool {
	sameOrg := func(networkMember discovery2.NetworkMember) bool {
		return bytes.Equal(gc.GetOrgOfPeer(networkMember.PKIid), gc.selfOrg)
	}
	return filter2.CombineRoutingFilters(gc.EligibleForChannel, sameOrg)(member)
}

func (gc *gossipChannel) publishStateInfo() {
	if atomic.LoadInt32(&gc.shouldGossipStateInfo) == int32(0) {
		return
	}

	//log.Logger.Warnf("publish state info : %s", gc.stateInfoMsg)

	gc.RLock()
	stateInfoMsg := gc.stateInfoMsg
	gc.RUnlock()
	gc.Gossip(stateInfoMsg)
	if len(gc.GetMembership()) > 0 {
		atomic.StoreInt32(&gc.shouldGossipStateInfo, int32(0))
	} else {
		//log.Logger.Errorf("no membership can get faltal err")
	}
}

func (gc *gossipChannel) createBlockPuller() pull2.Mediator {
	conf := pull2.Config{
		MsgType:           proto.PullMsgType_BLOCK_MSG,
		Channel:           []byte(gc.chainID),
		ID:                gc.GetConf().ID,
		PeerCountToSelect: gc.GetConf().PullPeerNum,
		PullInterval:      gc.GetConf().PullInterval,
		Tag:               proto.GossipMessage_CHAN_AND_ORG,
	}
	seqNumFromMsg := func(msg *proto.SignedGossipMessage) string {
		dataMsg := msg.GetDataMsg()
		if dataMsg == nil || dataMsg.Payload == nil {
			log.Logger.Warn("Non-data block or with no payload")
			return ""
		}
		return fmt.Sprintf("%d", dataMsg.Payload.SeqNum)
	}
	adapter := &pull2.PullAdapter{
		Sndr:        gc,
		MemSvc:      gc.memFilter,
		IdExtractor: seqNumFromMsg,
		MsgCons: func(msg *proto.SignedGossipMessage) {
			gc.DeMultiplex(msg)
		},
	}
	return pull2.NewPullMediator(conf, adapter)
}

// IsMemberInChan checks whether the given member is eligible to be in the channel
func (gc *gossipChannel) IsMemberInChan(member discovery2.NetworkMember) bool {
	org := gc.GetOrgOfPeer(member.PKIid)
	if org == nil {
		return false
	}

	return gc.IsOrgInChannel(org)
}

// IsOrgInChannel returns whether the given organization is in the channel
func (gc *gossipChannel) IsOrgInChannel(membersOrg api2.OrgIdentityType) bool {
	gc.RLock()
	defer gc.RUnlock()
	for _, orgOfChan := range gc.orgs {
		if bytes.Equal(orgOfChan, membersOrg) {
			return true
		}
	}
	return false
}

// EligibleForChannel returns whether the given member should get blocks
// for this channel
func (gc *gossipChannel) EligibleForChannel(member discovery2.NetworkMember) bool {
	if !gc.IsMemberInChan(member) {
		return false
	}

	identity := gc.GetIdentityByPKIID(member.PKIid)
	msg := gc.stateInfoMsgStore.MsgByID(member.PKIid)
	if msg == nil || identity == nil {
		return false
	}

	return gc.mcs.VerifyByChannel(gc.chainID, identity, msg.Envelope.Signature, msg.Envelope.Payload) == nil
}

// AddToMsgStore adds a given GossipMessage to the message store
func (gc *gossipChannel) AddToMsgStore(msg *proto.SignedGossipMessage) {
	if msg.IsDataMsg() {
		gc.blockMsgStore.Add(msg)
		gc.blocksPuller.Add(msg)
	}

	if msg.IsStateInfoMsg() {
		gc.stateInfoMsgStore.Add(msg)
	}
}

// ConfigureChannel (re)configures the list of organizations
// that are eligible to be in the channel
func (gc *gossipChannel) ConfigureChannel(joinMsg api2.JoinChannelMessage) {
	gc.Lock()
	defer gc.Unlock()

	if len(joinMsg.Members()) == 0 {
		log.Logger.Warn("Received join channel message with empty set of members")
		return
	}

	if gc.joinMsg == nil {
		gc.joinMsg = joinMsg
	}

	if gc.joinMsg.SequenceNumber() > (joinMsg.SequenceNumber()) {
		log.Logger.Warn("Already have a more updated JoinChannel message(", gc.joinMsg.SequenceNumber(), ") than", gc.joinMsg.SequenceNumber())
		return
	}

	gc.orgs = joinMsg.Members()
	gc.joinMsg = joinMsg
}

// HandleMessage processes a message sent by a remote peer
func (gc *gossipChannel) HandleMessage(msg proto.ReceivedMessage) {

	//log.Logger.Warnf("gossip channel handle msg : %s", msg)

	if !gc.verifyMsg(msg) {
		log.Logger.Warn("Failed verifying message:", msg.GetGossipMessage().GossipMessage)
		return
	}
	m := msg.GetGossipMessage()
	if !m.IsChannelRestricted() {
		log.Logger.Warn("Got message", msg.GetGossipMessage(), "but it's not a per-channel message, discarding it")
		return
	}
	orgID := gc.GetOrgOfPeer(msg.GetConnectionInfo().ID)
	if len(orgID) == 0 {
		log.Logger.Debug("Couldn't find org identity of peer", msg.GetConnectionInfo())
		return
	}
	if !gc.IsOrgInChannel(orgID) {
		log.Logger.Warn("Point to point message came from", msg.GetConnectionInfo(),
			", org(", string(orgID), ") but it's not eligible for the channel", string(gc.chainID))
		return
	}

	if m.IsStateInfoPullRequestMsg() {
		msg.Respond(gc.createStateInfoSnapshot(orgID))
		return
	}

	if m.IsStateInfoSnapshot() {
		gc.handleStateInfSnapshot(m.GossipMessage, msg.GetConnectionInfo().ID)
		return
	}

	if m.IsDataMsg() || m.IsStateInfoMsg() {
		added := false

		if m.IsDataMsg() {
			if m.GetDataMsg().Payload == nil {
				log.Logger.Warn("Payload is empty, got it from", msg.GetConnectionInfo().ID)
				return
			}
			// Would this block go into the message store if it was verified?
			if !gc.blockMsgStore.CheckValid(msg.GetGossipMessage()) {
				return
			}
			if !gc.verifyBlock(m.GossipMessage, msg.GetConnectionInfo().ID) {
				log.Logger.Warn("Failed verifying block", m.GetDataMsg().Payload.SeqNum)
				return
			}
			added = gc.blockMsgStore.Add(msg.GetGossipMessage())
		} else { // StateInfoMsg verification should be handled in a layer above
			//  since we don't have access to the id mapper here
			added = gc.stateInfoMsgStore.Add(msg.GetGossipMessage())
			log.Logger.Infof("handle stateinfo from peer : %s", msg.GetConnectionInfo())
		}

		if added {
			// Forward the message
			gc.Gossip(msg.GetGossipMessage())
			// DeMultiplex to local subscribers
			gc.DeMultiplex(m)

			if m.IsDataMsg() {
				gc.blocksPuller.Add(msg.GetGossipMessage())
				log.Logger.Infof("blocksPuller add msg")
			}
		}
		return
	}
	if m.IsPullMsg() && m.GetPullMsgType() == proto.PullMsgType_BLOCK_MSG {
		// If we don't have a StateInfo message from the peer,
		// no way of validating its eligibility in the channel.
		if gc.stateInfoMsgStore.MsgByID(msg.GetConnectionInfo().ID) == nil {
			log.Logger.Debug("Don't have StateInfo message of peer", msg.GetConnectionInfo())
			return
		}
		if !gc.eligibleForChannelAndSameOrg(discovery2.NetworkMember{PKIid: msg.GetConnectionInfo().ID}) {
			log.Logger.Warn(msg.GetConnectionInfo(), "isn't eligible for pulling blocks of", string(gc.chainID))
			return
		}
		if m.IsDataUpdate() {
			// Iterate over the envelopes, and filter out blocks
			// that we already have in the blockMsgStore, or blocks that
			// are too far in the past.
			var filteredEnvelopes []*proto.Envelope
			for _, item := range m.GetDataUpdate().Data {
				gMsg, err := item.ToGossipMessage()
				if err != nil {
					log.Logger.Warn("Data update contains an invalid message:", err)
					return
				}
				if !bytes.Equal(gMsg.Channel, gc.chainID) {
					log.Logger.Warn("DataUpdate message contains item with channel", gMsg.Channel, "but should be", gc.chainID)
					return
				}
				// Would this block go into the message store if it was verified?
				if !gc.blockMsgStore.CheckValid(msg.GetGossipMessage()) {
					return
				}
				if !gc.verifyBlock(gMsg.GossipMessage, msg.GetConnectionInfo().ID) {
					return
				}
				added := gc.blockMsgStore.Add(gMsg)
				if !added {
					// If this block doesn't need to be added, it means it either already
					// exists in memory or that it is too far in the past
					continue
				}
				filteredEnvelopes = append(filteredEnvelopes, item)
			}
			// Replace the update message with just the blocks that should be processed
			m.GetDataUpdate().Data = filteredEnvelopes
		}
		gc.blocksPuller.HandleMessage(msg)
	}

	if m.IsLeadershipMsg() {
		// Handling leadership message
		added := gc.leaderMsgStore.Add(m)
		if added {
			gc.DeMultiplex(m)
		}
	}
}

func (gc *gossipChannel) handleStateInfSnapshot(m *proto.GossipMessage, sender common2.PKIidType) {
	chanName := string(gc.chainID)
	for _, envelope := range m.GetStateSnapshot().Elements {
		stateInf, err := envelope.ToGossipMessage()
		if err != nil {
			log.Logger.Warn("Channel", chanName, ": StateInfo snapshot contains an invalid message:", err)
			return
		}
		if !stateInf.IsStateInfoMsg() {
			log.Logger.Warn("Channel", chanName, ": Element of StateInfoSnapshot isn't a StateInfoMessage:",
				stateInf, "message sent from", sender)
			return
		}
		si := stateInf.GetStateInfo()
		orgID := gc.GetOrgOfPeer(si.PkiId)
		if orgID == nil {
			log.Logger.Debug("Channel", chanName, ": Couldn't find org identity of peer",
				string(si.PkiId), "message sent from", string(sender))
			return
		}

		if !gc.IsOrgInChannel(orgID) {
			log.Logger.Warn("Channel", chanName, ": Peer", stateInf.GetStateInfo().PkiId,
				"is not in an eligible org, can't process a stateInfo from it, sent from", sender)
			return
		}

		expectedMAC := GenerateMAC(si.PkiId, gc.chainID)
		if !bytes.Equal(si.Channel_MAC, expectedMAC) {
			log.Logger.Warn("Channel", chanName, ": StateInfo message", stateInf,
				", has an invalid MAC. Expected", expectedMAC, ", got", si.Channel_MAC, ", sent from", sender)
			return
		}
		err = gc.ValidateStateInfoMessage(stateInf)
		if err != nil {
			log.Logger.Warn("Channel", chanName, ": Failed validating state info message:",
				stateInf, ":", err, "sent from", sender)
			return
		}

		if gc.Lookup(si.PkiId) == nil {
			// Skip StateInfo messages that belong to peers
			// that have been expired
			continue
		}

		gc.stateInfoMsgStore.Add(stateInf)
	}
}

func (gc *gossipChannel) verifyBlock(msg *proto.GossipMessage, sender common2.PKIidType) bool {
	if !msg.IsDataMsg() {
		log.Logger.Warn("Received from ", sender, "a DataUpdate message that contains a non-block GossipMessage:", msg)
		return false
	}
	payload := msg.GetDataMsg().Payload
	if payload == nil {
		log.Logger.Warn("Received empty payload from", sender)
		return false
	}
	seqNum := payload.SeqNum
	rawBlock := payload.Data
	err := gc.mcs.VerifyBlock(msg.Channel, seqNum, rawBlock)
	if err != nil {
		log.Logger.Warn("Received blockchainated block from", sender, "in DataUpdate:", err)
		return false
	}
	return true
}

func (gc *gossipChannel) createStateInfoSnapshot(requestersOrg api2.OrgIdentityType) *proto.GossipMessage {
	sameOrg := bytes.Equal(gc.selfOrg, requestersOrg)
	rawElements := gc.stateInfoMsgStore.Get()
	var elements []*proto.Envelope
	for _, rawEl := range rawElements {
		msg := rawEl.(*proto.SignedGossipMessage)
		orgOfCurrentMsg := gc.GetOrgOfPeer(msg.GetStateInfo().PkiId)
		// If we're in the same org as the requester, or the message belongs to a foreign org
		// don't do any filtering
		if sameOrg || !bytes.Equal(orgOfCurrentMsg, gc.selfOrg) {
			elements = append(elements, msg.Envelope)
			continue
		}
		// Else, the requester is in a different org, so disclose only StateInfo messages that their
		// corresponding AliveMessages have external endpoints
		if netMember := gc.Lookup(msg.GetStateInfo().PkiId); netMember == nil || netMember.Endpoint == "" {
			continue
		}
		elements = append(elements, msg.Envelope)
	}

	return &proto.GossipMessage{
		Channel: gc.chainID,
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Nonce:   0,
		Content: &proto.GossipMessage_StateSnapshot{
			StateSnapshot: &proto.StateInfoSnapshot{
				Elements: elements,
			},
		},
	}
}

func (gc *gossipChannel) verifyMsg(msg proto.ReceivedMessage) bool {
	if msg == nil {
		log.Logger.Warn("Messsage is nil")
		return false
	}
	m := msg.GetGossipMessage()
	if m == nil {
		log.Logger.Warn("Message content is empty")
		return false
	}

	if msg.GetConnectionInfo().ID == nil {
		log.Logger.Warn("Message has nil PKI-ID")
		return false
	}

	if m.IsStateInfoMsg() {
		si := m.GetStateInfo()
		expectedMAC := GenerateMAC(si.PkiId, gc.chainID)
		if !bytes.Equal(expectedMAC, si.Channel_MAC) {
			log.Logger.Warn("Message contains wrong channel MAC(", si.Channel_MAC, "), expected", expectedMAC)
			return false
		}
		return true
	}

	if m.IsStateInfoPullRequestMsg() {
		sipr := m.GetStateInfoPullReq()
		expectedMAC := GenerateMAC(msg.GetConnectionInfo().ID, gc.chainID)
		if !bytes.Equal(expectedMAC, sipr.Channel_MAC) {
			log.Logger.Warn("Message contains wrong channel MAC(", sipr.Channel_MAC, "), expected", expectedMAC)
			return false
		}
		return true
	}

	if !bytes.Equal(m.Channel, gc.chainID) {
		log.Logger.Warn("Message contains wrong channel(", m.Channel, "), expected", gc.chainID)
		return false
	}
	return true
}

func (gc *gossipChannel) createStateInfoRequest() (*proto.SignedGossipMessage, error) {
	return (&proto.GossipMessage{
		Tag:   proto.GossipMessage_CHAN_OR_ORG,
		Nonce: 0,
		Content: &proto.GossipMessage_StateInfoPullReq{
			StateInfoPullReq: &proto.StateInfoPullRequest{
				Channel_MAC: GenerateMAC(gc.pkiID, gc.chainID),
			},
		},
	}).NoopSign()
}

// UpdateStateInfo updates this channel's StateInfo message
// that is periodically published
func (gc *gossipChannel) UpdateStateInfo(msg *proto.SignedGossipMessage) {
	if !msg.IsStateInfoMsg() {
		return
	}
	gc.stateInfoMsgStore.Add(msg)
	gc.Lock()
	defer gc.Unlock()
	gc.stateInfoMsg = msg
	atomic.StoreInt32(&gc.shouldGossipStateInfo, int32(1))
}

func newStateInfoCache(sweepInterval time.Duration, hasExpired func(interface{}) bool) *stateInfoCache {
	membershipStore := util2.NewMembershipStore()
	pol := proto.NewGossipMessageComparator(0)
	s := &stateInfoCache{
		MembershipStore: membershipStore,
		stopChan:        make(chan struct{}),
	}
	invalidationTrigger := func(m interface{}) {
		pkiID := m.(*proto.SignedGossipMessage).GetStateInfo().PkiId
		membershipStore.Remove(pkiID)
	}
	s.MessageStore = msgstore2.NewMessageStore(pol, invalidationTrigger)

	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			case <-time.After(sweepInterval):
				s.Purge(hasExpired)
			}
		}
	}()
	return s
}

// stateInfoCache is actually a messageStore
// that also indexes messages that are added
// so that they could be extracted later
type stateInfoCache struct {
	*util2.MembershipStore
	msgstore2.MessageStore
	stopChan chan struct{}
}

// Add attempts to add the given message to the stateInfoCache,
// and if the message was added, also indexes it.
// Message must be a StateInfo message.
func (cache *stateInfoCache) Add(msg *proto.SignedGossipMessage) bool {
	added := cache.MessageStore.Add(msg)
	if added {
		pkiID := msg.GetStateInfo().PkiId
		cache.MembershipStore.Put(pkiID, msg)
	}
	return added
}

func (cache *stateInfoCache) Stop() {
	cache.stopChan <- struct{}{}
}

// GenerateMAC returns a byte slice that is derived from the peer's PKI-ID
// and a channel name
func GenerateMAC(pkiID common2.PKIidType, channelID common2.ChainID) []byte {
	// Hash is computed on (PKI-ID || channel ID)
	preImage := append(pkiID, []byte(channelID)...)
	return common_utils.WriteAndSumSha256(preImage)
}
