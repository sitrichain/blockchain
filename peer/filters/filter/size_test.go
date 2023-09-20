package filter

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/peer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestMaxBytesFilter_LessThan(t *testing.T) {
	f := NewMaxBytesFilter(100)
	action, _ := f.Apply(makeMessage(make([]byte, 20)))
	assert.EqualValues(t, filters.Forward, action)
}

func TestMaxBytesFilter_Exact(t *testing.T) {
	f := NewMaxBytesFilter(100)
	action, _ := f.Apply(makeMessage(make([]byte, 98))) // data=98 最终大小为100
	assert.EqualValues(t, filters.Forward, action)
}

func TestMaxBytesFilter_TooBig(t *testing.T) {
	f := NewMaxBytesFilter(100)
	action, _ := f.Apply(makeMessage(make([]byte, 101)))
	assert.EqualValues(t, filters.Reject, action)
}

func makeMessage(data []byte) *cb.Envelope {
	data, err := proto.Marshal(&cb.Payload{Data: data})
	if err != nil {
		panic(err)
	}
	return &cb.Envelope{Payload: data}
}
