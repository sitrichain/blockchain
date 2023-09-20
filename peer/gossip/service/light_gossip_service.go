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
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	election2 "github.com/rongzer/blockchain/peer/gossip/election"
	gossip2 "github.com/rongzer/blockchain/peer/gossip/gossip"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	integration2 "github.com/rongzer/blockchain/peer/gossip/integration"
	light2 "github.com/rongzer/blockchain/peer/gossip/light"
	state2 "github.com/rongzer/blockchain/peer/gossip/state"
	"github.com/rongzer/blockchain/protos/common"
	proto "github.com/rongzer/blockchain/protos/gossip"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	lightGossipServiceInstance *lightGossipServiceImpl
)

type lightGossipServiceImpl struct {
	gossipSvc
	chains            map[string]state2.GossipStateProvider
	ordererAddressMap map[string][]string
	leaderElection    map[string]election2.LeaderElectionService
	deliveryService   deliverclient.DeliverService
	deliveryFactory   DeliveryServiceFactory
	lock              sync.RWMutex
	idMapper          identity2.Mapper
	mcs               api2.MessageCryptoService
	peerIdentity      []byte
	secAdv            api2.SecurityAdvisor
}

// InitGossipService initialize gossip service
func InitLightGossipService(peerIdentity []byte, endpoint string, s *grpc.Server, mcs api2.MessageCryptoService,
	secAdv api2.SecurityAdvisor, secureDialOpts api2.PeerSecureDialOpts, bootPeers ...string) error {
	// TODO: Remove this.
	// TODO: This is a temporary work-around to make the gossip leader election module load its log.Logger at startup
	// TODO: in order for the flogging package to register this log.Logger in time so it can set the log levels as requested in the config
	return InitLightGossipServiceCustomDeliveryFactory(peerIdentity, endpoint, s, &deliveryFactoryImpl{},
		mcs, secAdv, secureDialOpts, bootPeers...)
}

// InitGossipServiceCustomDeliveryFactory initialize gossip service with customize delivery factory
// implementation, might be useful for testing and mocking purposes
func InitLightGossipServiceCustomDeliveryFactory(peerIdentity []byte, endpoint string, s *grpc.Server,
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
		lightGossipServiceInstance = &lightGossipServiceImpl{
			mcs:               mcs,
			gossipSvc:         gossip,
			chains:            make(map[string]state2.GossipStateProvider),
			ordererAddressMap: make(map[string][]string),
			leaderElection:    make(map[string]election2.LeaderElectionService),
			deliveryFactory:   factory,
			idMapper:          idMapper,
			peerIdentity:      peerIdentity,
			secAdv:            secAdv,
		}
	})
	return err
}

// GetGossipService returns an instance of gossip service
func GetLightGossipService() GossipService {
	return lightGossipServiceInstance
}

// NewConfigEventer creates a ConfigProcessor which the configtx.Manager can ultimately route config updates to
func (g *lightGossipServiceImpl) NewConfigEventer() ConfigProcessor {
	return newConfigEventer(g)
}

// InitializeChannel allocates the state provider and should be invoked once per channel per execution
func (g *lightGossipServiceImpl) InitializeChannel(chainID string, committer committer.Committer, endpoints []string) {
	g.lock.Lock()
	defer g.lock.Unlock()
	// Initialize new state provider for given committer
	log.Logger.Debug("Creating state provider for chainID", chainID)
	g.chains[chainID] = light2.NewGossipStateProvider(chainID, g, committer, g.mcs)
	g.ordererAddressMap[chainID] = endpoints
	for i := 0; i < len(endpoints); i++ {
		log.Logger.Infof("endpoints %d  is : %s ", i, endpoints[i])
	}

	if g.deliveryService == nil {
		var err error
		g.deliveryService, err = g.deliveryFactory.Service(lightGossipServiceInstance, endpoints, g.mcs)
		if err != nil {
			log.Logger.Warn("Cannot create delivery client, due to", err)
		}
	}
}

// configUpdated constructs a joinChannelMessage and sends it to the gossipSvc
func (g *lightGossipServiceImpl) configUpdated(config Config) {
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
func (g *lightGossipServiceImpl) GetBlock(chainID string, index uint64) *common.Block {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.chains[chainID].GetBlock(index)
}

// AddPayload appends message payload to for given chain
func (g *lightGossipServiceImpl) AddPayload(chainID string, payload *proto.Payload) error {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.chains[chainID].AddPayload(payload)
}

// 获取缓存数
func (g *lightGossipServiceImpl) GetPayloadSize(chainID string) int {
	return g.chains[chainID].GetPayloadSize()
}

//  Stop stops the gossip component
func (g *lightGossipServiceImpl) Stop() {
	g.lock.Lock()
	defer g.lock.Unlock()
	for _, ch := range g.chains {
		log.Logger.Info("Stopping chain", ch)
		ch.Stop()
	}

	for chainID, electionService := range g.leaderElection {
		log.Logger.Infof("Stopping leader election for %s", chainID)
		electionService.Stop()
	}
	g.gossipSvc.Stop()
	if g.deliveryService != nil {
		g.deliveryService.Stop()
	}
}

func (g *lightGossipServiceImpl) amIinChannel(myOrg string, config Config) bool {
	for _, orgName := range orgListFromConfig(config) {
		if orgName == myOrg {
			return true
		}
	}
	return false
}
