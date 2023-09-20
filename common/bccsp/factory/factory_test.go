package factory

import (
	"fmt"
	"os"
	"testing"

	"github.com/rongzer/blockchain/common/conf"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	fmt.Print("Trying configuration\n")
	InitFactories(conf.V.BCCSP)
	InitFactories(nil)
	m.Run()
	os.Exit(0)
}

func TestGetDefault(t *testing.T) {
	bccsp := GetDefault()
	if bccsp == nil {
		t.Fatal("Failed getting default BCCSP. Nil instance.")
	}
}

func TestGetBCCSP(t *testing.T) {
	bccsp, err := GetBCCSP("SW")
	assert.NoError(t, err)
	assert.NotNil(t, bccsp)

	bccsp, err = GetBCCSP("BadName")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Could not find BCCSP, no 'BadName' provider")
	assert.Nil(t, bccsp)
}
