package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClone(t *testing.T) {
	src := []byte{0, 1, 2, 3, 4}
	clone := Clone(src)
	assert.Equal(t, src, clone)
}
