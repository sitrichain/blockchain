package factory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactoryOptsFactoryName(t *testing.T) {
	assert.Equal(t, GetDefaultOpts().FactoryName(), "SW")
}
