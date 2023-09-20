package sw

import (
	"errors"
	"hash"
	"reflect"
	"testing"

	"github.com/minio/sha256-simd"
	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/stretchr/testify/assert"
)

type HashOpts struct{}

func (HashOpts) Algorithm() string {
	return "Mock HashOpts"
}

type MockHasher struct {
	MsgArg  []byte
	OptsArg bccsp.HashOpts

	Value     []byte
	ValueHash hash.Hash
	Err       error
}

func (h *MockHasher) Hash(msg []byte, opts bccsp.HashOpts) (hash []byte, err error) {
	if !reflect.DeepEqual(h.MsgArg, msg) {
		return nil, errors.New("invalid message")
	}
	if !reflect.DeepEqual(h.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return h.Value, h.Err
}

func (h *MockHasher) GetHash(opts bccsp.HashOpts) (hash.Hash, error) {
	if !reflect.DeepEqual(h.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return h.ValueHash, h.Err
}

func TestHash(t *testing.T) {
	expectetMsg := []byte{1, 2, 3, 4}
	expectedOpts := &HashOpts{}
	expectetValue := []byte{1, 2, 3, 4, 5}
	expectedErr := errors.New("Expected Error")

	hashers := make(map[reflect.Type]Hasher)
	hashers[reflect.TypeOf(&HashOpts{})] = &MockHasher{
		MsgArg:  expectetMsg,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp := impl{hashers: hashers}
	value, err := csp.Hash(expectetMsg, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)

	hashers = make(map[reflect.Type]Hasher)
	hashers[reflect.TypeOf(&HashOpts{})] = &MockHasher{
		MsgArg:  expectetMsg,
		OptsArg: expectedOpts,
		Value:   nil,
		Err:     expectedErr,
	}
	csp = impl{hashers: hashers}
	value, err = csp.Hash(expectetMsg, expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())
}

func TestGetHash(t *testing.T) {
	expectedOpts := &HashOpts{}
	expectetValue := sha256.New()
	expectedErr := errors.New("Expected Error")

	hashers := make(map[reflect.Type]Hasher)
	hashers[reflect.TypeOf(&HashOpts{})] = &MockHasher{
		OptsArg:   expectedOpts,
		ValueHash: expectetValue,
		Err:       nil,
	}
	csp := impl{hashers: hashers}
	value, err := csp.GetHash(expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)

	hashers = make(map[reflect.Type]Hasher)
	hashers[reflect.TypeOf(&HashOpts{})] = &MockHasher{
		OptsArg:   expectedOpts,
		ValueHash: expectetValue,
		Err:       expectedErr,
	}
	csp = impl{hashers: hashers}
	value, err = csp.GetHash(expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())
}

func TestHasher(t *testing.T) {
	hasher := &hasher{hash: sha256.New}

	msg := []byte("Hello World")
	out, err := hasher.Hash(msg, nil)
	assert.NoError(t, err)
	h := sha256.New()
	h.Write(msg)
	out2 := h.Sum(nil)
	assert.Equal(t, out, out2)

	hf, err := hasher.GetHash(nil)
	assert.NoError(t, err)
	assert.Equal(t, hf, sha256.New())
}
