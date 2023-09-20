package sw

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"strings"
	"testing"

	"github.com/minio/sha256-simd"
	"github.com/stretchr/testify/assert"
)

type MockSignerOpts struct {
	HashFuncValue crypto.Hash
}

func (o *MockSignerOpts) HashFunc() crypto.Hash {
	return o.HashFuncValue
}

func TestRSAPrivateKey(t *testing.T) {
	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	k := &rsaPrivateKey{lowLevelKey}

	assert.False(t, k.Symmetric())
	assert.True(t, k.Private())

	_, err = k.Bytes()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Not supported.")

	k.privKey = nil
	ski := k.SKI()
	assert.Nil(t, ski)

	k.privKey = lowLevelKey
	ski = k.SKI()
	raw, _ := asn1.Marshal(rsaPublicKeyASN{N: k.privKey.N, E: k.privKey.E})
	hash := sha256.New()
	hash.Write(raw)
	ski2 := hash.Sum(nil)
	assert.Equal(t, ski2, ski, "SKI is not computed in the right way.")

	pk, err := k.PublicKey()
	assert.NoError(t, err)
	assert.NotNil(t, pk)
	ecdsaPK, ok := pk.(*rsaPublicKey)
	assert.True(t, ok)
	assert.Equal(t, &lowLevelKey.PublicKey, ecdsaPK.pubKey)
}

func TestRSAPublicKey(t *testing.T) {
	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	k := &rsaPublicKey{&lowLevelKey.PublicKey}

	assert.False(t, k.Symmetric())
	assert.False(t, k.Private())

	k.pubKey = nil
	ski := k.SKI()
	assert.Nil(t, ski)

	k.pubKey = &lowLevelKey.PublicKey
	ski = k.SKI()
	raw, _ := asn1.Marshal(rsaPublicKeyASN{N: k.pubKey.N, E: k.pubKey.E})
	hash := sha256.New()
	hash.Write(raw)
	ski2 := hash.Sum(nil)
	assert.Equal(t, ski, ski2, "SKI is not computed in the right way.")

	pk, err := k.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, k, pk)

	bytes, err := k.Bytes()
	assert.NoError(t, err)
	bytes2, err := x509.MarshalPKIXPublicKey(k.pubKey)
	assert.Equal(t, bytes2, bytes, "bytes are not computed in the right way.")
}

func TestRSASignerSign(t *testing.T) {
	signer := &rsaSigner{}
	verifierPrivateKey := &rsaPrivateKeyVerifier{}
	verifierPublicKey := &rsaPublicKeyKeyVerifier{}

	// Generate a key
	lowLevelKey, err := rsa.GenerateKey(rand.Reader, 1024)
	assert.NoError(t, err)
	k := &rsaPrivateKey{lowLevelKey}
	pk, err := k.PublicKey()
	assert.NoError(t, err)

	// Sign
	msg := []byte("Hello World!!!")

	_, err = signer.Sign(k, msg, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid options. Must be different from nil.")

	_, err = signer.Sign(k, msg, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	assert.Error(t, err)

	hf := sha256.New()
	hf.Write(msg)
	digest := hf.Sum(nil)
	sigma, err := signer.Sign(k, digest, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256})
	assert.NoError(t, err)

	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: crypto.SHA256}
	// Verify against msg, must fail
	err = rsa.VerifyPSS(&lowLevelKey.PublicKey, crypto.SHA256, msg, sigma, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "crypto/rsa: verification error")

	// Verify against digest, must succeed
	err = rsa.VerifyPSS(&lowLevelKey.PublicKey, crypto.SHA256, digest, sigma, opts)
	assert.NoError(t, err)

	valid, err := verifierPrivateKey.Verify(k, sigma, msg, opts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "crypto/rsa: verification error"))

	valid, err = verifierPrivateKey.Verify(k, sigma, digest, opts)
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = verifierPublicKey.Verify(pk, sigma, msg, opts)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "crypto/rsa: verification error"))

	valid, err = verifierPublicKey.Verify(pk, sigma, digest, opts)
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestRSAVerifiersInvalidInputs(t *testing.T) {
	verifierPrivate := &rsaPrivateKeyVerifier{}
	_, err := verifierPrivate.Verify(nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Invalid options. It must not be nil."))

	_, err = verifierPrivate.Verify(nil, nil, nil, &MockSignerOpts{})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Opts type not recognized ["))

	verifierPublic := &rsaPublicKeyKeyVerifier{}
	_, err = verifierPublic.Verify(nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Invalid options. It must not be nil."))

	_, err = verifierPublic.Verify(nil, nil, nil, &MockSignerOpts{})
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Opts type not recognized ["))
}
