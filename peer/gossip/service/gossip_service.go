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

package service

import (
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/committer"
	"github.com/rongzer/blockchain/peer/deliverclient"
	"github.com/rongzer/blockchain/peer/deliverclient/blocksprovider"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	election2 "github.com/rongzer/blockchain/peer/gossip/election"
	gossip2 "github.com/rongzer/blockchain/peer/gossip/gossip"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	integration2 "github.com/rongzer/blockchain/peer/gossip/integration"
	state2 "github.com/rongzer/blockchain/peer/gossip/state"
	"github.com/rongzer/blockchain/protos/common"
	proto "github.com/rongzer/blockchain/protos/gossip"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	gossipServiceInstance *gossipServiceImpl
	once                  sync.Once
)

type gossipSvc gossip2.Gossip

// GossipService encapsulates gossip and state capabilities into single interface
type GossipService interface {
	gossip2.Gossip

	// NewConfigEventer creates a ConfigProcessor which the configtx.Manager can ultimately route config updates to
	NewConfigEventer() ConfigProcessor
	// InitializeChannel allocates the state provider and should be invoked once per channel per execution
	InitializeChannel(chainID string, committer committer.Committer, endpoints []string)
	// GetBlock returns block for given chain
	GetBlock(chainID string, index uint64) *common.Block
	// AddPayload appends message payload to for given chain
	AddPayload(chainID string, payload *proto.Payload) error
	// 获取缓存数
	GetPayloadSize(chainID string) int
}

// DeliveryServiceFactory factory to create and initialize delivery service instance
type DeliveryServiceFactory interface {
	// Returns an instance of delivery client
	Service(g GossipService, endpoints []string, msc api2.MessageCryptoService) (deliverclient.DeliverService, error)
}

type deliveryFactoryImpl struct {
}

//  Returns an instance of delivery client
func (*deliveryFactoryImpl) Service(g GossipService, endpoints []string, mcs api2.MessageCryptoService) (deliverclient.DeliverService, error) {
	return deliverclient.NewDeliverService(&deliverclient.Config{
		CryptoSvc:   mcs,
		Gossip:      g,
		Endpoints:   endpoints,
		ConnFactory: deliverclient.DefaultConnectionFactory,
		ABCFactory:  deliverclient.DefaultABCFactory,
	})
}

type gossipServiceImpl struct {
	gossipSvc
	chains            map[string]state2.GossipStateProvider
	ordererAddressMap map[string][]string
	leaderElection    map[string]election2.LeaderElectionService
	deliveryService   map[string]deliverclient.DeliverService
	deliveryFactory   DeliveryServiceFactory
	lock              sync.RWMutex
	idMapper          identity2.Mapper
	mcs               api2.MessageCryptoService
	peerIdentity      []byte
	secAdv            api2.SecurityAdvisor
}

// This is an implementation of api.JoinChannelMessage.
type joinChannelMessage struct {
	seqNum              uint64
	members2AnchorPeers map[string][]api2.AnchorPeer
}

func (jcm *joinChannelMessage) SequenceNumber() uint64 {
	return jcm.seqNum
}

// Members returns the organizations of the channel
func (jcm *joinChannelMessage) Members() []api2.OrgIdentityType {
	members := make([]api2.OrgIdentityType, len(jcm.members2AnchorPeers))
	i := 0
	for org := range jcm.members2AnchorPeers {
		members[i] = api2.OrgIdentityType(org)
		i++
	}
	return members
}

// AnchorPeersOf returns the anchor peers of the given organization
func (jcm *joinChannelMessage) AnchorPeersOf(org api2.OrgIdentityType) []api2.AnchorPeer {
	return jcm.members2AnchorPeers[string(org)]
}

// InitGossipService initialize gossip service
func InitGossipService(peerIdentity []byte, endpoint string, s *grpc.Server, mcs api2.MessageCryptoService,
	secAdv api2.SecurityAdvisor, secureDialOpts api2.PeerSecureDialOpts, bootPeers ...string) error {
	// TODO: Remove this.
	// TODO: This is a temporary work-around to make the gossip leader election module load its log.Logger at startup
	// TODO: in order for the flogging package to register this log.Logger in time so it can set the log levels as requested in the config
	return InitGossipServiceCustomDeliveryFactory(peerIdentity, endpoint, s, &deliveryFactoryImpl{},
		mcs, secAdv, secureDialOpts, bootPeers...)
}

// InitGossipServiceCustomDeliveryFactory initialize gossip service with customize delivery factory
// implementation, might be useful for testing and mocking purposes
func InitGossipServiceCustomDeliveryFactory(peerIdentity []byte, endpoint string, s *grpc.Server,
	factory DeliveryServiceFactory, mcs api2.MessageCryptoService, secAdv api2.SecurityAdvisor,
	secureDialOpts api2.PeerSecureDialOpts, bootPeers ...string) error {
	var err error
	var gossip gossip2.Gossip
	once.Do(func() {
		if overrideEndpoint := viper.GetString("peer.gossip.endpoint"); overrideEndpoint != "" {
			endpoint = overrideEndpoint
		}

		log.Logger.Info("Initialize gossip with endpoint", endpoint, "and bootstrap set", bootPeers)

		idMapper := identity2.NewIdentityMapper(mcs, peerIdentity)
		gossip, err = integration2.NewGossipComponent(peerIdentity, endpoint, s, secAdv,
			mcs, idMapper, secureDialOpts, bootPeers...)
		gossipServiceInstance = &gossipServiceImpl{
			mcs:               mcs,
			gossipSvc:         gossip,
			chains:            make(map[string]state2.GossipStateProvider),
			ordererAddressMap: make(map[string][]string),
			leaderElection:    make(map[string]election2.LeaderElectionService),
			deliveryService:   make(map[string]deliverclient.DeliverService),
			deliveryFactory:   factory,
			idMapper:          idMapper,
			peerIdentity:      peerIdentity,
			secAdv:            secAdv,
		}
	})
	return err
}

// GetGossipService returns an instance of gossip service
func GetGossipService() GossipService {
	// 若gossipServiceInstance未被初始化，则说明是light节点启动
	if gossipServiceInstance == nil {
		return lightGossipServiceInstance
	}
	// 否则，作为正常peer节点启动
	return gossipServiceInstance
}

// NewConfigEventer creates a ConfigProcessor which the configtx.Manager can ultimately route config updates to
func (g *gossipServiceImpl) NewConfigEventer() ConfigProcessor {
	return newConfigEventer(g)
}

// InitializeChannel allocates the state provider and should be invoked once per channel per execution
func (g *gossipServiceImpl) InitializeChannel(chainID string, committer committer.Committer, endpoints []string) {
	g.lock.Lock()
	defer g.lock.Unlock()
	// Initialize new state provider for given committer
	log.Logger.Debug("Creating state provider for chainID", chainID)
	g.chains[chainID] = state2.NewGossipStateProvider(chainID, g, committer, g.mcs)
	g.ordererAddressMap[chainID] = endpoints
	for i := 0; i < len(endpoints); i++ {
		log.Logger.Infof("endpoints %d  is : %s ", i, endpoints[i])
	}

	if g.deliveryService[chainID] == nil {
		var err error
		g.deliveryService[chainID], err = g.deliveryFactory.Service(gossipServiceInstance, endpoints, g.mcs)
		if err != nil {
			log.Logger.Warn("Cannot create delivery client, due to", err)
		}
	}

	// Delivery service might be nil only if it was not able to get connected
	// to the ordering service
	if g.deliveryService != nil {
		// Parameters:
		//              - peer.gossip.useLeaderElection
		//              - peer.gossip.orgLeader
		//
		// are mutual exclusive, setting both to true is not defined, hence
		// peer will panic and terminate
		leaderElection := viper.GetBool("peer.gossip.useLeaderElection")
		isStaticOrgLeader := viper.GetBool("peer.gossip.orgLeader")

		if leaderElection && isStaticOrgLeader {
			log.Logger.Panic("Setting both orgLeader and useLeaderElection to true isn't supported, aborting execution")
		}

		if leaderElection {
			log.Logger.Debug("Delivery uses dynamic leader election mechanism, channel", chainID)
			g.leaderElection[chainID] = g.newLeaderElectionComponent(chainID, g.onStatusChangeFactory(chainID, committer))
		} else if isStaticOrgLeader {
			log.Logger.Debug("This peer is configured to connect to ordering service for blocks delivery, channel", chainID)
			g.deliveryService[chainID].StartDeliverForChannel(chainID, committer, func() {})
		} else {
			log.Logger.Debug("This peer is not configured to connect to ordering service for blocks delivery, channel", chainID)
		}
	} else {
		log.Logger.Warn("Delivery client is down won't be able to pull blocks for chain", chainID)
	}
}

// configUpdated constructs a joinChannelMessage and sends it to the gossipSvc
func (g *gossipServiceImpl) configUpdated(config Config) {
	myOrg := string(g.secAdv.OrgByPeerIdentity(g.peerIdentity))
	if !g.amIinChannel(myOrg, config) {
		log.Logger.Error("Tried joining channel", config.ChainID(), "but our org(", myOrg, "), isn't "+
			"among the orgs of the channel:", orgListFromConfig(config), ", aborting.")
		return
	}
	jcm := &joinChannelMessage{seqNum: config.Sequence(), members2AnchorPeers: map[string][]api2.AnchorPeer{}}
	for _, appOrg := range config.Organizations() {
		log.Logger.Debug(appOrg.MSPID(), "anchor peers:", appOrg.AnchorPeers())
		jcm.members2AnchorPeers[appOrg.MSPID()] = []api2.AnchorPeer{}
		for _, ap := range appOrg.AnchorPeers() {
			anchorPeer := api2.AnchorPeer{
				Host: ap.Host,
				Port: int(ap.Port),
			}
			jcm.members2AnchorPeers[appOrg.MSPID()] = append(jcm.members2AnchorPeers[appOrg.MSPID()], anchorPeer)
		}
	}

	// Initialize new state provider for given committer
	log.Logger.Debug("Creating state provider for chainID", config.ChainID())
	g.JoinChan(jcm, common2.ChainID(config.ChainID()))
}

// GetBlock returns block for given chain
func (g *gossipServiceImpl) GetBlock(chainID string, index uint64) *common.Block {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.chains[chainID].GetBlock(index)
}

// AddPayload appends message payload to for given chain
func (g *gossipServiceImpl) AddPayload(chainID string, payload *proto.Payload) error {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.chains[chainID].AddPayload(payload)
}

// 获取缓存数
func (g *gossipServiceImpl) GetPayloadSize(chainID string) int {
	return g.chains[chainID].GetPayloadSize()
}

//  Stop stops the gossip component
func (g *gossipServiceImpl) Stop() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for chainID := range g.chains {
		log.Logger.Info("Stopping chain", chainID)
		if le, exists := g.leaderElection[chainID]; exists {
			log.Logger.Infof("Stopping leader election for %s", chainID)
			le.Stop()
		}
		g.chains[chainID].Stop()

		if g.deliveryService[chainID] != nil {
			g.deliveryService[chainID].Stop()
		}
	}

	g.gossipSvc.Stop()
}

func (g *gossipServiceImpl) newLeaderElectionComponent(chainID string, callback func(bool)) election2.LeaderElectionService {
	PKIid := g.idMapper.GetPKIidOfCert(g.peerIdentity)
	adapter := election2.NewAdapter(g, PKIid, common2.ChainID(chainID))
	return election2.NewLeaderElectionService(adapter, string(PKIid), callback)
}

func (g *gossipServiceImpl) amIinChannel(myOrg string, config Config) bool {
	for _, orgName := range orgListFromConfig(config) {
		if orgName == myOrg {
			return true
		}
	}
	return false
}

func (g *gossipServiceImpl) onStatusChangeFactory(chainID string, committer blocksprovider.LedgerInfo) func(bool) {
	return func(isLeader bool) {
		if isLeader {
			yield := func() {
				g.lock.RLock()
				le := g.leaderElection[chainID]
				g.lock.RUnlock()
				le.Yield()
			}
			log.Logger.Info("Elected as a leader, starting delivery service for channel", chainID)
			if err := g.deliveryService[chainID].StartDeliverForChannel(chainID, committer, yield); err != nil {
				log.Logger.Error("Delivery service is not able to start blocks delivery for chain, due to", err)
			}
		} else {
			log.Logger.Info("Renounced leadership, stopping delivery service for channel", chainID)
			if err := g.deliveryService[chainID].StopDeliverForChannel(chainID); err != nil {
				log.Logger.Error("Delivery service is not able to stop blocks delivery for chain, due to", err)
			}
		}
	}
}

func orgListFromConfig(config Config) []string {
	var orgList []string
	for _, appOrg := range config.Organizations() {
		orgList = append(orgList, appOrg.MSPID())
	}
	return orgList
}
