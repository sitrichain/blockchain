package filters

import (
	ab "github.com/rongzer/blockchain/protos/common"
)

// Action 过滤结果
type Action uint8

const (
	// Accept 标识该消息应当被处理
	Accept = iota
	// Reject 标识该消息不该被处理
	Reject
	// Forward 标识当前规则没法决定结果, 交给下一个规则
	Forward
)

// Filter 过滤规则接口, 决定是否应该处理收到的Envelope消息
type Filter interface {
	// Apply 对收到的消息进行过滤, 如果接受了某个消息, 需要提供committer供后续写入链
	Apply(message *ab.Envelope) (Action, Committer)
}

// Committer 消息确认接口, 被过滤器验证通过后返回, 在写入链时被调用一次
type Committer interface {
	// Commit 确认提交消息后要做的所有操作
	Commit()
	// Isolated 标识该交易是否需要独立落块
	Isolated() bool
}
