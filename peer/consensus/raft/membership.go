package raft

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
)

// MembershipByEndpoint convert consenters map into set encapsulated by map
// where key is node's endpoint
func MembershipByEndpoint(consenters map[uint64]*etcdraft.Consenter) map[string]uint64 {
	set := map[string]uint64{}
	for nodeID, c := range consenters {
		set[fmt.Sprintf("%v:%v", c.Host, c.Port)] = nodeID
	}
	return set
}

// MembershipChanges keeps information about membership
// changes introduced during configuration update
type MembershipChanges struct {
	NewBlockMetadata *etcdraft.BlockMetadata
	NewConsenters    map[uint64]*etcdraft.Consenter
	AddedNodes       []*etcdraft.Consenter
	RemovedNodes     []*etcdraft.Consenter
	ConfChange       *raftpb.ConfChange
	RotatedNode      uint64
}

// ComputeMembershipChanges computes membership update based on information about new consenters, returns
// two slices: a slice of added consenters and a slice of consenters to be removed
func ComputeMembershipChanges(oldMetadata *etcdraft.BlockMetadata, oldConsenters map[uint64]*etcdraft.Consenter, newConsenters []*etcdraft.Consenter) (mc *MembershipChanges, err error) {
	result := &MembershipChanges{
		NewConsenters:    map[uint64]*etcdraft.Consenter{},
		NewBlockMetadata: proto.Clone(oldMetadata).(*etcdraft.BlockMetadata),
		AddedNodes:       []*etcdraft.Consenter{},
		RemovedNodes:     []*etcdraft.Consenter{},
	}

	result.NewBlockMetadata.ConsenterEndpoints = make([]string, len(newConsenters))
	result.NewBlockMetadata.PeerEndpoints = make([]string, len(newConsenters))

	var addedNodeIndex int
	currentConsentersSet := MembershipByEndpoint(oldConsenters)
	for i, c := range newConsenters {
		if nodeID, exists := currentConsentersSet[fmt.Sprintf("%v:%v", c.Host, c.Port)]; exists {
			result.NewBlockMetadata.ConsenterEndpoints[i] = fmt.Sprintf("%v:%v", c.Host, c.Port)
			result.NewBlockMetadata.PeerEndpoints[i] = fmt.Sprintf("%v:%v", c.Host, c.PeerPort)
			result.NewConsenters[nodeID] = c
			continue
		}
		addedNodeIndex = i
		result.AddedNodes = append(result.AddedNodes, c)
	}
	// 本次迭代不考虑raft集群配置的节点删除，只考虑节点增加，且一次仅增加一个节点（raft论文规定）

	switch {
	case len(result.AddedNodes) == 1:
		// new node
		nodeID := result.NewBlockMetadata.NextConsenterId
		result.NewConsenters[nodeID] = result.AddedNodes[0]
		result.NewBlockMetadata.ConsenterEndpoints[addedNodeIndex] = fmt.Sprintf("%v:%v", result.AddedNodes[0].Host, result.AddedNodes[0].Port)
		result.NewBlockMetadata.PeerEndpoints[addedNodeIndex] = fmt.Sprintf("%v:%v", result.AddedNodes[0].Host, result.AddedNodes[0].PeerPort)
		result.NewBlockMetadata.NextConsenterId++
		result.ConfChange = &raftpb.ConfChange{
			NodeID: nodeID,
			Type:   raftpb.ConfChangeAddNode,
		}
	case len(result.AddedNodes) == 0:
		// no change
	default:
		// len(result.AddedNodes) > 1 || len(result.RemovedNodes) > 1 {
		return nil, errors.Errorf("update of more than one Consenter at a time is not supported, requested changes: %s", result)
	}

	return result, nil
}

// Stringer implements fmt.Stringer interface
func (mc *MembershipChanges) String() string {
	return fmt.Sprintf("add %d node(s), remove %d node(s)", len(mc.AddedNodes), len(mc.RemovedNodes))
}

// Changed indicates whether these changes actually do anything
func (mc *MembershipChanges) Changed() bool {
	return len(mc.AddedNodes) > 0 || len(mc.RemovedNodes) > 0
}

// Rotated indicates whether the change was a rotation
func (mc *MembershipChanges) Rotated() bool {
	return len(mc.AddedNodes) == 1 && len(mc.RemovedNodes) == 1
}

// UnacceptableQuorumLoss returns true if membership change will result in avoidable quorum loss,
// given current number of active nodes in cluster. Avoidable means that more nodes can be started
// to prevent quorum loss. Sometimes, quorum loss is inevitable, for example expanding 1-node cluster.
func (mc *MembershipChanges) UnacceptableQuorumLoss(active []uint64) bool {
	activeMap := make(map[uint64]struct{})
	for _, i := range active {
		activeMap[i] = struct{}{}
	}

	isCFT := len(mc.NewConsenters) > 2 // if resulting cluster cannot tolerate any fault, quorum loss is inevitable
	quorum := len(mc.NewConsenters)/2 + 1

	switch {
	case mc.ConfChange != nil && mc.ConfChange.Type == raftpb.ConfChangeAddNode: // Add
		return isCFT && len(active) < quorum

	case mc.RotatedNode != raft.None: // Rotate
		delete(activeMap, mc.RotatedNode)
		return isCFT && len(activeMap) < quorum

	case mc.ConfChange != nil && mc.ConfChange.Type == raftpb.ConfChangeRemoveNode: // Remove
		delete(activeMap, mc.ConfChange.NodeID)
		return len(activeMap) < quorum

	default: // No change
		return false
	}
}
