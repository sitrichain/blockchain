package sw

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/stretchr/testify/assert"
)

type MockEncryptor struct {
	KeyArg       bccsp.Key
	PlaintextArg []byte
	OptsArg      bccsp.EncrypterOpts

	EncValue []byte
	EncErr   error
}

func (e *MockEncryptor) Encrypt(k bccsp.Key, plaintext []byte, opts bccsp.EncrypterOpts) (ciphertext []byte, err error) {
	if !reflect.DeepEqual(e.KeyArg, k) {
		return nil, errors.New("invalid key")
	}
	if !reflect.DeepEqual(e.PlaintextArg, plaintext) {
		return nil, errors.New("invalid plaintext")
	}
	if !reflect.DeepEqual(e.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return e.EncValue, e.EncErr
}

type MockSigner struct {
	KeyArg    bccsp.Key
	DigestArg []byte
	OptsArg   bccsp.SignerOpts

	Value []byte
	Err   error
}

func (s *MockSigner) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	if !reflect.DeepEqual(s.KeyArg, k) {
		return nil, errors.New("invalid key")
	}
	if !reflect.DeepEqual(s.DigestArg, digest) {
		return nil, errors.New("invalid digest")
	}
	if !reflect.DeepEqual(s.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return s.Value, s.Err
}

type MockKeyStore struct {
	GetKeyValue bccsp.Key
	GetKeyErr   error
	StoreKeyErr error
}

func (*MockKeyStore) ReadOnly() bool {
	panic("Not yet implemented")
}

func (ks *MockKeyStore) GetKey(_ []byte) (k bccsp.Key, err error) {
	return ks.GetKeyValue, ks.GetKeyErr
}

func (ks *MockKeyStore) StoreKey(_ bccsp.Key) (err error) {
	return ks.StoreKeyErr
}

type MockHashOpts struct{}

func (MockHashOpts) Algorithm() string {
	return "Mock HashOpts"
}

type MockVerifier struct {
	KeyArg       bccsp.Key
	SignatureArg []byte
	DigestArg    []byte
	OptsArg      bccsp.SignerOpts

	Value bool
	Err   error
}

func (s *MockVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	if !reflect.DeepEqual(s.KeyArg, k) {
		return false, errors.New("invalid key")
	}
	if !reflect.DeepEqual(s.SignatureArg, signature) {
		return false, errors.New("invalid signature")
	}
	if !reflect.DeepEqual(s.DigestArg, digest) {
		return false, errors.New("invalid digest")
	}
	if !reflect.DeepEqual(s.OptsArg, opts) {
		return false, errors.New("invalid opts")
	}

	return s.Value, s.Err
}

func TestEncrypt(t *testing.T) {
	expectedKey := &MockKey{}
	expectedPlaintext := []byte{1, 2, 3, 4}
	expectedOpts := &MockEncrypterOpts{}
	expectedCiphertext := []byte{0, 1, 2, 3, 4}
	expectedErr := errors.New("no error")

	encryptors := make(map[reflect.Type]Encryptor)
	encryptors[reflect.TypeOf(&MockKey{})] = &MockEncryptor{
		KeyArg:       expectedKey,
		PlaintextArg: expectedPlaintext,
		OptsArg:      expectedOpts,
		EncValue:     expectedCiphertext,
		EncErr:       expectedErr,
	}

	csp := impl{encryptors: encryptors}

	ct, err := csp.Encrypt(expectedKey, expectedPlaintext, expectedOpts)
	assert.Equal(t, expectedCiphertext, ct)
	assert.Equal(t, expectedErr, err)
}

func TestSign(t *testing.T) {
	expectedKey := &MockKey{}
	expectetDigest := []byte{1, 2, 3, 4}
	expectedOpts := &MockSignerOpts{}
	expectetValue := []byte{0, 1, 2, 3, 4}
	expectedErr := errors.New("Expected Error")

	signers := make(map[reflect.Type]Signer)
	signers[reflect.TypeOf(&MockKey{})] = &MockSigner{
		KeyArg:    expectedKey,
		DigestArg: expectetDigest,
		OptsArg:   expectedOpts,
		Value:     expectetValue,
		Err:       nil,
	}
	csp := impl{signers: signers}
	value, err := csp.Sign(expectedKey, expectetDigest, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)

	signers = make(map[reflect.Type]Signer)
	signers[reflect.TypeOf(&MockKey{})] = &MockSigner{
		KeyArg:    expectedKey,
		DigestArg: expectetDigest,
		OptsArg:   expectedOpts,
		Value:     nil,
		Err:       expectedErr,
	}
	csp = impl{signers: signers}
	value, err = csp.Sign(expectedKey, expectetDigest, expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())
}

func TestKeyGenInvalidInputs(t *testing.T) {
	// Init a BCCSP instance with a key store that returns an error on store
	csp, err := New(256, "SHA2", &MockKeyStore{StoreKeyErr: errors.New("cannot store key")})
	assert.NoError(t, err)

	_, err = csp.KeyGen(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Opts parameter. It must not be nil.")

	_, err = csp.KeyGen(&MockKeyGenOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyGenOpts' provided [")

	_, err = csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{})
	assert.Error(t, err, "Generation of a non-ephemeral key must fail. KeyStore is programmed to fail.")
	assert.Contains(t, err.Error(), "cannot store key")
}

func TestKeyDerivInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{StoreKeyErr: errors.New("cannot store key")})
	assert.NoError(t, err)

	_, err = csp.KeyDeriv(nil, &bccsp.ECDSAReRandKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.KeyDeriv(&MockKey{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.KeyDeriv(&MockKey{}, &bccsp.ECDSAReRandKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'Key' provided [")

	keyDerivers := make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&MockKey{})] = &MockKeyDeriver{
		KeyArg:  &MockKey{},
		OptsArg: &MockKeyDerivOpts{EphemeralValue: false},
		Value:   nil,
		Err:     nil,
	}
	csp.(*impl).keyDerivers = keyDerivers
	_, err = csp.KeyDeriv(&MockKey{}, &MockKeyDerivOpts{EphemeralValue: false})
	assert.Error(t, err, "KeyDerivation of a non-ephemeral key must fail. KeyStore is programmed to fail.")
	assert.Contains(t, err.Error(), "cannot store key")
}

func TestKeyImportInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.KeyImport(nil, &bccsp.AES256ImportKeyOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw. It must not be nil.")

	_, err = csp.KeyImport([]byte{0, 1, 2, 3, 4}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.KeyImport([]byte{0, 1, 2, 3, 4}, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'KeyImportOpts' provided [")
}

func TestGetKeyInvalidInputs(t *testing.T) {
	// Init a BCCSP instance with a key store that returns an error on get
	csp, err := New(256, "SHA2", &MockKeyStore{GetKeyErr: errors.New("cannot get key")})
	assert.NoError(t, err)

	_, err = csp.GetKey(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot get key")

	// Init a BCCSP instance with a key store that returns a given key
	k := &MockKey{}
	csp, err = New(256, "SHA2", &MockKeyStore{GetKeyValue: k})
	assert.NoError(t, err)
	// No SKI is needed here
	k2, err := csp.GetKey(nil)
	assert.NoError(t, err)
	assert.Equal(t, k, k2, "Keys must be the same.")
}

func TestSignInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.Sign(nil, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Sign(&MockKey{}, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid digest. Cannot be empty.")

	_, err = csp.Sign(&MockKey{}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'SignKey' provided [")
}

func TestVerifyInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.Verify(nil, []byte{1, 2, 3, 5}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Verify(&MockKey{}, nil, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid signature. Cannot be empty.")

	_, err = csp.Verify(&MockKey{}, []byte{1, 2, 3, 5}, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid digest. Cannot be empty.")

	_, err = csp.Verify(&MockKey{}, []byte{1, 2, 3, 5}, []byte{1, 2, 3, 5}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'VerifyKey' provided [")
}

func TestEncryptInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.Encrypt(nil, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Encrypt(&MockKey{}, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'EncryptKey' provided [")
}

func TestDecryptInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.Decrypt(nil, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key. It must not be nil.")

	_, err = csp.Decrypt(&MockKey{}, []byte{1, 2, 3, 4}, &bccsp.AESCBCPKCS7ModeOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'DecryptKey' provided [")
}

func TestHashInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.Hash(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.Hash(nil, &MockHashOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'HashOpt' provided [")
}

func TestGetHashInvalidInputs(t *testing.T) {
	csp, err := New(256, "SHA2", &MockKeyStore{})
	assert.NoError(t, err)

	_, err = csp.GetHash(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid opts. It must not be nil.")

	_, err = csp.GetHash(&MockHashOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported 'HashOpt' provided [")
}

func TestVerify(t *testing.T) {
	expectedKey := &MockKey{}
	expectetSignature := []byte{1, 2, 3, 4, 5}
	expectetDigest := []byte{1, 2, 3, 4}
	expectedOpts := &MockSignerOpts{}
	expectetValue := true
	expectedErr := errors.New("Expected Error")

	verifiers := make(map[reflect.Type]Verifier)
	verifiers[reflect.TypeOf(&MockKey{})] = &MockVerifier{
		KeyArg:       expectedKey,
		SignatureArg: expectetSignature,
		DigestArg:    expectetDigest,
		OptsArg:      expectedOpts,
		Value:        expectetValue,
		Err:          nil,
	}
	csp := impl{verifiers: verifiers}
	value, err := csp.Verify(expectedKey, expectetSignature, expectetDigest, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)

	verifiers = make(map[reflect.Type]Verifier)
	verifiers[reflect.TypeOf(&MockKey{})] = &MockVerifier{
		KeyArg:       expectedKey,
		SignatureArg: expectetSignature,
		DigestArg:    expectetDigest,
		OptsArg:      expectedOpts,
		Value:        false,
		Err:          expectedErr,
	}
	csp = impl{verifiers: verifiers}
	value, err = csp.Verify(expectedKey, expectetSignature, expectetDigest, expectedOpts)
	assert.False(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())
}
