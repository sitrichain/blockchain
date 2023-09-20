/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package raft

import (
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"github.com/rongzer/blockchain/protos/utils"
	"go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
)

// RaftPeers maps consenters to slice of raft.Peer
func RaftPeers(consenterIDs []string) []raft.Peer {
	var peers []raft.Peer

	for index, _ := range consenterIDs {
		peers = append(peers, raft.Peer{ID: uint64(index + 1)})
	}
	return peers
}

// ConfigChannelHeader expects a config block and returns the header type
// of the config envelope wrapped in it, e.g. HeaderType_ORDERER_TRANSACTION
func ConfigChannelHeader(block *common.Block) (hdr *common.ChannelHeader, err error) {
	envelope, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract envelope from the block")
	}

	channelHeader, err := utils.ChannelHeader(envelope)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract channel header")
	}

	return channelHeader, nil
}

// NodeExists returns trues if node id exists in the slice
// and false otherwise
func NodeExists(id uint64, nodes []uint64) bool {
	for _, nodeID := range nodes {
		if nodeID == id {
			return true
		}
	}
	return false
}

// ConfChange computes Raft configuration changes based on current Raft
// configuration state and consenters IDs stored in RaftMetadata.
func ConfChange(blockMetadata *etcdraft.BlockMetadata, confState *raftpb.ConfState) *raftpb.ConfChange {
	raftConfChange := &raftpb.ConfChange{}

	// need to compute conf changes to propose
	if len(confState.Voters) < len(blockMetadata.ConsenterEndpoints) {
		// adding new node
		raftConfChange.Type = raftpb.ConfChangeAddNode
		for index, _ := range blockMetadata.ConsenterEndpoints {
			if NodeExists(uint64(index+1), confState.Voters) {
				continue
			}
			raftConfChange.NodeID = uint64(index + 1)
		}
	}
	// 不考虑删除节点的情况

	return raftConfChange
}
