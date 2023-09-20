package factory

import (
	"testing"

	"github.com/rongzer/blockchain/common/bccsp/pkcs11"
	"github.com/stretchr/testify/assert"
)

func TestInitFactories(t *testing.T) {
	// Reset errors from previous negative test runs
	factoriesInitError = nil

	err := InitFactories(nil)
	assert.NoError(t, err)
}

func TestSetFactories(t *testing.T) {
	err := setFactories(nil)
	assert.NoError(t, err)

	err = setFactories(&FactoryOpts{})
	assert.NoError(t, err)
}

func TestSetFactoriesInvalidArgs(t *testing.T) {
	err := setFactories(&FactoryOpts{
		ProviderName: "SW",
		SwOpts:       &SwOpts{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed initializing SW.BCCSP")

	err = setFactories(&FactoryOpts{
		ProviderName: "PKCS11",
		Pkcs11Opts:   &pkcs11.PKCS11Opts{},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed initializing PKCS11.BCCSP")
}
