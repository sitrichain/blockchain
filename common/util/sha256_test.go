package util

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func originalSha(p []byte) []byte {
	s := sha256.New()
	s.Write(p)
	return s.Sum(nil)
}

func TestWriteAndSumSha256(t *testing.T) {
	a := []byte("123456789012345678901234567890123456789012345678901234567890")
	assert.EqualValues(t, originalSha(a), WriteAndSumSha256(a), "sha256 result different")

	b := []byte("Accelerate SHA256 computations in pure Go using AVX512, SHA Extensions and AVX2 for Intel and ARM64 for ARM. On AVX512 it provides an up to 8x improvement (over 3 GB/s per core) in comparison to AVX2. SHA Extensions give a performance boost of close to 4x over AVX2.")
	assert.EqualValues(t, originalSha(b), WriteAndSumSha256(b), "sha256 result different")
}

func BenchmarkWriteAndSumSha256(b *testing.B) {
	a := []byte("123456789012345678901234567890123456789012345678901234567890")
	for i := 0; i < b.N; i++ {
		WriteAndSumSha256(a)
	}
}
