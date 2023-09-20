package dispatcher

import (
	"sync"

	"github.com/rongzer/blockchain/common/conf"

	"github.com/rongzer/blockchain/common/log"
	"go.uber.org/ratelimit"
)

var limitPerPeer int

// 负载均衡器,使用RoundRobin.
type loadBalancer struct {
	sync.RWMutex
	peers   []string
	index   int
	limiter ratelimit.Limiter
}

func newLoadBalance() *loadBalancer {
	limitPerPeer = conf.V.Sealer.EndorseLimitPerPeer
	if limitPerPeer == 0 {
		limitPerPeer = 500
	}
	log.Logger.Infof("Set endorser loadbalance rate limit to %d", limitPerPeer)
	return &loadBalancer{
		limiter: ratelimit.New(limitPerPeer),
	}
}

func (b *loadBalancer) all() []string {
	b.RLock()
	defer b.RUnlock()
	return b.peers
}

// 增加节点
func (b *loadBalancer) add(peer string) {
	b.Lock()
	defer b.Unlock()
	for i := range b.peers {
		if b.peers[i] == peer {
			return
		}
	}
	b.peers = append(b.peers, peer)
	count := limitPerPeer * len(b.peers)
	b.limiter = ratelimit.New(count)
	log.Logger.Infof("Set endorser loadbalance rate limit to %d because peer added", count)
}

// 移除节点
func (b *loadBalancer) remove(peer, reason string) {
	b.Lock()
	defer b.Unlock()
	for i := range b.peers {
		if b.peers[i] == peer {
			b.peers = append(b.peers[:i], b.peers[i+1:]...)
			if len(b.peers) != 0 {
				// 没有可背书节点时不改限流器
				count := limitPerPeer * len(b.peers)
				b.limiter = ratelimit.New(count)
				log.Logger.Infof("Set endorser loadbalance rate limit to %d because %s", count, reason)
			}

			return
		}
	}
}

// 轮训选取节点, 选择背书节点1/3
func (b *loadBalancer) pick() ([]string, int) {
	b.limiter.Take()
	b.RLock()
	defer b.RUnlock()

	total := len(b.peers)
	if total == 0 {
		return nil, 0
	}
	// 取总数的1/3
	count := total / 3
	if count == 0 {
		count = 1
	}
	// 索引越界时归0
	if b.index >= total {
		b.index = 0
	}

	var endorsers []string
	for i := 0; i < count; i++ {
		endorsers = append(endorsers, b.peers[b.index])
		b.index = (b.index + 1) % total
	}

	return endorsers, count
}
