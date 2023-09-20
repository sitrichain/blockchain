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

package light

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rongzer/blockchain/protos/utils"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/committer"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	comm2 "github.com/rongzer/blockchain/peer/gossip/comm"
	common3 "github.com/rongzer/blockchain/peer/gossip/common"
	discovery2 "github.com/rongzer/blockchain/peer/gossip/discovery"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
	"github.com/rongzer/blockchain/protos/common"
	proto "github.com/rongzer/blockchain/protos/gossip"
	"github.com/spf13/viper"
)

// GossipStateProvider is the interface to acquire sequences of the ledger blocks
// capable to full fill missing blocks by running state replication and
// sending request to get missing block to other nodes
type GossipStateProvider interface {
	// Retrieve block with sequence number equal to index
	GetBlock(index uint64) *common.Block

	AddPayload(payload *proto.Payload) error

	// Stop terminates state transfer object
	Stop()

	// 获取缓存数
	GetPayloadSize() int
}

const (
	defAntiEntropyInterval             = 10 * time.Second
	defAntiEntropyStateResponseTimeout = 8 * time.Second
	defAntiEntropyBatchSize            = 10

	defChannelBufferSize     = 100
	defAntiEntropyMaxRetries = 3
)

// GossipAdapter defines gossip/communication required interface for state provider
type GossipAdapter interface {
	// Send sends a message to remote peers
	Send(msg *proto.GossipMessage, peers ...*comm2.RemotePeer)

	// Accept returns a dedicated read-only channel for messages sent by other nodes that match a certain predicate.
	// If passThrough is false, the messages are processed by the gossip layer beforehand.
	// If passThrough is true, the gossip layer doesn't intervene and the messages
	// can be used to send a reply back to the sender
	Accept(acceptor common3.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage)

	// UpdateChannelMetadata updates the self metadata the peer
	// publishes to other peers about its channel-related state
	UpdateChannelMetadata(metadata []byte, chainID common3.ChainID)

	// PeersOfChannel returns the NetworkMembers considered alive
	// and also subscribed to the channel given
	PeersOfChannel(common3.ChainID) []discovery2.NetworkMember
}

// GossipStateProviderImpl the implementation of the GossipStateProvider interface
// the struct to handle in memory sliding window of
// new ledger block to be acquired by hyper ledger
type GossipStateProviderImpl struct {
	// MessageCryptoService
	mcs api2.MessageCryptoService

	// Chain id
	chainID string

	// The gossiping service
	gossip GossipAdapter

	// Channel to read gossip messages from
	gossipChan <-chan *proto.GossipMessage

	commChan <-chan proto.ReceivedMessage

	// Queue of payloads which wasn't acquired yet
	payloads PayloadsBuffer

	committer committer.Committer

	stateResponseCh chan proto.ReceivedMessage

	stateRequestCh chan proto.ReceivedMessage

	stopCh chan struct{}

	done sync.WaitGroup

	once sync.Once

	stateTransferActive int32

	//轻量账本存储交易的来源会员No
	fromMembers []string
}

// NewGossipStateProvider creates initialized instance of gossip state provider
func NewGossipStateProvider(chainID string, g GossipAdapter, committer committer.Committer, mcs api2.MessageCryptoService) GossipStateProvider {
	gossipChan, _ := g.Accept(func(message interface{}) bool {
		// Get only data messages
		return message.(*proto.GossipMessage).IsDataMsg() &&
			bytes.Equal(message.(*proto.GossipMessage).Channel, []byte(chainID))
	}, false)

	remoteStateMsgFilter := func(message interface{}) bool {
		receivedMsg := message.(proto.ReceivedMessage)
		msg := receivedMsg.GetGossipMessage()
		if !msg.IsRemoteStateMessage() {
			return false
		}
		// If we're not running with authentication, no point
		// in enforcing access control
		if !receivedMsg.GetConnectionInfo().IsAuthenticated() {
			return true
		}
		connInfo := receivedMsg.GetConnectionInfo()
		authErr := mcs.VerifyByChannel(msg.Channel, connInfo.Identity, connInfo.Auth.Signature, connInfo.Auth.SignedData)
		if authErr != nil {
			log.Logger.Warn("Got unauthorized nodeMetastate transfer request from", string(connInfo.Identity))
			return false
		}
		return true
	}

	// Filter message which are only relevant for nodeMetastate transfer
	_, commChan := g.Accept(remoteStateMsgFilter, true)

	height, err := committer.LedgerHeight()
	if height == 0 {
		// Panic here since this is an indication of invalid situation which should not happen in normal
		// code path.
		log.Logger.Panic("Committer height cannot be zero, ledger should include at least one block (genesis).")
	}

	if err != nil {
		log.Logger.Error("Could not read ledger info to obtain current ledger height due to: ", err)
		// Exiting as without ledger it will be impossible
		// to deliver new blocks
		return nil
	}
	var fromMembers []string
	// 从环境变量中读取轻量账本需要存储的交易的来源会员的证书
	fromMembers = viper.GetStringSlice("peer.gossip.fromMembers")
	if len(fromMembers) == 0 {
		log.Logger.Error("there must be peer.gossip.fromMembers configured in environment variables")
		return nil
	}

	s := &GossipStateProviderImpl{
		// MessageCryptoService
		mcs: mcs,

		// Chain ID
		chainID: chainID,

		// Instance of the gossip
		gossip: g,

		// Channel to read new messages from
		gossipChan: gossipChan,

		// Channel to read direct messages from other peers
		commChan: commChan,

		// Create a queue for payload received
		payloads: NewPayloadsBuffer(height),

		committer: committer,

		stateResponseCh: make(chan proto.ReceivedMessage, defChannelBufferSize),

		stateRequestCh: make(chan proto.ReceivedMessage, defChannelBufferSize),

		stopCh: make(chan struct{}, 1),

		stateTransferActive: 0,

		once: sync.Once{},

		fromMembers: fromMembers,
	}
	// 轻量节点的nodeMetastate始终保持为0
	nodeMetastate := NewNodeMetastate(0)

	log.Logger.Infof("Updating node metadata information, "+
		"current ledger sequence is at = %d, next expected block is = %d", nodeMetastate.LedgerHeight, s.payloads.Next())

	b, err := nodeMetastate.Bytes()
	if err == nil {
		log.Logger.Warnf("Updating gossip metadate. Ledger height: %d", nodeMetastate.LedgerHeight)
		g.UpdateChannelMetadata(b, common3.ChainID(s.chainID))
	} else {
		log.Logger.Errorf("Unable to serialize node meta nodeMetastate, error = %s", err)
	}

	s.done.Add(3)

	// Listen for incoming communication
	go s.listen()
	// Deliver in order messages into the incoming channel
	go s.deliverPayloads()
	// Execute anti entropy to fill missing gaps
	go s.antiEntropy()

	return s
}

func (s *GossipStateProviderImpl) listen() {
	defer s.done.Done()

	for {
		select {
		case msg := <-s.gossipChan:
			log.Logger.Warnf("Received new message via gossip channel")
			go s.queueNewMessage(msg)
		case msg := <-s.commChan:
			log.Logger.Warnf("Direct message ", msg)
			go s.directMessage(msg)
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			log.Logger.Warnf("Stop listening for new messages")
			return
		}
	}
}

func (s *GossipStateProviderImpl) directMessage(msg proto.ReceivedMessage) {
	log.Logger.Debug("[ENTER] -> directMessage")
	defer log.Logger.Debug("[EXIT] ->  directMessage")

	if msg == nil {
		log.Logger.Error("Got nil message via end-to-end channel, should not happen!")
		return
	}

	if !bytes.Equal(msg.GetGossipMessage().Channel, []byte(s.chainID)) {
		log.Logger.Warn("Received state transfer request for channel",
			string(msg.GetGossipMessage().Channel), "while expecting channel", s.chainID, "skipping request...")
		return
	}

	incoming := msg.GetGossipMessage()

	if incoming.GetStateResponse() != nil {
		log.Logger.Warnf("receive get state response : %s stateTransferActive : %d", msg, s.stateTransferActive)
		// If no state transfer procedure activate there is
		// no reason to process the message
		if atomic.LoadInt32(&s.stateTransferActive) == 1 {
			// Send signal of state response message
			s.stateResponseCh <- msg
		}
	}
}

func (s *GossipStateProviderImpl) handleStateResponse(msg proto.ReceivedMessage) (uint64, error) {

	log.Logger.Warnf("handleStateResponse peerinfo : %s : %s", msg.GetConnectionInfo(), msg)

	max := uint64(0)
	// Send signal that response for given nonce has been received
	response := msg.GetGossipMessage().GetStateResponse()
	// Extract payloads, verify and push into buffer
	if len(response.GetPayloads()) == 0 {
		return uint64(0), errors.New("Received state tranfer response without payload")
	}

	for i, payload := range response.GetPayloads() {
		log.Logger.Debugf("Received payload with sequence number %d.", payload.SeqNum)
		if err := s.mcs.VerifyBlock(common3.ChainID(s.chainID), payload.SeqNum, payload.Data); err != nil {
			log.Logger.Warnf("Error verifying block with sequence number %d, due to %s", payload.SeqNum, err)
			return uint64(0), err
		}

		block, err := utils.GetBlockFromBlockBytes(payload.Data)
		if err != nil {
			return uint64(0), err
		}

		payloads := response.GetPayloads()
		// 第一个无条件接受, 第二个块开始做hash检测
		if payload.SeqNum == 1 {
			log.Logger.Warnf("Received payload with sequence number %d.", payload.SeqNum)
			if max < payload.SeqNum {
				max = payload.SeqNum
			}
			err = s.payloads.Push(payload)
			if err != nil {
				log.Logger.Warnf("Payload with sequence number %d was received earlier", payload.SeqNum)
			}

			return max, nil
		}

		bhr := block.GetHeader()
		bphash := bhr.GetPreviousHash()

		var lastBlock *common.Block
		var currentHeight uint64
		if i == 0 {
			// 这里的for循环是保证s.committer.LedgerHeight()获取的账本高度一定>1,才能在后面使用s.GetBlock()；否则会获取到创世块
			for {
				currentHeight, err = s.committer.LedgerHeight()
				if err != nil {
					return uint64(0), err
				}
				if currentHeight == 1 {
					continue
				}
				break
			}

			blockNum := bhr.GetNumber()
			if currentHeight != blockNum {
				return uint64(0), errors.New(fmt.Sprintf("last block : %d  not committed", currentHeight-1))
			}
			lastBlock = s.GetBlock(currentHeight - 1)

		} else {
			lastPayload := payloads[i-1]
			if lastPayload == nil {
				return uint64(0), errors.New("invalid pre block")
			}
			lastBlock, err = utils.GetBlockFromBlockBytes(payloads[i-1].Data)
			if err != nil {
				return uint64(0), err
			}

		}

		lbhr := lastBlock.GetHeader()

		lhhash := lbhr.Hash()
		ldhash := lbhr.GetDataHash()

		prevHash := hex.EncodeToString(bphash)
		headHash := hex.EncodeToString(bhr.Hash())
		dataHash := hex.EncodeToString(bhr.GetDataHash())

		lprevHash := hex.EncodeToString(lbhr.GetPreviousHash())
		lheadHash := hex.EncodeToString(lhhash)
		ldataHash := hex.EncodeToString(ldhash)

		if !bytes.Equal(bphash, lhhash) {
			log.Logger.Warnf("invalid block, block num : %d block prev hash : %s block header hash : %s data hash : %s last block num : %d prev hash : %s head hash : %s data hash : %s",
				bhr.GetNumber(), prevHash, headHash, dataHash, lbhr.GetNumber(), lprevHash, lheadHash, ldataHash)
			return uint64(0), errors.New("block prehash not equal block hash")
		}

		if max < payload.SeqNum {
			max = payload.SeqNum
		}

		err = s.payloads.Push(payload)
		if err != nil {
			log.Logger.Warnf("Payload with sequence number %d was received earlier", payload.SeqNum)
		}

	}
	return max, nil

}

// Stop function send halting signal to all go routines
func (s *GossipStateProviderImpl) Stop() {
	// Make sure stop won't be executed twice
	// and stop channel won't be used again
	s.once.Do(func() {
		s.stopCh <- struct{}{}
		// Make sure all go-routines has finished
		s.done.Wait()
		// Close all resources
		s.committer.Close()
		close(s.stateRequestCh)
		close(s.stateResponseCh)
		close(s.stopCh)
	})
}

// New message notification/handler
func (s *GossipStateProviderImpl) queueNewMessage(msg *proto.GossipMessage) {
	if !bytes.Equal(msg.Channel, []byte(s.chainID)) {
		log.Logger.Warn("Received enqueue for channel",
			string(msg.Channel), "while expecting channel", s.chainID, "ignoring enqueue")
		return
	}

	log.Logger.Warnf("queueNewMessage : %s", msg)

	dataMsg := msg.GetDataMsg()
	if dataMsg != nil {
		// Add new payload to ordered set

		log.Logger.Debugf("Received new payload with sequence number = [%d]", dataMsg.Payload.SeqNum)
		s.payloads.Push(dataMsg.GetPayload())
	} else {
		log.Logger.Debug("Gossip message received is not of data message type, usually this should not happen.")
	}
}

func (s *GossipStateProviderImpl) deliverPayloads() {
	defer s.done.Done()

	for {
		select {
		// Wait for notification that next seq has arrived
		case <-s.payloads.Ready():
			log.Logger.Warnf("Ready to transfer payloads to the ledger, next sequence number is = [%d]", s.payloads.Next())
			// Collect all subsequent payloads
			for payload := s.payloads.Pop(); payload != nil; payload = s.payloads.Pop() {

				rawBlock := &common.Block{}
				if err := pb.Unmarshal(payload.Data, rawBlock); err != nil {
					log.Logger.Errorf("Error getting block with seqNum = %d due to (%s)...dropping block", payload.SeqNum, err)
					continue
				}

				if rawBlock.Data == nil || rawBlock.Header == nil {
					log.Logger.Errorf("Block with claimed sequence %d has no header (%v) or data (%v)",
						payload.SeqNum, rawBlock.Header, rawBlock.Data)
					continue
				}

				log.Logger.Debug("New block with claimed sequence number ", payload.SeqNum, " transactions num ", len(rawBlock.Data.Data))
				s.commitBlock(rawBlock)
			}

		case <-s.stopCh:
			s.stopCh <- struct{}{}
			log.Logger.Debug("State provider has been stoped, finishing to push new blocks.")
			return
		}
	}
}

func (s *GossipStateProviderImpl) antiEntropy() {
	defer s.done.Done()
	defer log.Logger.Debug("State Provider stopped, stopping anti entropy procedure.")

	antiEntropyInterval := util2.GetDurationOrDefault("peer.gossip.antiEntropyInterval", defAntiEntropyInterval)
	for {
		select {
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		case <-time.After(antiEntropyInterval):
			if atomic.LoadInt32(&s.stateTransferActive) == 1 {
				continue
			}

			current, err := s.committer.LedgerHeight()
			if err != nil {
				// Unable to read from ledger continue to the next round
				log.Logger.Error("Cannot obtain ledger height, due to", err)
				continue
			}
			if current == 0 {
				log.Logger.Error("Ledger reported block height of 0 but this should be impossible")
				continue
			}

			stime := time.Now().UnixNano()
			max := s.maxAvailableLedgerHeight()
			etime := time.Now().UnixNano()

			if current-1 >= max {
				continue
			}

			if max-current > 1000 {
				max = current + 1000
			}

			log.Logger.Warnf("get max ledger spend time : %d", etime-stime)

			s.requestBlocksInRange(current, max)
		}
	}
}

// Iterate over all available peers and check advertised meta state to
// find maximum available ledger height across peers
func (s *GossipStateProviderImpl) maxAvailableLedgerHeight() uint64 {
	max := uint64(0)
	for _, p := range s.gossip.PeersOfChannel(common3.ChainID(s.chainID)) {
		if nodeMetastate, err := FromBytes(p.Metadata); err == nil {
			if max < nodeMetastate.LedgerHeight {
				max = nodeMetastate.LedgerHeight
			}
		}
	}
	return max
}

// GetBlocksInRange capable to acquire blocks with sequence
// numbers in the range [start...end].
func (s *GossipStateProviderImpl) requestBlocksInRange(start uint64, end uint64) {
	atomic.StoreInt32(&s.stateTransferActive, 1)
	defer atomic.StoreInt32(&s.stateTransferActive, 0)

	antiEntropyBatchSize := util2.GetIntOrDefault("peer.gossip.antiEntropyBatchSize", defAntiEntropyBatchSize)

	for prev := start; prev <= end; {
		next := min(end, prev+uint64(antiEntropyBatchSize))

		isExist, prev1 := s.payloads.IsExist(prev, next)
		if isExist {
			prev = next + 1

			//time.Sleep(100 * time.Millisecond)
			continue
		}

		prev = prev1

		gossipMsg := s.stateRequestMessage(prev, next)

		responseReceived := false
		tryCounts := 0

		for !responseReceived {
			if tryCounts > defAntiEntropyMaxRetries {
				log.Logger.Warnf("Wasn't  able to get blocks in range [%d...%d], after %d retries",
					prev, next, tryCounts)
				return
			}
			// Select peers to ask for blocks
			stime := time.Now().UnixNano()
			selectedPeer, err := s.selectPeerToRequestFrom(next)
			if err != nil {
				log.Logger.Warnf("Cannot send state request for blocks in range [%d...%d], due to",
					prev, next, err)
				return
			}
			etime := time.Now().UnixNano()
			log.Logger.Warnf("select selectedPeer : %s to get block spend time : %d", selectedPeer.Endpoint, etime-stime)

			//log.Logger.Warnf("State transfer, with selectedPeer %s, requesting blocks in range [%d...%d], "+
			//	"for chainID %s", selectedPeer.Endpoint, prev, next, s.chainID)

			s.gossip.Send(gossipMsg, selectedPeer)

			tryCounts++

			antiEntropyStateResponseTimeout := util2.GetDurationOrDefault("peer.gossip.antiEntropyStateResponseTimeout", defAntiEntropyStateResponseTimeout)

			// Wait until timeout or response arrival
			select {
			case msg := <-s.stateResponseCh:
				if msg.GetGossipMessage().Nonce != gossipMsg.Nonce {
					continue
				}
				// Got corresponding response for state request, can continue
				index, err := s.handleStateResponse(msg)
				if err != nil {
					log.Logger.Warnf("Wasn't able to process state response for "+
						"blocks [%d...%d], due to %s", prev, next, err)
					continue
				}
				prev = index + 1
				responseReceived = true
			case <-time.After(antiEntropyStateResponseTimeout):
				log.Logger.Warnf("requestBlocksInRange [%d...%d] time out", prev, next)
			case <-s.stopCh:
				s.stopCh <- struct{}{}
				return
			}

		}
	}
}

// Generate state request message for given blocks in range [beginSeq...endSeq]
func (s *GossipStateProviderImpl) stateRequestMessage(beginSeq uint64, endSeq uint64) *proto.GossipMessage {
	// 轻量节点的请求消息为LightWeightStateRequest
	return &proto.GossipMessage{
		Nonce:   util2.RandomUInt64(),
		Channel: []byte(s.chainID),
		Tag:     proto.GossipMessage_CHAN_OR_ORG,
		Content: &proto.GossipMessage_LightweightRequest{
			LightweightRequest: &proto.LightWeightStateRequest{
				StartSeqNum: beginSeq,
				EndSeqNum:   endSeq,
				Members:     s.fromMembers,
			},
		},
	}
}

// Select peer which has required blocks to ask missing blocks from
func (s *GossipStateProviderImpl) selectPeerToRequestFrom(height uint64) (*comm2.RemotePeer, error) {
	// Filter peers which posses required range of missing blocks
	var peers []*comm2.RemotePeer

	peers = s.filterPeers(s.hasRequiredHeight(height))

	n := len(peers)
	if n == 0 {
		return nil, errors.New("there are no peers to ask for missing blocks from")
	}

	// Select peers to ask for blocks
	return peers[util2.RandomInt(n)], nil
}

// filterPeers return list of peers which aligns the predicate provided
func (s *GossipStateProviderImpl) filterPeers(predicate func(peer discovery2.NetworkMember) bool) []*comm2.RemotePeer {
	var peers []*comm2.RemotePeer

	for _, member := range s.gossip.PeersOfChannel(common3.ChainID(s.chainID)) {
		if predicate(member) {
			peers = append(peers, &comm2.RemotePeer{Endpoint: member.PreferredEndpoint(), PKIID: member.PKIid})
		}
	}

	return peers
}

// hasRequiredHeight returns predicate which is capable to filter peers with ledger height above than indicated
// by provided input parameter
func (s *GossipStateProviderImpl) hasRequiredHeight(height uint64) func(peer discovery2.NetworkMember) bool {
	return func(peer discovery2.NetworkMember) bool {
		if nodeMetadata, err := FromBytes(peer.Metadata); err != nil {
			log.Logger.Errorf("Unable to de-serialize node meta state, error = %s", err)
		} else if nodeMetadata.LedgerHeight >= height {
			return true
		}

		return false
	}
}

// GetBlock return ledger block given its sequence number as a parameter
func (s *GossipStateProviderImpl) GetBlock(index uint64) *common.Block {
	// Try to read missing block from the ledger, should return no nil with
	// content including at least one block
	if blocks := s.committer.GetBlocks([]uint64{index}); blocks != nil && len(blocks) > 0 {
		return blocks[0]
	}

	return nil
}

// AddPayload add new payload into state
func (s *GossipStateProviderImpl) AddPayload(payload *proto.Payload) error {
	log.Logger.Infof("Adding new payload into the buffer, seqNum %d", payload.SeqNum)

	return s.payloads.Push(payload)
}

// AddPayload add new payload into state
func (s *GossipStateProviderImpl) GetPayloadSize() int {

	return s.payloads.Size()
}

func (s *GossipStateProviderImpl) commitBlock(block *common.Block) error {
	if err := s.committer.Commit(block, s.payloads.Size()); err != nil {
		log.Logger.Errorf("Got error while committing(%s)", err)
		return err
	}

	// 轻量节点不更新nodeMetastate
	//// Update ledger level within node metadata
	//nodeMetastate := NewNodeMetastate(block.Header.Number)
	//// Decode nodeMetastate to byte array
	//b, err := nodeMetastate.Bytes()
	//if err == nil {
	//	s.gossip.UpdateChannelMetadata(b, common2.ChainID(s.chainID))
	//	log.Logger.Warnf("update channel metadata : %s", nodeMetastate)
	//} else {
	//	log.Logger.Errorf("Unable to serialize node meta nodeMetastate, error = %s", err)
	//}

	//log.Logger.Warnf("update channel metadata : %d", block.Header.Number)
	log.Logger.Warnf("Channel [%s]: Created block [%d] with %d transaction(s),remain payload(%d)",
		s.chainID, block.Header.Number, len(block.Data.Data), s.payloads.Size())

	return nil
}

func min(a uint64, b uint64) uint64 {
	return b ^ ((a ^ b) & (-((a - b) >> 63)))
}
