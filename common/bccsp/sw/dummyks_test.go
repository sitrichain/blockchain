package sw

import (
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/stretchr/testify/assert"
)

type MockKey struct {
	BytesValue []byte
	BytesErr   error
	Symm       bool
	PK         bccsp.Key
	PKErr      error
}

func (m *MockKey) Bytes() ([]byte, error) {
	return m.BytesValue, m.BytesErr
}

func (*MockKey) SKI() []byte {
	panic("Not yet implemented")
}

func (m *MockKey) Symmetric() bool {
	return m.Symm
}

func (*MockKey) Private() bool {
	panic("Not yet implemented")
}

func (m *MockKey) PublicKey() (bccsp.Key, error) {
	return m.PK, m.PKErr
}

func TestNewDummyKeyStore(t *testing.T) {
	ks := NewDummyKeyStore()
	assert.NotNil(t, ks)
}

func TestDummyKeyStore_GetKey(t *testing.T) {
	ks := NewDummyKeyStore()
	_, err := ks.GetKey([]byte{0, 1, 2, 3, 4})
	assert.Error(t, err)
}

func TestDummyKeyStore_ReadOnly(t *testing.T) {
	ks := NewDummyKeyStore()
	assert.True(t, ks.ReadOnly())
}

func TestDummyKeyStore_StoreKey(t *testing.T) {
	ks := NewDummyKeyStore()
	err := ks.StoreKey(&MockKey{})
	assert.Error(t, err)
}
