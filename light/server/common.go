package server

import (
	"fmt"
	"os"

	"github.com/rongzer/blockchain/common/bccsp/factory"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/viperutil"
	"github.com/rongzer/blockchain/peer/config"
	"github.com/spf13/viper"
)

//InitConfig initializes viper config
func InitConfig(cmdRoot string) error {
	if err := config.InitViper(nil, cmdRoot); err != nil {
		return err
	}
	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("Error when reading %s config file: %s", cmdRoot, err)
	}

	return nil
}

//InitCrypto initializes crypto for this peer
func InitCrypto(mspMgrConfigDir string, localMSPID string) error {
	var err error
	// Check whenever msp folder exists
	_, err = os.Stat(mspMgrConfigDir)
	if os.IsNotExist(err) {
		// No need to try to load MSP from folder which is not available
		return fmt.Errorf("cannot init crypto, missing %s folder", mspMgrConfigDir)
	}

	// Init the BCCSP
	var bccspConfig *factory.FactoryOpts
	err = viperutil.EnhancedExactUnmarshalKey("peer.BCCSP", &bccspConfig)
	if err != nil {
		return fmt.Errorf("could not parse YAML config [%s]", err)
	}

	err = mspmgmt.LoadLocalMsp(mspMgrConfigDir, bccspConfig, localMSPID)
	if err != nil {
		return fmt.Errorf("error when setting up MSP from directory %s: err %s", mspMgrConfigDir, err)
	}

	return nil
}
