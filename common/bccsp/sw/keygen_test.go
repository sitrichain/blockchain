package sw

import (
	"crypto/elliptic"
	"errors"
	"reflect"
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/stretchr/testify/assert"
)

type MockKeyGenOpts struct {
	EphemeralValue bool
}

func (*MockKeyGenOpts) Algorithm() string {
	return "Mock KeyGenOpts"
}

func (o *MockKeyGenOpts) Ephemeral() bool {
	return o.EphemeralValue
}

type MockKeyGenerator struct {
	OptsArg bccsp.KeyGenOpts

	Value bccsp.Key
	Err   error
}

func (kg *MockKeyGenerator) KeyGen(opts bccsp.KeyGenOpts) (k bccsp.Key, err error) {
	if !reflect.DeepEqual(kg.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return kg.Value, kg.Err
}

func TestKeyGen(t *testing.T) {
	expectedOpts := &MockKeyGenOpts{EphemeralValue: true}
	expectetValue := &MockKey{}
	expectedErr := errors.New("Expected Error")

	keyGenerators := make(map[reflect.Type]KeyGenerator)
	keyGenerators[reflect.TypeOf(&MockKeyGenOpts{})] = &MockKeyGenerator{
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     expectedErr,
	}
	csp := impl{keyGenerators: keyGenerators}
	value, err := csp.KeyGen(expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())

	keyGenerators = make(map[reflect.Type]KeyGenerator)
	keyGenerators[reflect.TypeOf(&MockKeyGenOpts{})] = &MockKeyGenerator{
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp = impl{keyGenerators: keyGenerators}
	value, err = csp.KeyGen(expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)
}

func TestECDSAKeyGenerator(t *testing.T) {
	kg := &ecdsaKeyGenerator{curve: elliptic.P256()}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	ecdsaK, ok := k.(*ecdsaPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, ecdsaK.privKey)
	assert.Equal(t, ecdsaK.privKey.Curve, elliptic.P256())
}

func TestRSAKeyGenerator(t *testing.T) {
	kg := &rsaKeyGenerator{length: 512}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	rsaK, ok := k.(*rsaPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, rsaK.privKey)
	assert.Equal(t, rsaK.privKey.N.BitLen(), 512)
}

func TestAESKeyGenerator(t *testing.T) {
	kg := &aesKeyGenerator{length: 32}

	k, err := kg.KeyGen(nil)
	assert.NoError(t, err)

	aesK, ok := k.(*aesPrivateKey)
	assert.True(t, ok)
	assert.NotNil(t, aesK.privKey)
	assert.Equal(t, len(aesK.privKey), 32)
}

func TestAESKeyGeneratorInvalidInputs(t *testing.T) {
	kg := &aesKeyGenerator{length: -1}

	_, err := kg.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Len must be larger than 0")
}

func TestRSAKeyGeneratorInvalidInputs(t *testing.T) {
	kg := &rsaKeyGenerator{length: -1}

	_, err := kg.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed generating RSA -1 key")
}
