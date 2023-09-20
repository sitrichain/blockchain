package endorse

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/util"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/spf13/viper"
)

// peer列表, 根据peer注册信息而来
type chainPeerList struct {
	sync.RWMutex
	peers      *cb.PeerList  // peer列表
	balancer   *loadBalancer // 负载均衡
	height     uint64        // 链的高度, 从各peer收集来的
	whitePeers string        // 白名单
	blackPeers string        // 黑名单
}

func newChainPeerList() *chainPeerList {
	l := &chainPeerList{
		peers:      &cb.PeerList{},
		balancer:   newLoadBalance(),
		height:     0,
		whitePeers: strings.TrimSpace(viper.GetString("white.endorser.peers")),
		blackPeers: strings.TrimSpace(viper.GetString("black.endorser.peers")),
	}
	// 优先启用白名单
	if len(l.whitePeers) > 0 {
		l.blackPeers = ""
	}
	// 定时评估当前所有peer状态, 将可背书的节点交给负载均衡
	go func() {
		t := time.NewTicker(time.Second * 5)
		for {
			select {
			case <-t.C:
				l.evaluate()
			}
		}
	}()

	return l
}

// 评价各节点状态, 更新可背书的节点地址列表
func (l *chainPeerList) evaluate() {
	nowTime := util.CreateUtcTimestamp()

	l.Lock()
	defer l.Unlock()

	var peers cb.PeerList // 记录新的节点列表
	for _, p := range l.peers.List {
		intervals := nowTime.Seconds - p.SeekTime.Seconds
		if intervals > 3*24*3600 {
			// 节点三天未连接，移除节点
			log.Logger.Warnf("Remove peer[%s] which lost connection more than 3 days", p.Address)
			continue
		} else if intervals > 20 {
			// 20秒未收到节点反馈, 不参与背书
			if p.Status != 3 {
				p.Status = 3
				l.balancer.remove(p.Address, fmt.Sprintf("peer[%s] which lost connection more than 20 seconds", p.Address))
			}
		} else if intervals > 10 { // 10秒未收到节点反馈
			if p.Status != 2 {
				p.Status = 2
			}
		}

		peers.List = append(peers.List, p)
	}
	// 使用新的节点列表
	l.peers = &peers

	return
}

// 查找节点
func (l *chainPeerList) find(addr string) (*cb.PeerInfo, bool) {
	l.RLock()
	defer l.RUnlock()

	for i := range l.peers.List {
		if l.peers.List[i].Address == addr {
			return l.peers.List[i], true
		}
	}
	return nil, false
}

// 增加节点
func (l *chainPeerList) add(pi *cb.PeerInfo) {
	nowTime := util.CreateUtcTimestamp()
	info, ok := l.find(pi.Address)

	l.Lock()
	defer l.Unlock()

	if ok {
		// 更新节点信息
		info.SeekTime = nowTime
		info.BlockInfo = pi.BlockInfo
		if info.Status != 5 {
			if l.height > info.BlockInfo.Height && (l.height-info.BlockInfo.Height) > 600 {
				// 块高度差距600，节点状态繁忙, 不参与背书
				if info.Status != 4 {
					info.Status = 4
					l.balancer.remove(info.Address, fmt.Sprintf("peer[%s] which block height is %d shorter 600 than topest %d", info.Address, info.BlockInfo.Height, l.height))
				}
			} else {
				if info.Status != 1 {
					info.Status = 1
					l.balancer.add(info.Address)
				}
			}
		}
	} else {
		// 新增节点
		pi.Status = 1
		pi.SeekTime = nowTime
		if len(l.whitePeers) > 0 {
			// 有白名单时, 只有在白名单内的节点才允许接收背书
			if strings.Index(l.whitePeers, pi.Address) < 0 {
				pi.Status = 5 // 设为非背书节点
				log.Logger.Infof("Mark peer[%s] do not endorse because not in white list.", pi.Address)
			}
		} else if len(l.blackPeers) > 0 {
			// 有黑名单时, 不在黑名单内的节点才允许接收背书
			if strings.Index(l.blackPeers, pi.Address) >= 0 {
				pi.Status = 5 // 设为非背书节点
				log.Logger.Infof("Mark peer[%s] do not endorse because in black list.", pi.Address)
			}
		}
		if l.height > pi.BlockInfo.Height && (l.height-pi.BlockInfo.Height) > 600 {
			// 块高度差距600，节点状态繁忙, 不参与背书
			pi.Status = 4
			log.Logger.Infof("Mark peer[%s] do not endorse because which block height is %d shorter 600 than topest %d", pi.Address, pi.BlockInfo.Height, l.height)
		}
		if pi.Status == 1 {
			l.balancer.add(pi.Address)
			log.Logger.Infof("Add peer[%s] to endorse list.", pi.Address)
		}

		l.peers.List = append(l.peers.List, pi)
	}

	// 更新链的高度为该节点的高度
	atomic.CompareAndSwapUint64(&l.height, l.height, pi.BlockInfo.Height)
}

// 序列化
func (l *chainPeerList) marshal() ([]byte, error) {
	l.RLock()
	defer l.RUnlock()
	return l.peers.Marshal()
}
