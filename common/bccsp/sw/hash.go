package sw

import (
	"hash"

	"github.com/rongzer/blockchain/common/bccsp"
)

type hasher struct {
	hash func() hash.Hash
}

func (c *hasher) Hash(msg []byte, _ bccsp.HashOpts) (hash []byte, err error) {
	h := c.hash()
	h.Write(msg)
	return h.Sum(nil), nil
}

func (c *hasher) GetHash(_ bccsp.HashOpts) (h hash.Hash, err error) {
	return c.hash(), nil
}
