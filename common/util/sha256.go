package util

import (
	"hash"
	"sync"

	"github.com/minio/sha256-simd"
)

var _hashPool = sync.Pool{New: func() interface{} {
	return sha256.New()
}}

// WriteAndSumSha256 写入数据计算SHA256结果
func WriteAndSumSha256(p []byte) []byte {
	s := _hashPool.Get().(hash.Hash)
	s.Reset()
	s.Write(p)
	r := s.Sum(nil)
	_hashPool.Put(s)
	return r
}
