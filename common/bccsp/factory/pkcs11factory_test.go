package factory

import (
	"testing"

	"github.com/rongzer/blockchain/common/bccsp/pkcs11"
	"github.com/stretchr/testify/assert"
)

func TestPKCS11FactoryName(t *testing.T) {
	f := &PKCS11Factory{}
	assert.Equal(t, f.Name(), PKCS11BasedFactoryName)
}

func TestPKCS11FactoryGetInvalidArgs(t *testing.T) {
	f := &PKCS11Factory{}

	_, err := f.Get(nil)
	assert.Error(t, err, "Invalid config. It must not be nil.")

	_, err = f.Get(&FactoryOpts{})
	assert.Error(t, err, "Invalid config. It must not be nil.")

	opts := &FactoryOpts{
		Pkcs11Opts: &pkcs11.PKCS11Opts{},
	}
	_, err = f.Get(opts)
	assert.Error(t, err, "CSP:500 - Failed initializing configuration at [0,]")
}
