package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/ledger/blkstorage/fsblkstorage"
	"github.com/rongzer/blockchain/common/localmsp"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/metadata"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/orderer/chain"
	"github.com/rongzer/blockchain/orderer/clients"
	"github.com/rongzer/blockchain/orderer/consensus"
	"github.com/rongzer/blockchain/orderer/consensus/kafka"
	"github.com/rongzer/blockchain/orderer/consensus/solo"
	"github.com/rongzer/blockchain/orderer/ledger"
	"github.com/rongzer/blockchain/orderer/localconfig"
	"github.com/rongzer/blockchain/orderer/server"
	"github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/utils"
)

func main() {
	defer log.SyncLog()

	// 读取配置
	conf, err := localconfig.Load()
	if err != nil {
		panic(err)
	}

	// 初始化日志
	if err := log.InitLogger(metadata.Version, "/var/rongzer/blockchain/logs/", "orderer", conf.General.LogLevel); err != nil {
		panic(err)
	}

	// 打印当前版本信息
	metadata.PrintVersionInfo()

	// 初始化pprof服务
	initializeProfilingService(conf)

	// 载入本地MSP
	if err := mgmt.LoadLocalMsp(conf.General.LocalMSPDir, conf.General.BCCSP, conf.General.LocalMSPID); err != nil {
		log.Logger.Panicw("Failed to initialize local MSP",
			"LocalMSPDir", conf.General.LocalMSPDir,
			"LocalMSPID", conf.General.LocalMSPID,
			"BCCSP", conf.General.BCCSP,
			"Error", err,
		)
	}

	// 创建签名对象
	signer := localmsp.NewSigner()

	// 初始化链管理器
	chainManager, err := initializeChainManager(conf, signer)
	if err != nil {
		panic(err)
	}

	// 初始化连接管理器
	clientManager, err := clients.NewManager()
	if err != nil {
		panic(err)
	}

	// 初始化GRPC服务
	grpcServer, err := initializeGrpcServer(conf)
	if err != nil {
		panic(err)
	}
	// 绑定接口实现
	ab.RegisterAtomicBroadcastServer(grpcServer.Server(), server.NewAtomicBroadcastServer(chainManager, signer, clientManager))
	// 启动GRPC服务
	grpcServer.Start()
}

// 初始化pprof服务
func initializeProfilingService(conf *localconfig.TopLevel) {
	if conf.General.Profile.Enabled {
		go func() {
			log.Logger.Info("Starting Go pprof profiling service on:", conf.General.Profile.Address)
			log.Logger.Panic("Go pprof service failed:", http.ListenAndServe(conf.General.Profile.Address, nil))
		}()
	}
}

// 初始化GRPC服务
func initializeGrpcServer(conf *localconfig.TopLevel) (comm.GRPCServer, error) {
	// 初始化网络安全配置
	secureConfig, err := initializeSecureServerConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("Initialize secure config error. %w", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", conf.General.ListenAddress, conf.General.ListenPort))
	if err != nil {
		return nil, fmt.Errorf("Failed to listen tcp. %w", err)
	}

	// 创建GRPC服务
	return comm.NewGRPCServerFromListener(lis, secureConfig)
}

// 初始化网络安全配置
func initializeSecureServerConfig(conf *localconfig.TopLevel) (comm.SecureServerConfig, error) {
	secureConfig := comm.SecureServerConfig{
		UseTLS:            conf.General.TLS.Enabled,
		RequireClientCert: conf.General.TLS.ClientAuthEnabled,
	}

	if secureConfig.UseTLS {
		log.Logger.Info("Starting orderer with TLS enabled")
		// 读取证书
		serverCertificate, err := ioutil.ReadFile(conf.General.TLS.Certificate)
		if err != nil {
			return secureConfig, fmt.Errorf("Failed to load ServerCertificate file '%s'. %w", conf.General.TLS.Certificate, err)
		}
		secureConfig.ServerCertificate = serverCertificate
		// 读取私钥
		serverKey, err := ioutil.ReadFile(conf.General.TLS.PrivateKey)
		if err != nil {
			return secureConfig, fmt.Errorf("Failed to load PrivateKey file '%s' %w", conf.General.TLS.PrivateKey, err)
		}
		secureConfig.ServerKey = serverKey
		// 读取根证书
		var serverRootCAs [][]byte
		for _, serverRoot := range conf.General.TLS.RootCAs {
			root, err := ioutil.ReadFile(serverRoot)
			if err != nil {
				return secureConfig, fmt.Errorf("Failed to load ServerRootCAs file '%s' %w", serverRoot, err)
			}
			serverRootCAs = append(serverRootCAs, root)
		}
		secureConfig.ServerRootCAs = serverRootCAs
		// 读取客户端根证书
		var clientRootCAs [][]byte
		if secureConfig.RequireClientCert {
			for _, clientRoot := range conf.General.TLS.ClientRootCAs {
				root, err := ioutil.ReadFile(clientRoot)
				if err != nil {
					return secureConfig, fmt.Errorf("Failed to load ClientRootCAs file '%s' %w", clientRoot, err)
				}
				clientRootCAs = append(clientRootCAs, root)
			}
		}
		secureConfig.ClientRootCAs = clientRootCAs
	}

	return secureConfig, nil
}

// 初始化链管理器
func initializeChainManager(conf *localconfig.TopLevel, signer crypto.LocalSigner) (*chain.Manager, error) {
	// 初始化账本管理器
	location := conf.FileLedger.Location
	if location == "" {
		// 没有配置账本储存路径时, 创建并使用临时目录
		var err error
		location, err = ioutil.TempDir("", conf.FileLedger.Prefix)
		if err != nil {
			return nil, err
		}
	}
	ledgerManager := ledger.NewManager(location)
	// 账本中找不到任何链数据时需要初始化创建链账本
	chains, err := ledgerManager.ChainIDs()
	if err != nil || len(chains) == 0 {
		log.Logger.Infof("Need initialize bootstrap chain because not found any chain's ledger folder")

		// 创建子目录
		chainsDir := filepath.Join(location, fsblkstorage.ChainsDir)
		if _, err := os.Stat(chainsDir); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(chainsDir, 0755); err != nil {
					return nil, fmt.Errorf("Error creating chains dir %s. %w", chainsDir, err)
				}
			}
		}

		if err := initializeChainLedger(conf, ledgerManager); err != nil {
			log.Logger.Errorf("Initialize bootstrap chain error. %v", err)
			return nil, err
		}
	} else {
		log.Logger.Info("Not bootstrapping because of existing chains")
	}

	// 注册共识模式
	modes := make(map[string]consensus.Mode)
	modes["solo"] = solo.Init()
	modes["kafka"] = kafka.Init(conf.Kafka.TLS, conf.Kafka.Retry, conf.Kafka.Version)

	// 初始化链管理器
	chainManager, err := chain.NewManager(ledgerManager, modes, signer)
	if err != nil {
		log.Logger.Errorf("multichain manager initialize error. %v", err)
		return nil, err
	}

	return chainManager, nil
}

// 初始化创建链账本
func initializeChainLedger(conf *localconfig.TopLevel, ledgerManager *ledger.Manager) error {
	// 读取并解析创世块文件
	bootstrapFile, err := ioutil.ReadFile(conf.General.GenesisFile)
	if err != nil {
		return fmt.Errorf("Error reading genesis block file. %w", err)
	}
	genesisBlock := &common.Block{}
	if err := genesisBlock.Unmarshal(bootstrapFile); err != nil {
		return fmt.Errorf("Error unmarshalling genesis block. %w", err)
	}

	// 从块中的第一笔交易中获取链ID
	chainID, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return fmt.Errorf("Failed to parse chain ID from genesis block. %w", err)
	}
	// 创建链账本
	l, err := ledgerManager.GetOrCreate(chainID)
	if err != nil {
		return fmt.Errorf("Failed to create the system chain. %w", err)
	}
	// 写入创世块数据
	if err = l.Append(genesisBlock); err != nil {
		return fmt.Errorf("Could not write genesis block to ledger. %w", err)
	}

	return nil
}
