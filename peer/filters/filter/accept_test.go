package filter

import (
	"testing"

	"github.com/rongzer/blockchain/peer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/stretchr/testify/assert"
)

func TestNoopCommitter(t *testing.T) {
	assert.NotNil(t, NoopCommitter)
	NoopCommitter.Commit() // Commit什么也不会做
	assert.False(t, NoopCommitter.Isolated(), "Should return false")
}

func TestNewAcceptFilter(t *testing.T) {
	f := NewAcceptFilter()
	action, _ := f.Apply(&cb.Envelope{})
	assert.EqualValues(t, filters.Accept, action)
}
