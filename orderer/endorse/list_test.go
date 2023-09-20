package endorse

import (
	"testing"
	"time"

	"github.com/rongzer/blockchain/common/util"
	"github.com/rongzer/blockchain/protos/common"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// 黑名单上的节点就算注册节点信息也不允许背书
func TestChainPeerList_BlackList(t *testing.T) {
	viper.Set("black.endorser.peers", "127.0.0.1")

	l := newChainPeerList()
	node := &common.PeerInfo{
		Id:      "a",
		Address: "127.0.0.1",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(node)

	assert.Equal(t, 5, int(l.peers.List[0].Status))
}

// 非白名单上的节点就算注册节点信息也不允许背书
func TestChainPeerList_WhiteList(t *testing.T) {
	viper.Set("white.endorser.peers", "127.0.0.1")

	l := newChainPeerList()
	node := &common.PeerInfo{
		Id:      "a",
		Address: "127.0.0.2",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(node)

	assert.Equal(t, 5, int(l.peers.List[0].Status))
}

// 白名单优先级高于黑名单
func TestChainPeerList_BlackWhiteListPriority(t *testing.T) {
	viper.Set("black.endorser.peers", "127.0.0.1")
	viper.Set("white.endorser.peers", "127.0.0.1,192.168.0.1")

	l := newChainPeerList()
	assert.Empty(t, l.blackPeers, "set black and white list at same time")
	assert.EqualValues(t, "127.0.0.1,192.168.0.1", l.whitePeers, "set black and white list at same time")
}

// 添加节点, 多次添加同地址节点, 只修改信息BlockInfo和SeekTime
func TestChainPeerList_AddNodeWithSameAddress(t *testing.T) {
	l := newChainPeerList()

	nodeA := &common.PeerInfo{
		Id:      "a",
		Address: "127.0.0.1",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(nodeA)
	assert.Contains(t, l.peers.List, nodeA)

	thisTime := util.CreateUtcTimestamp()
	nodeA_changed := &common.PeerInfo{
		Id:      "b",
		Address: "127.0.0.1",
		BlockInfo: &common.BlockchainInfo{
			Height: 5,
		},
	}
	l.add(nodeA_changed)

	assert.Equal(t, 1, len(l.peers.List))
	assert.EqualValues(t, thisTime.Seconds, l.peers.List[0].SeekTime.Seconds)
	assert.EqualValues(t, "a", l.peers.List[0].Id)
}

//  添加节点, 账本高度随最高节点高度增长
func TestChainPeerList_LedgerHeight(t *testing.T) {
	l := newChainPeerList()

	for i := 1; i <= 1000; i++ {
		node := &common.PeerInfo{
			Id:      "a",
			Address: "127.0.0.1",
			BlockInfo: &common.BlockchainInfo{
				Height: uint64(i),
			},
		}
		l.add(node)
		assert.EqualValues(t, i, int(l.height))
	}
}

// 序列化列表
func TestChainPeerList_Marshal(t *testing.T) {
	l := newChainPeerList()
	b, err := l.marshal()
	assert.NoError(t, err)
	assert.Empty(t, b)

	node := &common.PeerInfo{
		Id:      "a",
		Address: "127.0.0.2",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(node)
	b, err = l.marshal()
	assert.NoError(t, err)

	a := &cb.PeerList{}
	assert.NoError(t, a.Unmarshal(b))
	assert.EqualValues(t, node.Id, a.List[0].Id)
	assert.EqualValues(t, node.Address, a.List[0].Address)
	assert.EqualValues(t, node.BlockInfo.Height, a.List[0].BlockInfo.Height)
	assert.EqualValues(t, node.SeekTime.Seconds, a.List[0].SeekTime.Seconds)
}

// 状态定时自动变更
func TestChainPeerList_Ticker(t *testing.T) {
	viper.Set("black.endorser.peers", "127.0.0.3")

	l := newChainPeerList()

	node := &common.PeerInfo{
		Id:      "a",
		Address: "127.0.0.1",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(node)
	// 刚加入
	assert.EqualValues(t, 1, int(l.peers.List[0].Status))
	time.Sleep(time.Second)
	// 第一次状态检查
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 1, int(l.peers.List[0].Status))
	// 第二次状态检查
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 1, int(l.peers.List[0].Status))
	// 第三次状态检查
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 2, int(l.peers.List[0].Status))
	// 第四次状态检查
	node = &common.PeerInfo{
		Id:      "b",
		Address: "127.0.0.2",
		BlockInfo: &common.BlockchainInfo{
			Height: 1,
		},
	}
	l.add(node)
	l.peers.List[1].SeekTime.Seconds = l.peers.List[1].SeekTime.Seconds - 3*24*3600 - 5
	assert.EqualValues(t, 2, len(l.peers.List))
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 1, len(l.peers.List))
	// 第五次状态检查
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 3, int(l.peers.List[0].Status))
	// 第六次状态检查
	node = &common.PeerInfo{
		Id:      "c",
		Address: "127.0.0.3",
		BlockInfo: &common.BlockchainInfo{
			Height: 602,
		},
		SeekTime: util.CreateUtcTimestamp(),
	}
	l.add(node)
	time.Sleep(time.Second * 5)
	assert.EqualValues(t, 4, int(l.peers.List[0].Status))
	assert.EqualValues(t, 5, int(l.peers.List[1].Status))
}
