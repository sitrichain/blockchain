package factory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSWFactoryName(t *testing.T) {
	f := &SWFactory{}
	assert.Equal(t, f.Name(), SoftwareBasedFactoryName)
}

func TestSWFactoryGetInvalidArgs(t *testing.T) {
	f := &SWFactory{}

	_, err := f.Get(nil)
	assert.Error(t, err, "Invalid config. It must not be nil.")

	_, err = f.Get(&FactoryOpts{})
	assert.Error(t, err, "Invalid config. It must not be nil.")

	opts := &FactoryOpts{
		SwOpts: &SwOpts{},
	}
	_, err = f.Get(opts)
	assert.Error(t, err, "CSP:500 - Failed initializing configuration at [0,]")
}

func TestSWFactoryGet(t *testing.T) {
	f := &SWFactory{}

	opts := &FactoryOpts{
		SwOpts: &SwOpts{
			SecLevel:   256,
			HashFamily: "SHA2",
		},
	}
	csp, err := f.Get(opts)
	assert.NoError(t, err)
	assert.NotNil(t, csp)

	opts = &FactoryOpts{
		SwOpts: &SwOpts{
			SecLevel:     256,
			HashFamily:   "SHA2",
			FileKeystore: &FileKeystoreOpts{KeyStorePath: os.TempDir()},
		},
	}
	csp, err = f.Get(opts)
	assert.NoError(t, err)
	assert.NotNil(t, csp)

}
