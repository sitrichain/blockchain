/*
Copyright IBM Corp. 2017 All Rights Reserved.

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

package blocksprovider

import (
	"math"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/gossip/api"
	gossipcommon "github.com/rongzer/blockchain/peer/gossip/common"
	"github.com/rongzer/blockchain/peer/gossip/discovery"
	"github.com/rongzer/blockchain/protos/common"
	gossip_proto "github.com/rongzer/blockchain/protos/gossip"
	"github.com/rongzer/blockchain/protos/orderer"
)

// LedgerInfo an adapter to provide the interface to query
// the ledger committer for current ledger height
type LedgerInfo interface {
	// LedgerHeight returns current local ledger height
	LedgerHeight() (uint64, error)
}

// GossipServiceAdapter serves to provide basic functionality
// required from gossip service by delivery service
type GossipServiceAdapter interface {
	// PeersOfChannel returns slice with members of specified channel
	PeersOfChannel(gossipcommon.ChainID) []discovery.NetworkMember

	// AddPayload adds payload to the local state sync buffer
	AddPayload(chainID string, payload *gossip_proto.Payload) error

	// Gossip the message across the peers
	Gossip(msg *gossip_proto.GossipMessage)
	// 获取缓存数
	GetPayloadSize(chainID string) int
}

// BlocksProvider used to read blocks from the ordering service
// for specified chain it subscribed to
type BlocksProvider interface {
	// DeliverBlocks starts delivering and disseminating blocks
	DeliverBlocks()

	// Stop shutdowns blocks provider and stops delivering new blocks
	Stop()
}

// BlocksDeliverer defines interface which actually helps
// to abstract the AtomicBroadcast_DeliverClient with only
// required method for blocks provider.
// This also decouples the production implementation of the gRPC stream
// from the code in order for the code to be more modular and testable.
type BlocksDeliverer interface {
	// Recv retrieves a response from the ordering service
	Recv() (*orderer.DeliverResponse, error)

	// Send sends an envelope to the ordering service
	Send(*common.Envelope) error
}

type streamClient interface {
	BlocksDeliverer

	// Close closes the stream and its underlying connection
	Close()

	// Disconnect disconnects from the remote node
	Disconnect()
}

// blocksProviderImpl the actual implementation for BlocksProvider interface
type blocksProviderImpl struct {
	chainID string

	client streamClient

	gossip GossipServiceAdapter

	mcs api.MessageCryptoService

	done int32

	wrongStatusThreshold int
	rFunc                retryDeliveryBlocks
	height               uint64
	ledgerInfo           LedgerInfo
	endorseChain         chan *common.RBCMessage
}

const wrongStatusThreshold = 10

var MaxRetryDelay = time.Second * 10

// broadcastSetup is a function that is called by the broadcastClient immediately after each
// successful connection to the ordering service
type retryDeliveryBlocks func(ledgerInfo LedgerInfo) error

// NewBlocksProvider constructor function to create blocks deliverer instance
func NewBlocksProvider(chainID string, client streamClient, gossip GossipServiceAdapter, mcs api.MessageCryptoService, rFunc retryDeliveryBlocks, ledgerInfo LedgerInfo) BlocksProvider {
	height, _ := ledgerInfo.LedgerHeight()
	endorseChain := make(chan *common.RBCMessage, 1000)
	blocksProvider := &blocksProviderImpl{
		chainID:              chainID,
		client:               client,
		gossip:               gossip,
		mcs:                  mcs,
		wrongStatusThreshold: wrongStatusThreshold,
		rFunc:                rFunc,
		ledgerInfo:           ledgerInfo,
		endorseChain:         endorseChain,
	}
	log.Logger.Infof("start block provider ledger height %d", height)
	return blocksProvider
}

// DeliverBlocks used to pull out blocks from the ordering service to
// distributed them across peers
func (b *blocksProviderImpl) DeliverBlocks() {
	errorStatusCounter := 0
	statusCounter := 0
	defer b.client.Close()
	for !b.isDone() {
		msg, err := b.client.Recv()
		if err != nil {
			log.Logger.Warnf("[%s] Deliver blocks receive error: %s", b.chainID, err.Error())
			return
		}
		switch t := msg.Type.(type) {
		case *orderer.DeliverResponse_Status:
			if t.Status == common.Status_SUCCESS {
				log.Logger.Warnf("[%s] Deliver blocks received success for a seek that should never complete", b.chainID)
				//return
				go func() {
					for {
						time.Sleep(2 * time.Second)
						if b.gossip.GetPayloadSize(b.chainID) < 500 {
							debug.FreeOSMemory()
							runtime.GC()
							height, _ := b.ledgerInfo.LedgerHeight()
							b.rFunc(b.ledgerInfo)
							log.Logger.Infof("Deliver blocks retry block provider ledger height %d", height)
							break
						}
					}
				}()

				continue
			}
			if t.Status == common.Status_BAD_REQUEST || t.Status == common.Status_FORBIDDEN {
				log.Logger.Errorf("[%s] Deliver blocks got error %s", b.chainID, t.Status.String())
				errorStatusCounter++
				if errorStatusCounter > b.wrongStatusThreshold {
					log.Logger.Fatalf("[%s] Deliver blocks wrong statuses threshold passed, stopping block provider", b.chainID)
					return
				}
			} else {
				errorStatusCounter = 0
				log.Logger.Warnf("[%s]Deliver blocks got error %s", b.chainID, t.Status.String())
			}
			maxDelay := float64(MaxRetryDelay)
			currDelay := float64(time.Duration(math.Pow(2, float64(statusCounter))) * 100 * time.Millisecond)
			time.Sleep(time.Duration(math.Min(maxDelay, currDelay)))
			if currDelay < maxDelay {
				statusCounter++
			}

			b.client.Disconnect()
			continue
		case *orderer.DeliverResponse_Block:
			errorStatusCounter = 0
			statusCounter = 0
			seqNum := t.Block.Header.Number

			marshaledBlock, err := t.Block.Marshal()
			if err != nil {
				log.Logger.Errorf("[%s] Deliver blocks error serializing block with sequence number %d, due to %s", b.chainID, seqNum, err)
				continue
			}
			if err := b.mcs.VerifyBlock(gossipcommon.ChainID(b.chainID), seqNum, marshaledBlock); err != nil {
				log.Logger.Errorf("[%s] Deliver blocks error verifying block with sequnce number %d, due to %s", b.chainID, seqNum, err)
				continue
			}

			numberOfPeers := len(b.gossip.PeersOfChannel(gossipcommon.ChainID(b.chainID)))
			// Create payload with a block received
			payload := createPayload(seqNum, marshaledBlock)
			// Use payload to create gossip message
			gossipMsg := createGossipMsg(b.chainID, payload)

			log.Logger.Debugf("[%s] Deliver blocks adding payload locally, buffer seqNum = [%d], peers number [%d]", b.chainID, seqNum, numberOfPeers)
			// Add payload to local state payloads buffer
			b.gossip.AddPayload(b.chainID, payload)

			// Gossip messages with other nodes
			log.Logger.Debugf("[%s] Deliver blocks gossiping block [%d], peers number [%d]", b.chainID, seqNum, numberOfPeers)
			if false {
				b.gossip.Gossip(gossipMsg)

			}
		case *orderer.DeliverResponse_RbcMessage:

			//err := b.processRBCMessage(t.RBCMessage)
			//if err != nil {
			//	log.Logger.Errorf("Error process RBCMessage %s", err)
			//}

		default:
			log.Logger.Warnf("[%s] Deliver blocks received unknown: %v", b.chainID, t)
			return
		}
	}
}

// Stop stops blocks delivery provider
func (b *blocksProviderImpl) Stop() {
	atomic.StoreInt32(&b.done, 1)
	b.client.Close()
}

// Check whenever provider is stopped
func (b *blocksProviderImpl) isDone() bool {
	return atomic.LoadInt32(&b.done) == 1
}

func createGossipMsg(chainID string, payload *gossip_proto.Payload) *gossip_proto.GossipMessage {
	gossipMsg := &gossip_proto.GossipMessage{
		Nonce:   0,
		Tag:     gossip_proto.GossipMessage_CHAN_AND_ORG,
		Channel: []byte(chainID),
		Content: &gossip_proto.GossipMessage_DataMsg{
			DataMsg: &gossip_proto.DataMessage{
				Payload: payload,
			},
		},
	}
	return gossipMsg
}

func createPayload(seqNum uint64, marshaledBlock []byte) *gossip_proto.Payload {
	return &gossip_proto.Payload{
		Data:   marshaledBlock,
		SeqNum: seqNum,
	}
}
