package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/metadata"
	"github.com/rongzer/blockchain/light/server"
	"github.com/rongzer/blockchain/peer/config"
	"github.com/spf13/viper"
)

// Constants go here.
const cmdRoot = "core"

// initializeConfig 初始化配置
func initializeConfig() error {
	viper.SetEnvPrefix(cmdRoot)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	return server.InitConfig(cmdRoot)
}

// initializeMsp 初始化MSP
func initializeMsp() error {
	var mspMgrConfigDir = config.GetPath("peer.mspConfigPath")
	var mspID = viper.GetString("peer.localMspId")
	return server.InitCrypto(mspMgrConfigDir, mspID)
}

func main() {
	defer log.SyncLog()

	var chaincodeDevMode bool
	flag.BoolVar(&chaincodeDevMode, "peer-chaincodedev", false, "")
	flag.Parse()

	// 初始化配置
	if err := initializeConfig(); err != nil {
		log.Logger.Errorf("Fatal error when initializing %s config : %s\n", cmdRoot, err)
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
	if server.Serve(chaincodeDevMode) != nil {
		os.Exit(1)
	}
	log.Logger.Info("Exiting.....")
}
