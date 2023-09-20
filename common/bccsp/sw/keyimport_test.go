package sw

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"reflect"
	"testing"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/bccsp/utils"
	"github.com/rongzer/blockchain/common/gm"
	"github.com/stretchr/testify/assert"
)

type MockKeyImporter struct {
	RawArg  []byte
	OptsArg bccsp.KeyImportOpts
	Value   bccsp.Key
	Err     error
}

func (ki *MockKeyImporter) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	if !reflect.DeepEqual(ki.RawArg, raw) {
		return nil, errors.New("invalid raw")
	}
	if !reflect.DeepEqual(ki.OptsArg, opts) {
		return nil, errors.New("invalid opts")
	}

	return ki.Value, ki.Err
}

type MockKeyImportOpts struct{}

func (*MockKeyImportOpts) Algorithm() string {
	return "Mock KeyImportOpts"
}

func (*MockKeyImportOpts) Ephemeral() bool {
	panic("Not yet implemented")
}

func TestKeyImport(t *testing.T) {
	expectedRaw := []byte{1, 2, 3}
	expectedOpts := &MockKeyDerivOpts{EphemeralValue: true}
	expectetValue := &MockKey{BytesValue: []byte{1, 2, 3, 4, 5}}
	expectedErr := errors.New("Expected Error")

	keyImporters := make(map[reflect.Type]KeyImporter)
	keyImporters[reflect.TypeOf(&MockKeyDerivOpts{})] = &MockKeyImporter{
		RawArg:  expectedRaw,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     expectedErr,
	}
	csp := impl{keyImporters: keyImporters}
	value, err := csp.KeyImport(expectedRaw, expectedOpts)
	assert.Nil(t, value)
	assert.Contains(t, err.Error(), expectedErr.Error())

	keyImporters = make(map[reflect.Type]KeyImporter)
	keyImporters[reflect.TypeOf(&MockKeyDerivOpts{})] = &MockKeyImporter{
		RawArg:  expectedRaw,
		OptsArg: expectedOpts,
		Value:   expectetValue,
		Err:     nil,
	}
	csp = impl{keyImporters: keyImporters}
	value, err = csp.KeyImport(expectedRaw, expectedOpts)
	assert.Equal(t, expectetValue, value)
	assert.Nil(t, err)
}

func TestAES256ImportKeyOptsKeyImporter(t *testing.T) {
	ki := aes256ImportKeyOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport([]byte(nil), &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. It must not be nil.")

	_, err = ki.KeyImport([]byte{0}, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid Key Length [")
}

func TestHMACImportKeyOptsKeyImporter(t *testing.T) {
	ki := hmacImportKeyOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport([]byte(nil), &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. It must not be nil.")
}

func TestECDSAPKIXPublicKeyImportOptsKeyImporter(t *testing.T) {
	ki := ecdsaPKIXPublicKeyImportOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport([]byte(nil), &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw. It must not be nil.")

	_, err = ki.KeyImport([]byte{0}, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed converting PKIX to ECDSA public key [")

	k, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	raw, err := utils.PublicKeyToDER(&k.PublicKey)
	assert.NoError(t, err)
	_, err = ki.KeyImport(raw, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed casting to ECDSA public key. Invalid raw material.")
}

func TestECDSAPrivateKeyImportOptsKeyImporter(t *testing.T) {
	ki := ecdsaPrivateKeyImportOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected byte array.")

	_, err = ki.KeyImport([]byte(nil), &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw. It must not be nil.")

	_, err = ki.KeyImport([]byte{0}, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed converting PKIX to ECDSA public key")

	k, err := rsa.GenerateKey(rand.Reader, 512)
	assert.NoError(t, err)
	raw := gm.MarshalPKCS1PrivateKey(k)
	_, err = ki.KeyImport(raw, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed casting to ECDSA private key. Invalid raw material.")
}

func TestECDSAGoPublicKeyImportOptsKeyImporter(t *testing.T) {
	ki := ecdsaGoPublicKeyImportOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *ecdsa.PublicKey.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *ecdsa.PublicKey.")
}

func TestRSAGoPublicKeyImportOptsKeyImporter(t *testing.T) {
	ki := rsaGoPublicKeyImportOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *rsa.PublicKey.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *rsa.PublicKey.")
}

func TestX509PublicKeyImportOptsKeyImporter(t *testing.T) {
	ki := x509PublicKeyImportOptsKeyImporter{}

	_, err := ki.KeyImport("Hello World", &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *gm.Certificate.")

	_, err = ki.KeyImport(nil, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid raw material. Expected *gm.Certificate.")

	cert := &gm.Certificate{}
	cert.PublicKey = "Hello world"
	_, err = ki.KeyImport(cert, &MockKeyImportOpts{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Certificate's public key type not recognized. Supported keys: [ECDSA, RSA]")
}
