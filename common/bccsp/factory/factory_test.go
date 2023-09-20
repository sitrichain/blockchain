package factory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	var jsonBCCSP, yamlBCCSP *FactoryOpts
	jsonCFG := []byte(
		`{ "default": "SW", "SW":{ "security": 384, "hash": "SHA3" } }`)

	err := json.Unmarshal(jsonCFG, &jsonBCCSP)
	if err != nil {
		fmt.Printf("Could not parse JSON config [%s]", err)
		os.Exit(-1)
	}

	yamlCFG := `
BCCSP:
    default: SW
    SW:
        Hash: SHA3
        Security: 256`

	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer([]byte(yamlCFG)))
	if err != nil {
		fmt.Printf("Could not read YAML config [%s]", err)
		os.Exit(-1)
	}

	err = viper.UnmarshalKey("bccsp", &yamlBCCSP)
	if err != nil {
		fmt.Printf("Could not parse YAML config [%s]", err)
		os.Exit(-1)
	}

	cfgVariations := []*FactoryOpts{
		{
			ProviderName: "SW",
			SwOpts: &SwOpts{
				HashFamily: "SHA2",
				SecLevel:   256,

				Ephemeral: true,
			},
		},
		{},
		{
			ProviderName: "SW",
		},
		jsonBCCSP,
		yamlBCCSP,
	}

	for index, config := range cfgVariations {
		fmt.Printf("Trying configuration [%d]\n", index)
		InitFactories(config)
		InitFactories(nil)
		m.Run()
	}
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
