package signer

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"hash"
	"reflect"
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/bccsp/utils"
	"github.com/stretchr/testify/assert"
)

type MockBCCSP struct {
	SignArgKey    bccsp.Key
	SignDigestArg []byte
	SignOptsArg   bccsp.SignerOpts

	SignValue []byte
	SignErr   error

	VerifyValue bool
	VerifyErr   error
}

func (*MockBCCSP) KeyGen(_ bccsp.KeyGenOpts) (k bccsp.Key, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) KeyDeriv(_ bccsp.Key, _ bccsp.KeyDerivOpts) (dk bccsp.Key, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) KeyImport(_ interface{}, _ bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) GetKey(_ []byte) (k bccsp.Key, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) Hash(_ []byte, _ bccsp.HashOpts) (hash []byte, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) GetHash(_ bccsp.HashOpts) (h hash.Hash, err error) {
	panic("Not yet implemented")
}

func (b *MockBCCSP) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	if !reflect.DeepEqual(b.SignArgKey, k) {
		return nil, errors.New("invalid key")
	}
	if !reflect.DeepEqual(b.SignDigestArg, digest) {
		return nil, errors.New("invalid digest")
	}
	if !reflect.DeepEqual(b.SignOptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return b.SignValue, b.SignErr
}

func (b *MockBCCSP) Verify(_ bccsp.Key, _, _ []byte, _ bccsp.SignerOpts) (valid bool, err error) {
	return b.VerifyValue, b.VerifyErr
}

func (*MockBCCSP) Encrypt(_ bccsp.Key, _ []byte, _ bccsp.EncrypterOpts) (ciphertext []byte, err error) {
	panic("Not yet implemented")
}

func (*MockBCCSP) Decrypt(_ bccsp.Key, _ []byte, _ bccsp.DecrypterOpts) (plaintext []byte, err error) {
	panic("Not yet implemented")
}

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

type MockSignerOpts struct {
	HashFuncValue crypto.Hash
}

func (o *MockSignerOpts) HashFunc() crypto.Hash {
	return o.HashFuncValue
}

func TestInitFailures(t *testing.T) {
	_, err := New(nil, &MockKey{})
	assert.Error(t, err)

	_, err = New(&MockBCCSP{}, nil)
	assert.Error(t, err)

	_, err = New(&MockBCCSP{}, &MockKey{Symm: true})
	assert.Error(t, err)

	_, err = New(&MockBCCSP{}, &MockKey{PKErr: errors.New("No PK")})
	assert.Error(t, err)
	assert.Equal(t, "failed getting public key [No PK]", err.Error())

	_, err = New(&MockBCCSP{}, &MockKey{PK: &MockKey{BytesErr: errors.New("No bytes")}})
	assert.Error(t, err)
	assert.Equal(t, "failed marshalling public key [No bytes]", err.Error())

	_, err = New(&MockBCCSP{}, &MockKey{PK: &MockKey{BytesValue: []byte{0, 1, 2, 3}}})
	assert.Error(t, err)
}

func TestInit(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)
	pkRaw, err := utils.PublicKeyToDER(&k.PublicKey)
	assert.NoError(t, err)

	signer, err := New(&MockBCCSP{}, &MockKey{PK: &MockKey{BytesValue: pkRaw}})
	assert.NoError(t, err)
	assert.NotNil(t, signer)

	// Test public key
	R, S, err := ecdsa.Sign(rand.Reader, k, []byte{0, 1, 2, 3})
	assert.NoError(t, err)

	assert.True(t, ecdsa.Verify(signer.Public().(*ecdsa.PublicKey), []byte{0, 1, 2, 3}, R, S))
}

func TestPublic(t *testing.T) {
	pk := &MockKey{}
	signer := &bccspCryptoSigner{pk: pk}

	pk2 := signer.Public()
	assert.NotNil(t, pk, pk2)
}

func TestSign(t *testing.T) {
	expectedSig := []byte{0, 1, 2, 3, 4}
	expectedKey := &MockKey{}
	expectedDigest := []byte{0, 1, 2, 3, 4, 5}
	expectedOpts := &MockSignerOpts{}

	signer := &bccspCryptoSigner{
		key: expectedKey,
		csp: &MockBCCSP{
			SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts,
			SignValue: expectedSig}}
	signature, err := signer.Sign(nil, expectedDigest, expectedOpts)
	assert.NoError(t, err)
	assert.Equal(t, expectedSig, signature)

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &MockBCCSP{
			SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts,
			SignErr: errors.New("no signature")}}
	signature, err = signer.Sign(nil, expectedDigest, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "no signature")

	signer = &bccspCryptoSigner{
		key: nil,
		csp: &MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, expectedDigest, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid key")

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, nil, expectedOpts)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid digest")

	signer = &bccspCryptoSigner{
		key: expectedKey,
		csp: &MockBCCSP{SignArgKey: expectedKey, SignDigestArg: expectedDigest, SignOptsArg: expectedOpts}}
	_, err = signer.Sign(nil, expectedDigest, nil)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid opts")
}
