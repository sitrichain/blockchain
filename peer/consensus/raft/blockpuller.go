package raft

import (
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"

	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/consensus"
	"github.com/rongzer/blockchain/protos/common"
)

// LedgerBlockPuller pulls blocks upon demand, or fetches them from the ledger
type LedgerBlockPuller struct {
	BlockPuller
	BlockRetriever cluster.BlockRetriever
	Height         func() uint64
}

func (lp *LedgerBlockPuller) PullBlock(seq uint64) *common.Block {
	lastSeq := lp.Height() - 1
	if lastSeq >= seq {
		block, err := lp.BlockRetriever.GetBlockByNumber(seq)
		if err != nil {
			log.Logger.Errorf("cannot pull block: %v, err is: %v", seq, err)
			return nil
		}
		return block
	}
	return lp.BlockPuller.PullBlock(seq)
}

// NewBlockPuller creates a new block puller
func NewBlockPuller(support consensus.ChainResource,
	baseDialer *cluster.PredicateDialer,
	blockMetadata *etcdraft.BlockMetadata,
) (BlockPuller, error) {

	stdDialer := &cluster.StandardDialer{
		Config: baseDialer.Config.Clone(),
	}
	stdDialer.Config.AsyncConnect = false
	var endpoints []cluster.EndpointCriteria
	for _, ep := range blockMetadata.ConsenterEndpoints {
		endpoints = append(endpoints, cluster.EndpointCriteria{Endpoint: ep})
	}

	bp := &cluster.BlockPuller{
		Logger:              log.NewRaftLogger(log.Logger).With("channel", support.ChainID()),
		RetryTimeout:        conf.V.Sealer.Raft.ReplicationRetryTimeout,
		MaxTotalBufferBytes: conf.V.Sealer.Raft.ReplicationBufferSize,
		FetchTimeout:        conf.V.Sealer.Raft.ReplicationPullTimeout,
		Endpoints:           endpoints,
		Signer:              support,
		Channel:             support.ChainID(),
		Dialer:              stdDialer,
	}

	return &LedgerBlockPuller{
		Height:         support.Height,
		BlockRetriever: support,
		BlockPuller:    bp,
	}, nil
}
