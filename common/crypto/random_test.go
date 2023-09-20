package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomBytes(t *testing.T) {
	a, _ := GetRandomBytes(33)
	assert.Equal(t, 33, len(a))
	b, _ := GetRandomBytes(33)
	assert.Equal(t, 33, len(b))
	assert.NotEqual(t, a, b)
}

func TestGetRandomNonce(t *testing.T) {
	a, _ := GetRandomNonce()
	assert.Equal(t, NonceSize, len(a))
}
