package filter

import (
	"github.com/rongzer/blockchain/peer/filters"
	ab "github.com/rongzer/blockchain/protos/common"
)

// 该过滤规则直接接受消息
type acceptFilter struct{}

func NewAcceptFilter() filters.Filter {
	return &acceptFilter{}
}

func (a acceptFilter) Apply(*ab.Envelope) (filters.Action, filters.Committer) {
	return filters.Accept, NoopCommitter
}

// NoopCommitter 提交为空且非隔离
var NoopCommitter = &noopCommitter{}

type noopCommitter struct{}

func (nc noopCommitter) Commit() {}

func (nc noopCommitter) Isolated() bool { return false }
