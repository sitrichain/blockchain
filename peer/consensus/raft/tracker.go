package raft

import (
	"github.com/rongzer/blockchain/common/log"
	"sync/atomic"

	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"github.com/rongzer/blockchain/protos/utils"
	"go.etcd.io/etcd/raft"
)

// Tracker periodically poll Raft Status, and update disseminator
// so that status is populated to followers.
type Tracker struct {
	id     uint64
	sender *Disseminator
	active *atomic.Value

	counter int

	logger *log.RaftLogger
}

func (t *Tracker) Check(status *raft.Status) {
	// leaderless
	if status.Lead == raft.None {
		t.active.Store([]uint64{})
		return
	}

	// follower
	if status.RaftState == raft.StateFollower {
		return
	}

	// leader
	current := []uint64{t.id}
	for id, progress := range status.Progress {

		if id == t.id {
			// `RecentActive` for leader's Progress is expected to be false in current implementation of etcd/raft,
			// but because not marking the leader recently active might be considered a bug and fixed in the future,
			// we explicitly defend against adding the leader, to avoid potential duplicate
			continue
		}

		if progress.RecentActive {
			current = append(current, id)
		}
	}

	last := t.active.Load().([]uint64)
	t.active.Store(current)

	if len(current) != len(last) {
		t.counter = 0
		return
	}

	// consider active nodes to be stable if it holds for 3 iterations, to avoid glitch
	// in this value when the recent status is reset on leader election intervals
	if t.counter < 3 {
		t.counter++
		return
	}

	t.counter = 0
	t.logger.Debugf("Current active nodes in cluster are: %+v", current)

	metadata := utils.MarshalOrPanic(&etcdraft.ClusterMetadata{ActiveNodes: current})
	t.sender.UpdateMetadata(metadata)
}