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

package filter

import (
	comm2 "github.com/rongzer/blockchain/peer/gossip/comm"
	discovery2 "github.com/rongzer/blockchain/peer/gossip/discovery"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
)

// RoutingFilter defines a predicate on a NetworkMember
// It is used to assert whether a given NetworkMember should be
// selected for be given a message
type RoutingFilter func(discovery2.NetworkMember) bool

// SelectNonePolicy selects an empty set of members
var SelectNonePolicy = func(discovery2.NetworkMember) bool {
	return false
}

// SelectAllPolicy selects all members given
var SelectAllPolicy = func(discovery2.NetworkMember) bool {
	return true
}

// CombineRoutingFilters returns the logical AND of given routing filters
func CombineRoutingFilters(filters ...RoutingFilter) RoutingFilter {
	return func(member discovery2.NetworkMember) bool {
		for _, filter := range filters {
			if !filter(member) {
				return false
			}
		}
		return true
	}
}

// SelectPeers returns a slice of peers that match the routing filter
func SelectPeers(k int, peerPool []discovery2.NetworkMember, filter RoutingFilter) []*comm2.RemotePeer {
	var filteredPeers []*comm2.RemotePeer
	for _, peer := range peerPool {
		if filter(peer) {
			filteredPeers = append(filteredPeers, &comm2.RemotePeer{PKIID: peer.PKIid, Endpoint: peer.PreferredEndpoint()})
		}
	}

	var indices []int
	if len(filteredPeers) <= k {
		indices = make([]int, len(filteredPeers))
		for i := 0; i < len(filteredPeers); i++ {
			indices[i] = i
		}
	} else {
		indices = util2.GetRandomIndices(k, len(filteredPeers)-1)
	}

	var remotePeers []*comm2.RemotePeer
	for _, index := range indices {
		remotePeers = append(remotePeers, filteredPeers[index])
	}

	return remotePeers
}
