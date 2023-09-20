package localconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
)

func TestLoadGoodConfig(t *testing.T) {
	envVar1 := "BLOCKCHAIN_CFG_PATH"
	envVal1 := "../../sampleconfig/"
	os.Setenv(envVar1, envVal1)
	defer os.Unsetenv(envVar1)
	cfg, err := Load()
	assert.NotNil(t, cfg, "Could not load config")
	assert.Nil(t, err, "Load good config returned unexpected error")
}

func TestLoadMissingConfigFile(t *testing.T) {
	envVar1 := "BLOCKCHAIN_CFG_PATH"
	envVal1 := "invalid orderer cfg path"
	os.Setenv(envVar1, envVal1)
	defer os.Unsetenv(envVar1)

	cfg, err := Load()
	assert.Nil(t, cfg, "Loaded missing config file")
	assert.NotNil(t, err, "Loaded missing config file without error")
}

func TestLoadMalformedConfigFile(t *testing.T) {
	viper.Reset()
	name, err := ioutil.TempDir("", "rbc")
	assert.Nil(t, err, "Error creating temp dir: %s", err)
	defer func() {
		err = os.RemoveAll(name)
		assert.Nil(t, os.RemoveAll(name), "Error removing temp dir: %s", err)
	}()

	{
		// Create a malformed orderer.yaml file in temp dir
		f, err := os.OpenFile(filepath.Join(name, "orderer.yaml"), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		assert.Nil(t, err, "Error creating file: %s", err)
		f.WriteString("General: 42")
		assert.NoError(t, f.Close(), "Error closing file")
	}

	envVar1 := "BLOCKCHAIN_CFG_PATH"
	envVal1 := name
	os.Setenv(envVar1, envVal1)
	defer os.Unsetenv(envVar1)

	cfg, err := Load()
	assert.Nil(t, cfg, "Loaded missing config file")
	assert.NotNil(t, err, "Loaded missing config file without error")
}

// TestEnvInnerVar verifies that with the Unmarshal function that
// the environmental overrides still work on internal vars.  This was
// a bug in the original viper implementation that is worked around in
// the Load codepath for now
func TestEnvInnerVar(t *testing.T) {
	envVar := "BLOCKCHAIN_CFG_PATH"
	envVal := "../../sampleconfig/"
	os.Setenv(envVar, envVal)
	defer os.Unsetenv(envVar)

	envVar1 := "ORDERER_GENERAL_LISTENPORT"
	envVal1 := uint16(80)
	envVar2 := "ORDERER_KAFKA_RETRY_SHORTINTERVAL"
	envVal2 := "42s"
	os.Setenv(envVar1, fmt.Sprintf("%d", envVal1))
	os.Setenv(envVar2, envVal2)
	defer os.Unsetenv(envVar1)
	defer os.Unsetenv(envVar2)
	config, _ := Load()

	assert.NotNil(t, config, "Could not load config")
	assert.Equal(t, config.General.ListenPort, envVal1, "Environmental override of inner config test 1 did not work")

	v2, _ := time.ParseDuration(envVal2)
	assert.Equal(t, config.Kafka.Retry.ShortInterval, v2, "Environmental override of inner config test 2 did not work")
}
