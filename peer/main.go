package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/metadata"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/viperutil"
	"github.com/rongzer/blockchain/peer/config"
	"github.com/rongzer/blockchain/peer/server"
	"github.com/spf13/viper"
)

// 初始化配置
func initializeConfig() error {
	viper.SetEnvPrefix("core")
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	if err := config.InitViper(nil, "core"); err != nil {
		return err
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("Error when reading config file: %s", err)
	}
	return nil
}

// 初始化MSP
func initializeMsp() error {
	mspMgrConfigDir := config.GetPath("peer.mspConfigPath")
	mspID := viper.GetString("peer.localMspId")
	log.Logger.Infof("initialize local msp %s at %s", mspID, mspMgrConfigDir)
	// Check whenever msp folder exists
	if _, err := os.Stat(mspMgrConfigDir); os.IsNotExist(err) {
		// No need to try to load MSP from folder which is not available
		return fmt.Errorf("cannot init crypto, missing %s folder", mspMgrConfigDir)
	}
	// Init the BCCSP
	var bccspConfig *factory.FactoryOpts
	if err := viperutil.EnhancedExactUnmarshalKey("peer.BCCSP", &bccspConfig); err != nil {
		return fmt.Errorf("could not parse YAML config [%s]", err)
	}
	if err := mspmgmt.LoadLocalMsp(mspMgrConfigDir, bccspConfig, mspID); err != nil {
		return fmt.Errorf("error when setting up MSP from directory %s: err %s", mspMgrConfigDir, err)
	}
	return nil
}

func main() {
	defer log.SyncLog()

	// 初始化配置
	if err := initializeConfig(); err != nil {
		log.Logger.Errorf("Fatal error when initializing config : %s", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := log.InitLogger(metadata.Version, "/var/rongzer/production/logs/", "peer", viper.GetString("logging.level")); err != nil {
		panic(err)
	}

	// 打印当前版本信息
	metadata.PrintVersionInfo()

	// 创建pprof服务
	if viper.GetBool("peer.profile.enabled") {
		go func() {
			profileListenAddress := viper.GetString("peer.profile.listenAddress")
			log.Logger.Infof("Starting profiling server with listenAddress = %s", profileListenAddress)
			if profileErr := http.ListenAndServe(profileListenAddress, nil); profileErr != nil {
				log.Logger.Errorf("Error starting profiler: %s", profileErr)
			}
		}()
	}

	// 初始化MSP
	if err := initializeMsp(); err != nil {
		log.Logger.Errorf("Cannot run peer because %s", err)
		os.Exit(1)
	}

	// 运行服务
	var chaincodeDevMode bool
	flag.BoolVar(&chaincodeDevMode, "peer-chaincodedev", false, "")
	flag.Parse()
	if err := server.Serve(chaincodeDevMode); err != nil {
		log.Logger.Errorf("Peer server stop because %s", err)
		os.Exit(1)
	}
}
