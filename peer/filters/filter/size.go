package filter

import (
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/filters"
	ab "github.com/rongzer/blockchain/protos/common"
)

// 消息大小过滤器
type maxBytesFilter struct {
	maxBytes uint32
}

// NewMaxBytesFilter rejects messages larger than maxBytes
func NewMaxBytesFilter(maxBytes uint32) filters.Filter {
	return &maxBytesFilter{maxBytes: maxBytes}
}

func (r *maxBytesFilter) Apply(message *ab.Envelope) (filters.Action, filters.Committer) {
	if size := messageByteSize(message); size > r.maxBytes {
		log.Logger.Warnf("%d byte message payload exceeds maximum allowed %d bytes", size, r.maxBytes)
		return filters.Reject, nil
	}
	return filters.Forward, nil
}

func messageByteSize(message *ab.Envelope) uint32 {
	return uint32(len(message.Payload) + len(message.Signature))
}
