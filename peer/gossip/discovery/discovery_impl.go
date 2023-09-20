package discovery

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	util2 "github.com/rongzer/blockchain/common/util"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	msgstore2 "github.com/rongzer/blockchain/peer/gossip/gossip/msgstore"
	util3 "github.com/rongzer/blockchain/peer/gossip/util"
	proto "github.com/rongzer/blockchain/protos/gossip"
)

const defaultHelloInterval = time.Duration(5) * time.Second
const msgExpirationFactor = 20

var aliveExpirationCheckInterval time.Duration
var maxConnectionAttempts = 120

type timestamp struct {
	incTime  time.Time
	seqNum   uint64
	lastSeen time.Time
}

func (ts *timestamp) String() string {
	return fmt.Sprintf("%v, %v", ts.incTime.UnixNano(), ts.seqNum)
}

type gossipDiscoveryImpl struct {
	incTime         uint64
	seqNum          uint64
	self            NetworkMember
	deadLastTS      map[string]*timestamp     // H
	aliveLastTS     map[string]*timestamp     // V
	id2Member       map[string]*NetworkMember // all known members
	aliveMembership *util3.MembershipStore
	deadMembership  *util3.MembershipStore

	msgStore *aliveMsgStore

	comm  CommService
	crypt CryptoService
	lock  *sync.RWMutex

	toDieChan        chan struct{}
	toDieFlag        int32
	port             int
	disclosurePolicy DisclosurePolicy
	pubsub           *util3.PubSub
}

// NewDiscoveryService returns a new discovery service with the comm module passed and the crypto service passed
func NewDiscoveryService(self NetworkMember, comm CommService, crypt CryptoService, disPol DisclosurePolicy) Discovery {
	d := &gossipDiscoveryImpl{
		self:             self,
		incTime:          uint64(time.Now().UnixNano()),
		seqNum:           uint64(0),
		deadLastTS:       make(map[string]*timestamp),
		aliveLastTS:      make(map[string]*timestamp),
		id2Member:        make(map[string]*NetworkMember),
		aliveMembership:  util3.NewMembershipStore(),
		deadMembership:   util3.NewMembershipStore(),
		crypt:            crypt,
		comm:             comm,
		lock:             &sync.RWMutex{},
		toDieChan:        make(chan struct{}, 1),
		toDieFlag:        int32(0),
		disclosurePolicy: disPol,
		pubsub:           util3.NewPubSub(),
	}

	d.validateSelfConfig()
	d.msgStore = newAliveMsgStore(d)

	go d.periodicalSendAlive()
	go d.periodicalCheckAlive()
	go d.handleMessages()
	go d.periodicalReconnectToDead()
	go d.handlePresumedDeadPeers()

	log.Logger.Info("Started", self, "incTime is", d.incTime)

	return d
}

// Lookup returns a network member, or nil if not found
func (d *gossipDiscoveryImpl) Lookup(PKIID common2.PKIidType) *NetworkMember {
	if bytes.Equal(PKIID, d.self.PKIid) {
		return &d.self
	}
	d.lock.RLock()
	defer d.lock.RUnlock()
	nm := d.id2Member[util2.BytesToString(PKIID)]
	return nm
}

func (d *gossipDiscoveryImpl) Connect(member NetworkMember, id identifier) {
	for _, endpoint := range []string{member.InternalEndpoint, member.Endpoint} {
		if d.isMyOwnEndpoint(endpoint) {
			log.Logger.Debug("Skipping connecting to myself")
			return
		}
	}

	log.Logger.Debug("Entering", member)
	defer log.Logger.Debug("Exiting")
	go func() {
		for i := 0; i < maxConnectionAttempts && !d.toDie(); i++ {
			id, err := id()
			if err != nil {
				if d.toDie() {
					return
				}
				log.Logger.Warn("Could not connect to", member, ":", err)
				time.Sleep(getReconnectInterval())
				continue
			}
			peer := &NetworkMember{
				InternalEndpoint: member.InternalEndpoint,
				Endpoint:         member.Endpoint,
				PKIid:            id.ID,
			}
			m, err := d.createMembershipRequest(id.SelfOrg)
			if err != nil {
				log.Logger.Warn("Failed creating membership request:", err)
				continue
			}
			req, err := m.NoopSign()
			if err != nil {
				log.Logger.Warn("Failed creating SignedGossipMessage:", err)
				continue
			}
			req.Nonce = util3.RandomUInt64()
			req, err = req.NoopSign()
			if err != nil {
				log.Logger.Warn("Failed adding NONCE to SignedGossipMessage", err)
				continue
			}
			go d.sendUntilAcked(peer, req)
			log.Logger.Warnf("send alivemsg to : %s msg : %s", peer, req)
			return
		}

	}()
}

func (d *gossipDiscoveryImpl) isMyOwnEndpoint(endpoint string) bool {
	return endpoint == fmt.Sprintf("127.0.0.1:%d", d.port) || endpoint == fmt.Sprintf("localhost:%d", d.port) ||
		endpoint == d.self.InternalEndpoint || endpoint == d.self.Endpoint
}

func (d *gossipDiscoveryImpl) validateSelfConfig() {
	endpoint := d.self.InternalEndpoint
	if len(endpoint) == 0 {
		log.Logger.Panic("Internal endpoint is empty:", endpoint)
	}

	internalEndpointSplit := strings.Split(endpoint, ":")
	if len(internalEndpointSplit) != 2 {
		log.Logger.Panicf("Self endpoint %s isn't formatted as 'host:port'", endpoint)
	}
	myPort, err := strconv.ParseInt(internalEndpointSplit[1], 10, 64)
	if err != nil {
		log.Logger.Panicf("Self endpoint %s has not valid port'", endpoint)
	}

	if myPort > int64(math.MaxUint16) {
		log.Logger.Panicf("Self endpoint %s's port takes more than 16 bits", endpoint)
	}

	d.port = int(myPort)
}

func (d *gossipDiscoveryImpl) sendUntilAcked(peer *NetworkMember, message *proto.SignedGossipMessage) {
	nonce := message.Nonce
	for i := 0; i < maxConnectionAttempts && !d.toDie(); i++ {
		sub := d.pubsub.Subscribe(fmt.Sprintf("%d", nonce), time.Second*5)
		d.comm.SendToPeer(peer, message)
		if _, timeoutErr := sub.Listen(); timeoutErr == nil {
			return
		}
		time.Sleep(getReconnectInterval())
	}
}

func (d *gossipDiscoveryImpl) InitiateSync(peerNum int) {
	if d.toDie() {
		return
	}
	var peers2SendTo []*NetworkMember
	m, err := d.createMembershipRequest(true)
	if err != nil {
		log.Logger.Warn("Failed creating membership request:", err)
		return
	}
	memReq, err := m.NoopSign()
	if err != nil {
		log.Logger.Warn("Failed creating SignedGossipMessage:", err)
		return
	}
	d.lock.RLock()

	n := d.aliveMembership.Size()
	k := peerNum
	if k > n {
		k = n
	}

	aliveMembersAsSlice := d.aliveMembership.ToSlice()
	for _, i := range util3.GetRandomIndices(k, n-1) {
		pulledPeer := aliveMembersAsSlice[i].GetAliveMsg().Membership
		var internalEndpoint string
		if aliveMembersAsSlice[i].Envelope.SecretEnvelope != nil {
			internalEndpoint = aliveMembersAsSlice[i].Envelope.SecretEnvelope.InternalEndpoint()
		}
		netMember := &NetworkMember{
			Endpoint:         pulledPeer.Endpoint,
			Metadata:         pulledPeer.Metadata,
			PKIid:            pulledPeer.PkiId,
			InternalEndpoint: internalEndpoint,
		}
		peers2SendTo = append(peers2SendTo, netMember)
	}

	d.lock.RUnlock()

	for _, netMember := range peers2SendTo {
		d.comm.SendToPeer(netMember, memReq)
	}
}

func (d *gossipDiscoveryImpl) handlePresumedDeadPeers() {
	defer log.Logger.Debug("Stopped")

	for !d.toDie() {
		select {
		case deadPeer := <-d.comm.PresumedDead():
			if d.isAlive(deadPeer) {
				d.expireDeadMembers([]common2.PKIidType{deadPeer})
			}
		case s := <-d.toDieChan:
			d.toDieChan <- s
			return
		}
	}
}

func (d *gossipDiscoveryImpl) isAlive(pkiID common2.PKIidType) bool {
	d.lock.RLock()
	defer d.lock.RUnlock()
	_, alive := d.aliveLastTS[util2.BytesToString(pkiID)]
	return alive
}

func (d *gossipDiscoveryImpl) handleMessages() {
	defer log.Logger.Debug("Stopped")

	in := d.comm.Accept()
	for !d.toDie() {
		select {
		case s := <-d.toDieChan:
			d.toDieChan <- s
			return
		case m := <-in:
			d.handleMsgFromComm(m)
		}
	}
}

func (d *gossipDiscoveryImpl) handleMsgFromComm(m *proto.SignedGossipMessage) {
	if m == nil {
		return
	}
	if m.GetAliveMsg() == nil && m.GetMemRes() == nil && m.GetMemReq() == nil {
		log.Logger.Warn("Got message with wrong type (expected Alive or MembershipResponse or MembershipRequest message):", m.GossipMessage)
		return
	}

	log.Logger.Debug("Got message:", m)
	defer log.Logger.Debug("Exiting")

	if memReq := m.GetMemReq(); memReq != nil {
		selfInfoGossipMsg, err := memReq.SelfInformation.ToGossipMessage()
		if err != nil {
			log.Logger.Warn("Failed deserializing GossipMessage from envelope:", err)
			return
		}

		if d.msgStore.CheckValid(selfInfoGossipMsg) {
			d.handleAliveMessage(selfInfoGossipMsg)
		}

		var internalEndpoint string
		if m.Envelope.SecretEnvelope != nil {
			internalEndpoint = m.Envelope.SecretEnvelope.InternalEndpoint()
		}

		// Sending a membership response to a peer may block this routine
		// in case the sending is deliberately slow (i.e attack).
		// will keep this async until I'll write a timeout detector in the comm layer
		go d.sendMemResponse(selfInfoGossipMsg.GetAliveMsg().Membership, internalEndpoint, m.Nonce)

		//log.Logger.Warnf("receive memreq msg : %s", m)
		return
	}

	if m.IsAliveMsg() {

		if !d.msgStore.Add(m) {
			return
		}
		d.handleAliveMessage(m)

		d.comm.Gossip(m)
		return
	}

	if memResp := m.GetMemRes(); memResp != nil {
		d.pubsub.Publish(fmt.Sprintf("%d", m.Nonce), m.Nonce)
		for _, env := range memResp.Alive {
			am, err := env.ToGossipMessage()
			if err != nil {
				log.Logger.Warn("Membership response contains an invalid message from an online peer:", err)
				return
			}
			if !am.IsAliveMsg() {
				log.Logger.Warn("Expected alive message, got", am, "instead")
				return
			}

			if d.msgStore.CheckValid(am) {
				d.handleAliveMessage(am)
			}
		}

		for _, env := range memResp.Dead {
			dm, err := env.ToGossipMessage()
			if err != nil {
				log.Logger.Warn("Membership response contains an invalid message from an offline peer", err)
				return
			}
			if !d.crypt.ValidateAliveMsg(dm) {
				log.Logger.Debugf("Alive message isn't authentic, someone spoofed %s's identity", dm.GetAliveMsg().Membership)
				continue
			}

			if !d.msgStore.CheckValid(dm) {
				//Newer alive message exist
				return
			}

			var newDeadMembers []*proto.SignedGossipMessage
			d.lock.RLock()
			if _, known := d.id2Member[util2.BytesToString(dm.GetAliveMsg().Membership.PkiId)]; !known {
				newDeadMembers = append(newDeadMembers, dm)
			}
			d.lock.RUnlock()
			d.learnNewMembers([]*proto.SignedGossipMessage{}, newDeadMembers)
		}
	}
}

func (d *gossipDiscoveryImpl) sendMemResponse(targetMember *proto.Member, internalEndpoint string, nonce uint64) {
	log.Logger.Debug("Entering", targetMember)

	targetPeer := &NetworkMember{
		Endpoint:         targetMember.Endpoint,
		Metadata:         targetMember.Metadata,
		PKIid:            targetMember.PkiId,
		InternalEndpoint: internalEndpoint,
	}

	aliveMsg, err := d.createAliveMessage(true)
	if err != nil {
		log.Logger.Warn("Failed creating alive message:", err)
		return
	}
	memResp := d.createMembershipResponse(aliveMsg, targetPeer)
	if memResp == nil {
		errMsg := `Got a membership request from a peer that shouldn't have sent one: %v, closing connection to the peer as a result.`
		log.Logger.Warnf(errMsg, targetMember)
		d.comm.CloseConn(targetPeer)
		return
	}

	defer log.Logger.Debug("Exiting, replying with", memResp)

	msg, err := (&proto.GossipMessage{
		Tag:   proto.GossipMessage_EMPTY,
		Nonce: nonce,
		Content: &proto.GossipMessage_MemRes{
			MemRes: memResp,
		},
	}).NoopSign()
	if err != nil {
		log.Logger.Warn("Failed creating SignedGossipMessage:", err)
		return
	}
	d.comm.SendToPeer(targetPeer, msg)
}

func (d *gossipDiscoveryImpl) createMembershipResponse(aliveMsg *proto.SignedGossipMessage, targetMember *NetworkMember) *proto.MembershipResponse {
	shouldBeDisclosed, omitConcealedFields := d.disclosurePolicy(targetMember)

	if !shouldBeDisclosed(aliveMsg) {
		return nil
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	var deadPeers []*proto.Envelope

	for _, dm := range d.deadMembership.ToSlice() {

		if !shouldBeDisclosed(dm) {
			continue
		}
		deadPeers = append(deadPeers, omitConcealedFields(dm))
	}

	var aliveSnapshot []*proto.Envelope
	for _, am := range d.aliveMembership.ToSlice() {
		if !shouldBeDisclosed(am) {
			continue
		}
		aliveSnapshot = append(aliveSnapshot, omitConcealedFields(am))
	}

	return &proto.MembershipResponse{
		Alive: append(aliveSnapshot, omitConcealedFields(aliveMsg)),
		Dead:  deadPeers,
	}
}

func (d *gossipDiscoveryImpl) handleAliveMessage(m *proto.SignedGossipMessage) {
	log.Logger.Debug("Entering", m)
	defer log.Logger.Debug("Exiting")

	if !d.crypt.ValidateAliveMsg(m) {
		log.Logger.Debugf("Alive message isn't authentic, someone must be spoofing %s's identity", m.GetAliveMsg())
		return
	}

	pkiID := m.GetAliveMsg().Membership.PkiId
	if equalPKIid(pkiID, d.self.PKIid) {
		log.Logger.Debug("Got alive message about ourselves,", m)
		diffExternalEndpoint := d.self.Endpoint != m.GetAliveMsg().Membership.Endpoint
		var diffInternalEndpoint bool
		secretEnvelope := m.GetSecretEnvelope()
		if secretEnvelope != nil && secretEnvelope.InternalEndpoint() != "" {
			diffInternalEndpoint = secretEnvelope.InternalEndpoint() != d.self.InternalEndpoint
		}
		if diffInternalEndpoint || diffExternalEndpoint {
			log.Logger.Error("Bad configuration detected: Received AliveMessage from a peer with the same PKI-ID as myself:", m.GossipMessage)
		}

		return
	}

	ts := m.GetAliveMsg().Timestamp

	pkid := util2.BytesToString(pkiID)
	d.lock.RLock()
	_, known := d.id2Member[pkid]
	d.lock.RUnlock()

	if !known {
		d.learnNewMembers([]*proto.SignedGossipMessage{m}, []*proto.SignedGossipMessage{})
		return
	}

	d.lock.RLock()
	_, isAlive := d.aliveLastTS[pkid]
	lastDeadTS, isDead := d.deadLastTS[pkid]
	d.lock.RUnlock()

	if !isAlive && !isDead {
		log.Logger.Panicf("Member %s is known but not found neither in alive nor in dead lastTS maps, isAlive=%v, isDead=%v", m.GetAliveMsg().Membership.Endpoint, isAlive, isDead)
		return
	}

	if isAlive && isDead {
		log.Logger.Panicf("Member %s is both alive and dead at the same time", m.GetAliveMsg().Membership)
		return
	}

	if isDead {
		if before(lastDeadTS, ts) {
			// resurrect peer
			d.resurrectMember(m, *ts)
		} else if !same(lastDeadTS, ts) {
			log.Logger.Debug(m.GetAliveMsg().Membership, "lastDeadTS:", lastDeadTS, "but got ts:", ts)
		}
		return
	}

	d.lock.RLock()
	lastAliveTS, isAlive := d.aliveLastTS[pkid]
	d.lock.RUnlock()

	if isAlive {
		if before(lastAliveTS, ts) {
			d.learnExistingMembers([]*proto.SignedGossipMessage{m})
		} else if !same(lastAliveTS, ts) {
			log.Logger.Debug(m.GetAliveMsg().Membership, "lastAliveTS:", lastAliveTS, "but got ts:", ts)
		}

	}
	// else, ignore the message because it is too old
}

func (d *gossipDiscoveryImpl) resurrectMember(am *proto.SignedGossipMessage, t proto.PeerTime) {
	log.Logger.Debug("Entering, AliveMessage:", am, "t:", t)
	defer log.Logger.Debug("Exiting")
	d.lock.Lock()
	defer d.lock.Unlock()

	member := am.GetAliveMsg().Membership
	pkiID := member.PkiId
	pkid := util2.BytesToString(pkiID)
	d.aliveLastTS[pkid] = &timestamp{
		lastSeen: time.Now(),
		seqNum:   t.SeqNum,
		incTime:  tsToTime(t.IncNum),
	}

	var internalEndpoint string
	if prevNetMem := d.id2Member[pkid]; prevNetMem != nil {
		internalEndpoint = prevNetMem.InternalEndpoint
	}
	if am.Envelope.SecretEnvelope != nil {
		internalEndpoint = am.Envelope.SecretEnvelope.InternalEndpoint()
	}

	d.id2Member[pkid] = &NetworkMember{
		Endpoint:         member.Endpoint,
		Metadata:         member.Metadata,
		PKIid:            member.PkiId,
		InternalEndpoint: internalEndpoint,
	}

	delete(d.deadLastTS, pkid)
	d.deadMembership.Remove(pkiID)
	d.aliveMembership.Put(pkiID, &proto.SignedGossipMessage{GossipMessage: am.GossipMessage, Envelope: am.Envelope})
}

func (d *gossipDiscoveryImpl) periodicalReconnectToDead() {
	defer log.Logger.Debug("Stopped")

	for !d.toDie() {
		wg := &sync.WaitGroup{}
		//判断死节点是否是活的
		liveMems := d.aliveMembership.ToSlice()
		for _, member := range d.copyLastSeen(d.deadLastTS) {
			isLive := false
			for _, liveMem := range liveMems {
				if member.Endpoint == liveMem.GetAliveMsg().GetMembership().Endpoint {
					isLive = true
					continue
				}
			}
			if isLive {
				continue
			}

			wg.Add(1)
			go func(member NetworkMember) {
				defer wg.Done()
				if d.comm.Ping(&member) {
					log.Logger.Debug(member, "is responding, sending membership request")
					d.sendMembershipRequest(&member, true)
					log.Logger.Debug("sendMembershipRequest with interval : %d", getReconnectInterval())
				} else {
					log.Logger.Debug(member, "is still dead")
				}
			}(member)
		}

		wg.Wait()
		log.Logger.Debug("Sleeping", getReconnectInterval())
		time.Sleep(getReconnectInterval())
	}
}

func (d *gossipDiscoveryImpl) sendMembershipRequest(member *NetworkMember, includeInternalEndpoint bool) {
	m, err := d.createMembershipRequest(includeInternalEndpoint)
	if err != nil {
		log.Logger.Warn("Failed creating membership request:", err)
		return
	}
	req, err := m.NoopSign()
	if err != nil {
		log.Logger.Error("Failed creating SignedGossipMessage:", err)
		return
	}
	d.comm.SendToPeer(member, req)
}

func (d *gossipDiscoveryImpl) createMembershipRequest(includeInternalEndpoint bool) (*proto.GossipMessage, error) {
	am, err := d.createAliveMessage(includeInternalEndpoint)
	if err != nil {
		return nil, err
	}
	req := &proto.MembershipRequest{
		SelfInformation: am.Envelope,
		// TODO: sending the known peers is not secure because the remote peer might shouldn't know
		// TODO: about the known peers. I'm deprecating this until a secure mechanism will be implemented.
		// TODO: See FAB-2570 for tracking this issue.
		Known: [][]byte{},
	}
	return &proto.GossipMessage{
		Tag:   proto.GossipMessage_EMPTY,
		Nonce: uint64(0),
		Content: &proto.GossipMessage_MemReq{
			MemReq: req,
		},
	}, nil
}

func (d *gossipDiscoveryImpl) copyLastSeen(lastSeenMap map[string]*timestamp) []NetworkMember {
	d.lock.RLock()
	defer d.lock.RUnlock()

	var res []NetworkMember
	for pkiIDStr := range lastSeenMap {
		res = append(res, *(d.id2Member[pkiIDStr]))
	}
	return res
}

func (d *gossipDiscoveryImpl) periodicalCheckAlive() {
	defer log.Logger.Debug("Stopped")

	for !d.toDie() {
		time.Sleep(getAliveExpirationCheckInterval())
		dead := d.getDeadMembers()
		if len(dead) > 0 {
			log.Logger.Debugf("Got %v dead members: %v", len(dead), dead)
			d.expireDeadMembers(dead)
		}
	}
}

func (d *gossipDiscoveryImpl) expireDeadMembers(dead []common2.PKIidType) {
	log.Logger.Debug("Entering", dead)
	defer log.Logger.Debug("Exiting")

	var deadMembers2Expire []*NetworkMember

	d.lock.Lock()

	for _, pkiID := range dead {
		pkid := util2.BytesToString(pkiID)
		if _, isAlive := d.aliveLastTS[pkid]; !isAlive {
			continue
		}
		deadMembers2Expire = append(deadMembers2Expire, d.id2Member[pkid])
		// move lastTS from alive to dead
		lastTS, hasLastTS := d.aliveLastTS[pkid]
		if hasLastTS {
			d.deadLastTS[pkid] = lastTS
			delete(d.aliveLastTS, pkid)
		}

		if am := d.aliveMembership.MsgByID(pkiID); am != nil {
			d.deadMembership.Put(pkiID, am)
			d.aliveMembership.Remove(pkiID)
		}
	}

	d.lock.Unlock()

	for _, member2Expire := range deadMembers2Expire {
		log.Logger.Warn("Closing connection to ", member2Expire)
		//d.comm.CloseConn(member2Expire)
	}
}

func (d *gossipDiscoveryImpl) getDeadMembers() []common2.PKIidType {
	d.lock.RLock()
	defer d.lock.RUnlock()

	var dead []common2.PKIidType
	for id, last := range d.aliveLastTS {
		elapsedNonAliveTime := time.Since(last.lastSeen)
		if elapsedNonAliveTime.Nanoseconds() > getAliveExpirationTimeout().Nanoseconds() {
			log.Logger.Warn("Haven't heard from ", []byte(id), " for ", elapsedNonAliveTime)
			dead = append(dead, common2.PKIidType(id))
		}
	}
	return dead
}

func (d *gossipDiscoveryImpl) periodicalSendAlive() {
	defer log.Logger.Debug("Stopped")

	for !d.toDie() {
		log.Logger.Debug("Sleeping", getAliveTimeInterval())
		time.Sleep(getAliveTimeInterval())
		msg, err := d.createAliveMessage(true)
		if err != nil {
			log.Logger.Warn("Failed creating alive message:", err)
			return
		}
		d.comm.Gossip(msg)
	}
}

func (d *gossipDiscoveryImpl) createAliveMessage(includeInternalEndpoint bool) (*proto.SignedGossipMessage, error) {
	d.lock.Lock()
	d.seqNum++
	seqNum := d.seqNum

	endpoint := d.self.Endpoint
	meta := d.self.Metadata
	pkiID := d.self.PKIid
	internalEndpoint := d.self.InternalEndpoint

	d.lock.Unlock()

	msg2Gossip := &proto.GossipMessage{
		Tag: proto.GossipMessage_EMPTY,
		Content: &proto.GossipMessage_AliveMsg{
			AliveMsg: &proto.AliveMessage{
				Membership: &proto.Member{
					Endpoint: endpoint,
					Metadata: meta,
					PkiId:    pkiID,
				},
				Timestamp: &proto.PeerTime{
					IncNum: d.incTime,
					SeqNum: seqNum,
				},
			},
		},
	}

	envp := d.crypt.SignMessage(msg2Gossip, internalEndpoint)
	if envp == nil {
		return nil, errors.New("Failed signing message")
	}
	signedMsg := &proto.SignedGossipMessage{
		GossipMessage: msg2Gossip,
		Envelope:      envp,
	}

	if !includeInternalEndpoint {
		signedMsg.Envelope.SecretEnvelope = nil
	}

	return signedMsg, nil
}

func (d *gossipDiscoveryImpl) learnExistingMembers(aliveArr []*proto.SignedGossipMessage) {
	log.Logger.Debugf("Entering: learnedMembers={%v}", aliveArr)
	defer log.Logger.Debug("Exiting")

	d.lock.Lock()
	defer d.lock.Unlock()

	for _, m := range aliveArr {
		am := m.GetAliveMsg()
		if m == nil {
			log.Logger.Warn("Expected alive message, got instead:", m)
			return
		}
		log.Logger.Debug("updating", am)

		var internalEndpoint string
		pkid := util2.BytesToString(am.Membership.PkiId)
		if prevNetMem := d.id2Member[pkid]; prevNetMem != nil {
			internalEndpoint = prevNetMem.InternalEndpoint
		}
		if m.Envelope.SecretEnvelope != nil {
			internalEndpoint = m.Envelope.SecretEnvelope.InternalEndpoint()
		}

		// update member's data
		member := d.id2Member[pkid]
		member.Endpoint = am.Membership.Endpoint
		member.Metadata = am.Membership.Metadata
		member.InternalEndpoint = internalEndpoint

		if _, isKnownAsDead := d.deadLastTS[pkid]; isKnownAsDead {
			log.Logger.Warn(am.Membership, " has already expired")
			continue
		}

		if _, isKnownAsAlive := d.aliveLastTS[pkid]; !isKnownAsAlive {
			log.Logger.Warn(am.Membership, " has already expired")
			continue
		} else {
			log.Logger.Debug("Updating aliveness data:", am)
			// update existing aliveness data
			alive := d.aliveLastTS[pkid]
			alive.incTime = tsToTime(am.Timestamp.IncNum)
			alive.lastSeen = time.Now()
			alive.seqNum = am.Timestamp.SeqNum

			if am := d.aliveMembership.MsgByID(m.GetAliveMsg().Membership.PkiId); am == nil {
				log.Logger.Debug("Adding", am, "to aliveMembership")
				msg := &proto.SignedGossipMessage{GossipMessage: m.GossipMessage, Envelope: am.Envelope}
				d.aliveMembership.Put(m.GetAliveMsg().Membership.PkiId, msg)
			} else {
				log.Logger.Debug("Replacing", am, "in aliveMembership")
				am.GossipMessage = m.GossipMessage
				am.Envelope = m.Envelope
			}
		}
	}
}

func (d *gossipDiscoveryImpl) learnNewMembers(aliveMembers []*proto.SignedGossipMessage, deadMembers []*proto.SignedGossipMessage) {
	log.Logger.Debugf("Entering: learnedMembers={%v}, deadMembers={%v}", aliveMembers, deadMembers)
	defer log.Logger.Debugf("Exiting")

	d.lock.Lock()
	defer d.lock.Unlock()

	for _, am := range aliveMembers {
		if equalPKIid(am.GetAliveMsg().Membership.PkiId, d.self.PKIid) {
			continue
		}
		d.aliveLastTS[util2.BytesToString(am.GetAliveMsg().Membership.PkiId)] = &timestamp{
			incTime:  tsToTime(am.GetAliveMsg().Timestamp.IncNum),
			lastSeen: time.Now(),
			seqNum:   am.GetAliveMsg().Timestamp.SeqNum,
		}

		d.aliveMembership.Put(am.GetAliveMsg().Membership.PkiId, &proto.SignedGossipMessage{GossipMessage: am.GossipMessage, Envelope: am.Envelope})
		log.Logger.Debugf("Learned about a new alive member: %v", am)
	}

	for _, dm := range deadMembers {
		if equalPKIid(dm.GetAliveMsg().Membership.PkiId, d.self.PKIid) {
			continue
		}
		d.deadLastTS[util2.BytesToString(dm.GetAliveMsg().Membership.PkiId)] = &timestamp{
			incTime:  tsToTime(dm.GetAliveMsg().Timestamp.IncNum),
			lastSeen: time.Now(),
			seqNum:   dm.GetAliveMsg().Timestamp.SeqNum,
		}

		d.deadMembership.Put(dm.GetAliveMsg().Membership.PkiId, &proto.SignedGossipMessage{GossipMessage: dm.GossipMessage, Envelope: dm.Envelope})
		log.Logger.Debugf("Learned about a new dead member: %v", dm)
	}

	// update the member in any case
	for _, a := range [][]*proto.SignedGossipMessage{aliveMembers, deadMembers} {
		for _, m := range a {
			member := m.GetAliveMsg()
			if member == nil {
				log.Logger.Warn("Expected alive message, got instead:", m)
				return
			}

			var internalEndpoint string
			if m.Envelope.SecretEnvelope != nil {
				internalEndpoint = m.Envelope.SecretEnvelope.InternalEndpoint()
			}

			pkid := util2.BytesToString(member.Membership.PkiId)
			if prevNetMem := d.id2Member[pkid]; prevNetMem != nil {
				internalEndpoint = prevNetMem.InternalEndpoint
			}

			d.id2Member[pkid] = &NetworkMember{
				Endpoint:         member.Membership.Endpoint,
				Metadata:         member.Membership.Metadata,
				PKIid:            member.Membership.PkiId,
				InternalEndpoint: internalEndpoint,
			}
		}
	}

	// 用alive的ip加端口循环删除死节点
	liveMems := d.aliveMembership.ToSlice()
	deadMems := d.deadMembership.ToSlice()
	for _, liveMem := range liveMems {
		for _, deadMem := range deadMems {
			if deadMem.GetAliveMsg().GetMembership().Endpoint == liveMem.GetAliveMsg().GetMembership().Endpoint {
				log.Logger.Warnf("remove Dead Member %v , %s\n", deadMem.GetAliveMsg().Membership.PkiId, deadMem.GetAliveMsg().GetMembership().GetEndpoint())
				d.deadMembership.Remove(deadMem.GetAliveMsg().GetMembership().PkiId)
			}
		}
	}
}

func (d *gossipDiscoveryImpl) GetMembership() []NetworkMember {
	if d.toDie() {
		return []NetworkMember{}
	}
	d.lock.RLock()
	defer d.lock.RUnlock()

	var response []NetworkMember
	for _, m := range d.aliveMembership.ToSlice() {
		member := m.GetAliveMsg()
		response = append(response, NetworkMember{
			PKIid:            member.Membership.PkiId,
			Endpoint:         member.Membership.Endpoint,
			Metadata:         member.Membership.Metadata,
			InternalEndpoint: d.id2Member[util2.BytesToString(m.GetAliveMsg().Membership.PkiId)].InternalEndpoint,
		})
	}
	return response

}

func tsToTime(ts uint64) time.Time {
	return time.Unix(int64(0), int64(ts))
}

func (d *gossipDiscoveryImpl) UpdateMetadata(md []byte) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.self.Metadata = md
}

func (d *gossipDiscoveryImpl) UpdateEndpoint(endpoint string) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.self.Endpoint = endpoint
}

func (d *gossipDiscoveryImpl) Self() NetworkMember {
	return NetworkMember{
		Endpoint:         d.self.Endpoint,
		Metadata:         d.self.Metadata,
		PKIid:            d.self.PKIid,
		InternalEndpoint: d.self.InternalEndpoint,
	}
}

func (d *gossipDiscoveryImpl) toDie() bool {
	toDie := atomic.LoadInt32(&d.toDieFlag) == int32(1)
	return toDie
}

func (d *gossipDiscoveryImpl) Stop() {
	defer log.Logger.Info("Stopped")
	log.Logger.Info("Stopping")
	atomic.StoreInt32(&d.toDieFlag, int32(1))
	d.msgStore.Stop()
	d.toDieChan <- struct{}{}
}

func equalPKIid(a, b common2.PKIidType) bool {
	return bytes.Equal(a, b)
}

func same(a *timestamp, b *proto.PeerTime) bool {
	return uint64(a.incTime.UnixNano()) == b.IncNum && a.seqNum == b.SeqNum
}

func before(a *timestamp, b *proto.PeerTime) bool {
	return (uint64(a.incTime.UnixNano()) == b.IncNum && a.seqNum < b.SeqNum) ||
		uint64(a.incTime.UnixNano()) < b.IncNum
}

func getAliveTimeInterval() time.Duration {
	return util3.GetDurationOrDefault("core.peer.gossip.aliveTimeInterval", defaultHelloInterval)
}

func getAliveExpirationTimeout() time.Duration {
	return util3.GetDurationOrDefault("core.peer.gossip.aliveExpirationTimeout", 5*getAliveTimeInterval())
}

func getAliveExpirationCheckInterval() time.Duration {
	if aliveExpirationCheckInterval != 0 {
		return aliveExpirationCheckInterval
	}

	return getAliveExpirationTimeout() / 10
}

func getReconnectInterval() time.Duration {
	return util3.GetDurationOrDefault("core.peer.gossip.reconnectInterval", getAliveExpirationTimeout())
}

type aliveMsgStore struct {
	msgstore2.MessageStore
}

func newAliveMsgStore(d *gossipDiscoveryImpl) *aliveMsgStore {
	policy := proto.NewGossipMessageComparator(0)
	trigger := func(m interface{}) {}
	aliveMsgTTL := getAliveExpirationTimeout() * msgExpirationFactor
	externalLock := func() { d.lock.Lock() }
	externalUnlock := func() { d.lock.Unlock() }
	callback := func(m interface{}) {
		msg := m.(*proto.SignedGossipMessage)
		if !msg.IsAliveMsg() {
			return
		}
		id := msg.GetAliveMsg().Membership.PkiId
		d.aliveMembership.Remove(id)
		d.deadMembership.Remove(id)

		_id := util2.BytesToString(id)
		delete(d.id2Member, _id)
		delete(d.deadLastTS, _id)
		delete(d.aliveLastTS, _id)
	}

	s := &aliveMsgStore{
		MessageStore: msgstore2.NewMessageStoreExpirable(policy, trigger, aliveMsgTTL, externalLock, externalUnlock, callback),
	}
	return s
}

func (s *aliveMsgStore) Add(msg interface{}) bool {
	if !msg.(*proto.SignedGossipMessage).IsAliveMsg() {
		panic(fmt.Sprint("Msg ", msg, " is not AliveMsg"))
	}
	return s.MessageStore.Add(msg)
}

func (s *aliveMsgStore) CheckValid(msg interface{}) bool {
	if !msg.(*proto.SignedGossipMessage).IsAliveMsg() {
		panic(fmt.Sprint("Msg ", msg, " is not AliveMsg"))
	}
	return s.MessageStore.CheckValid(msg)
}
