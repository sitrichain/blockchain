package filter

import (
	"github.com/rongzer/blockchain/peer/filters"
	ab "github.com/rongzer/blockchain/protos/common"
)

// 空消息过滤器, 不接受payload为空的消息
type emptyRejectFilter struct{}

func NewEmptyRejectFilter() filters.Filter {
	return &emptyRejectFilter{}
}

func (a emptyRejectFilter) Apply(message *ab.Envelope) (filters.Action, filters.Committer) {
	if message.Payload == nil {
		return filters.Reject, nil
	}
	return filters.Forward, nil
}
