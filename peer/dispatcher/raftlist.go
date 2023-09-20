package dispatcher

import (
	"fmt"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/consensus/raft"
	"strings"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/log"
	cb "github.com/rongzer/blockchain/protos/common"
)

// peer列表, 根据peer注册信息而来
type RaftPeerList struct {
	sync.RWMutex
	peers    *cb.PeerList  // peer列表
	balancer *loadBalancer // 负载均衡
}

func newRaftPeerList(consenter *raft.Consenter, chainId string) *RaftPeerList {
	l := &RaftPeerList{
		balancer: newLoadBalance(),
		peers:      &cb.PeerList{},
	}
	// 定时评估当前所有peer状态, 将可背书的节点交给负载均衡
	go func() {
		t := time.NewTicker(time.Second * 1)
		for {
			select {
			case <-t.C:
				l.evaluate(consenter, chainId)
			}
		}
	}()

	return l
}

func (l *RaftPeerList) marshal() ([]byte, error) {
	l.RLock()
	defer l.RUnlock()
	return l.peers.Marshal()
}

// 评价各节点状态, 更新可背书的节点地址列表
func (l *RaftPeerList) evaluate(consenter *raft.Consenter, chainId string) {
	// 使用新的节点列表
	activeNodes := consenter.ActiveNodes.Load().([]uint64)
	var activePeers = cb.PeerList{List: []*cb.PeerInfo{}}

	for _, raftId := range activeNodes {
		c, ok := consenter.Opts.Consenters[raftId]
		if !ok {
			log.Logger.Errorf("cannot take endPoint of this raftId: %v from consenter Map", raftId)
			continue
		}
		targetLedger := chain.GetLedger(chainId)
		binfo, err := targetLedger.GetBlockchainInfo()
		if err != nil {
			log.Logger.Errorf("Failed to get block info with error %s", err)
			continue
		}
		activePeers.List = append(activePeers.List, &cb.PeerInfo{
			Id: conf.V.Peer.ID,
			Address: fmt.Sprintf("%v:%v", c.Host, c.PeerPort),
			Cpu: 2,
			PeerImageId: strings.TrimPrefix(conf.V.Peer.VM.ImageTag, ":"),
			JavaenvImageId: strings.TrimPrefix(conf.V.Peer.VM.ImageTag, ":"),
			BlockInfo: binfo,
		})
	}
	l.Lock()
	defer l.Unlock()
	l.peers = &activePeers
	// balancer增加新peer
	for _, newPeer := range l.peers.List {
		l.balancer.add(newPeer.Address)
	}
	// balancer移除挂掉的peer

	for _, maybeDeadPeer := range l.balancer.peers {
		for _, activePeer := range activePeers.List {
			if activePeer.Address == maybeDeadPeer {
				goto OUT
			}
		}
		l.balancer.remove(maybeDeadPeer, "")
	OUT:
		continue
	}

}
