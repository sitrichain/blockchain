package sw

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/stretchr/testify/assert"
)

type MockKeyDerivOpts struct {
	EphemeralValue bool
}

func (*MockKeyDerivOpts) Algorithm() string {
	return "Mock KeyDerivOpts"
}

func (o *MockKeyDerivOpts) Ephemeral() bool {
	return o.EphemeralValue
}

type MockKeyDeriver struct {
	KeyArg  bccsp.Key
	OptsArg bccsp.KeyDerivOpts

	Value bccsp.Key
	Err   error
}

func (kd *MockKeyDeriver) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (dk bccsp.Key, err error) {
	if !reflect.DeepEqual(kd.KeyArg, k) {
		return nil, errors.New("invalid key")
	}
	if !reflect.DeepEqual(kd.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return kd.Value, kd.Err
}

func TestKeyDeriv(t *testing.T) {
	expectedKey := &MockKey{BytesValue: []byte{1, 2, 3}}
	expectedOpts := &MockKeyDerivOpts{EphemeralValue: true}
	expectetValue := &MockKey{BytesValue: []byte{1, 2, 3, 4, 5}}
	expectedErr := errors.New("Expected Error")

	keyDerivers := make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&MockKey{})] = &MockKeyDeriver{
		KeyArg:  expectedKey,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     expectedErr,
	}
	csp := impl{keyDerivers: keyDerivers}
	value, err := csp.KeyDeriv(expectedKey, expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())

	keyDerivers = make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&MockKey{})] = &MockKeyDeriver{
		KeyArg:  expectedKey,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp = impl{keyDerivers: keyDerivers}
	value, err = csp.KeyDeriv(expectedKey, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)
}

func TestECDSAPublicKeyKeyDeriver(t *testing.T) {
	kd := ecdsaPublicKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&ecdsaPublicKey{}, &MockKeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}

func TestECDSAPrivateKeyKeyDeriver(t *testing.T) {
	kd := ecdsaPrivateKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&ecdsaPrivateKey{}, &MockKeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}

func TestAESPrivateKeyKeyDeriver(t *testing.T) {
	kd := aesPrivateKeyKeyDeriver{}

	_, err := kd.KeyDeriv(&MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts parameter. It must not be nil.")

	_, err = kd.KeyDeriv(&aesPrivateKey{}, &MockKeyDerivOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyDerivOpts' provided [")
}
