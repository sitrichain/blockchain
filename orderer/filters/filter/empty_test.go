package filter

import (
	"testing"

	"github.com/rongzer/blockchain/orderer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestNewEmptyRejectFilter(t *testing.T) {
	f := NewEmptyRejectFilter()

	result, _ := f.Apply(&cb.Envelope{})
	assert.EqualValues(t, filters.Reject, result)

	result, _ = f.Apply(&cb.Envelope{Payload: []byte("fakedata")})
	assert.EqualValues(t, filters.Forward, result)
}
