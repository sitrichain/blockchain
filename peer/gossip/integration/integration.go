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

package integration

import (
	"crypto/tls"
	"fmt"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	gossip2 "github.com/rongzer/blockchain/peer/gossip/gossip"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
	"net"
	"strconv"
	"time"

	"github.com/rongzer/blockchain/peer/config"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// This file is used to bootstrap a gossip instance and/or leader election service instance

func newConfig(selfEndpoint string, externalEndpoint string, bootPeers ...string) (*gossip2.Config, error) {
	_, p, err := net.SplitHostPort(selfEndpoint)

	if err != nil {
		return nil, fmt.Errorf("misconfigured endpoint %s, the error is %s", selfEndpoint, err)
	}

	port, err := strconv.ParseInt(p, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("misconfigured endpoint %s, failed to parse port number due to %s", selfEndpoint, err)
	}

	var cert *tls.Certificate
	if viper.GetBool("peer.tls.enabled") {
		certTmp, err := tls.LoadX509KeyPair(config.GetPath("peer.tls.cert.file"), config.GetPath("peer.tls.key.file"))
		if err != nil {
			return nil, fmt.Errorf("failed to load certificates because of %s", err)
		}
		cert = &certTmp
	}

	return &gossip2.Config{
		BindPort:                   int(port),
		BootstrapPeers:             bootPeers,
		ID:                         selfEndpoint,
		MaxBlockCountToStore:       util2.GetIntOrDefault("peer.gossip.maxBlockCountToStore", 100),
		MaxPropagationBurstLatency: util2.GetDurationOrDefault("core.peer.gossip.maxPropagationBurstLatency", 10*time.Millisecond),
		MaxPropagationBurstSize:    util2.GetIntOrDefault("peer.gossip.maxPropagationBurstSize", 10),
		PropagateIterations:        util2.GetIntOrDefault("peer.gossip.propagateIterations", 1),
		PropagatePeerNum:           util2.GetIntOrDefault("peer.gossip.propagatePeerNum", 3),
		PullInterval:               util2.GetDurationOrDefault("core.peer.gossip.pullInterval", 4*time.Second),
		PullPeerNum:                util2.GetIntOrDefault("peer.gossip.pullPeerNum", 3),
		InternalEndpoint:           selfEndpoint,
		ExternalEndpoint:           externalEndpoint,
		PublishCertPeriod:          util2.GetDurationOrDefault("core.peer.gossip.publishCertPeriod", 10*time.Second),
		RequestStateInfoInterval:   util2.GetDurationOrDefault("core.peer.gossip.requestStateInfoInterval", 4*time.Second),
		PublishStateInfoInterval:   util2.GetDurationOrDefault("core.peer.gossip.publishStateInfoInterval", 4*time.Second),
		SkipBlockVerification:      viper.GetBool("peer.gossip.skipBlockVerification"),
		TLSServerCert:              cert,
	}, nil
}

// NewGossipComponent creates a gossip component that attaches itself to the given gRPC server
func NewGossipComponent(peerIdentity []byte, endpoint string, s *grpc.Server,
	secAdv api2.SecurityAdvisor, cryptSvc api2.MessageCryptoService, idMapper identity2.Mapper,
	secureDialOpts api2.PeerSecureDialOpts, bootPeers ...string) (gossip2.Gossip, error) {

	externalEndpoint := viper.GetString("peer.gossip.externalEndpoint")

	conf, err := newConfig(endpoint, externalEndpoint, bootPeers...)
	if err != nil {
		return nil, err
	}
	gossipInstance := gossip2.NewGossipService(conf, s, secAdv, cryptSvc, idMapper,
		peerIdentity, secureDialOpts)

	return gossipInstance, nil
}
