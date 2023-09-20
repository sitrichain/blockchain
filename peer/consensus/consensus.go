package consensus

import (
	"github.com/rongzer/blockchain/peer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
)

// Mode 共识模式接口
type Mode interface {
	// NewConsenter 创建共识器实例
	NewConsenter(chain ChainResource, metadata *cb.Metadata) (Consenter, error)
}

// Consenter 共识器接口
type Consenter interface {
	// Order 接收消息进行共识
	Order(env *cb.Envelope, committer filters.Committer) bool
	// Errored 错误通道, 当共识器遇到异常时该通道会关闭
	Errored() <-chan struct{}
	// Start 启动routine开始接收并处理消息
	Start()
	// Halt 释放共识器所用资源
	Halt()
}
