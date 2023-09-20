package main

import (
	"fmt"
	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/localmsp"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/metadata"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/consensus"
	"github.com/rongzer/blockchain/peer/consensus/raft"
	"github.com/rongzer/blockchain/peer/endorserclient"
	"github.com/rongzer/blockchain/peer/ledger/kvledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/peer/server"
	sealer "github.com/rongzer/blockchain/peer/server/sealer"
	"github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"github.com/rongzer/blockchain/protos/utils"
	"google.golang.org/grpc"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
)

// 初始化MSP
func initializeMsp(module string) error {
	var mspId string
	var mspDir string
	if module == "peer" {
		mspId = conf.V.Peer.MSPID
		mspDir = conf.V.Peer.MSPDir
	}
	if module == "sealer" {
		mspId = conf.V.Sealer.MSPID
		mspDir = conf.V.Sealer.MSPDir
	}
	log.Logger.Infof("initialize local msp %s at %s", mspId, mspDir)
	// Check whenever msp folder exists
	if _, err := os.Stat(mspDir); os.IsNotExist(err) {
		// No need to try to load MSP from folder which is not available
		return fmt.Errorf("cannot init crypto, missing %s folder", mspDir)
	}
	// Init the BCCSP
	if err := mspmgmt.LoadLocalMsp(mspDir, conf.V.BCCSP, mspId); err != nil {
		return fmt.Errorf("error when setting up MSP from directory %s: err %s", mspDir, err)
	}
	return nil
}

//func initializeLicences() error {
//	rsaDir := conf.V.MSPDir
//	log.Logger.Infof("initialize licences %s ", rsaDir)
//	if _, err := os.Stat(rsaDir); os.IsNotExist(err) {
//		// No need to try to load MSP from folder which is not available
//		return fmt.Errorf("cannot init crypto, missing %s folder", rsaDir)
//	}
//
//	li := licences.NewLicences(rsaDir)
//	err := li.CheckLicences()
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

func main() {
	defer log.SyncLog()

	// 读取配置
	if err := conf.Initialize(); err != nil {
		panic(err)
	}
	// 初始化日志
	if err := log.InitLogger(metadata.Version, conf.V.LogPath, "peer", conf.V.LogLevel); err != nil {
		panic(err)
	}

	// 打印当前版本信息
	metadata.PrintVersionInfo()

	// 判断licence注册码
	//if err := initializeLicences(); err != nil {
	//	log.Logger.Errorf("Cannot run active licences because %s", err)
	//	os.Exit(1)
	//}

	// 创建pprof服务
	if conf.V.Profile.Enabled {
		go func() {
			log.Logger.Infof("Starting profiling server with listenAddress = %s", conf.V.Profile.Address)
			if err := http.ListenAndServe(conf.V.Profile.Address, nil); err != nil {
				log.Logger.Errorf("Error starting profiler: %s", err)
			}
		}()
	}

	peerRun()

}

func peerRun() {
	// 启动落块模块相关服务
	if err := initializeMsp("sealer"); err != nil {
		log.Logger.Errorf("Cannot run this node because %s", err)
		os.Exit(1)
	}
	// 创建签名对象
	signer := localmsp.NewSigner()
	// 初始化落块模块的GRPC服务
	grpcServer, err := initializeSealerGrpcServer()
	if err != nil {
		panic(err)
	}
	// 初始化落块模块的链管理器
	chainManager, err := initializeChainManager(signer, grpcServer.Server())
	if err != nil {
		panic(err)
	}

	// 初始化落块模块的连接管理器
	clientManager, err := endorserclient.NewManager()
	if err != nil {
		panic(err)
	}

	// 落块模块绑定接口实现
	ab.RegisterAtomicBroadcastServer(grpcServer.Server(), sealer.NewAtomicBroadcastServer(chainManager, signer, clientManager))
	// 启动落块模块的GRPC服务
	go func() {
		if err := grpcServer.Start(); err != nil {
			panic(err)
		}
	}()
	// 启动交互模块相关服务
	if err := initializeMsp("peer"); err != nil {
		log.Logger.Errorf("Cannot run this node because %s", err)
		os.Exit(1)
	}
	if err := server.Serve(); err != nil {
		log.Logger.Errorf("Peer server stop because %s", err)
		os.Exit(1)
	}
}

// 初始化GRPC服务
func initializeSealerGrpcServer() (comm.GRPCServer, error) {
	// 初始化网络安全配置
	secureConfig, err := initializeSecureServerConfig()
	if err != nil {
		return nil, fmt.Errorf("Initialize secure config error. %w", err)
	}

	lis, err := net.Listen("tcp", conf.V.Sealer.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen tcp. %w", err)
	}

	// 创建GRPC服务
	return comm.NewGRPCServerFromListener(lis, secureConfig)
}

// 初始化网络安全配置
func initializeSecureServerConfig() (comm.SecureServerConfig, error) {
	secureConfig := comm.SecureServerConfig{
		UseTLS:            conf.V.TLS.Enabled,
		RequireClientCert: conf.V.TLS.ClientAuthEnabled,
	}

	if secureConfig.UseTLS {
		log.Logger.Info("Starting sealer with TLS enabled")
		// 读取证书
		serverCertificate, err := ioutil.ReadFile(conf.V.TLS.Certificate)
		if err != nil {
			return secureConfig, fmt.Errorf("failed to load ServerCertificate file '%s'. %w", conf.V.TLS.Certificate, err)
		}
		secureConfig.ServerCertificate = serverCertificate
		// 读取私钥
		serverKey, err := ioutil.ReadFile(conf.V.TLS.PrivateKey)
		if err != nil {
			return secureConfig, fmt.Errorf("failed to load PrivateKey file '%s' %w", conf.V.TLS.PrivateKey, err)
		}
		secureConfig.ServerKey = serverKey
		// 读取根证书
		var serverRootCAs [][]byte
		for _, serverRoot := range conf.V.TLS.RootCAs {
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
			for _, clientRoot := range conf.V.TLS.ClientRootCAs {
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
func initializeChainManager(signer crypto.LocalSigner, srv *grpc.Server) (*chain.Manager, error) {
	ledgermgmt.Initialize()
	// 账本中找不到任何链数据时需要初始化创建链账本
	ledgerIds, err := ledgermgmt.GetLedgerIDs()
	if err != nil || len(ledgerIds) == 0 {
		log.Logger.Infof("Need initialize bootstrap chain because not found any chain's ledger folder")
		if err := initializeChainLedger(); err != nil {
			log.Logger.Errorf("Initialize bootstrap chain error. %v", err)
			return nil, err
		}
	} else {
		log.Logger.Info("Not bootstrapping because of existing chains")
	}

	// 注册共识模式
	modes := make(map[string]consensus.Mode)
	m := &chain.Manager{}
	// 在这里增加raft共识的注册
	cc := comm.ClientConfig{
		AsyncConnect: true,
		KaOpts:       comm.DefaultKeepaliveOptions(),
		Timeout:      conf.V.Sealer.Raft.DialTimeout,
	}
	clusterDialer := &cluster.PredicateDialer{
		Config: cc,
	}
	cryptoProvider := factory.GetDefault()
	modes["raft"] = raft.Init(clusterDialer, srv, m, cryptoProvider)
	// 初始化链管理器
	chainManager, err := chain.NewManager(modes, signer, m)
	if err != nil {
		log.Logger.Errorf("multichain manager initialize error. %v", err)
		return nil, err
	}

	return chainManager, nil
}

// 初始化创建链账本
func initializeChainLedger() error {
	// 读取并解析创世块文件
	bootstrapFile, err := ioutil.ReadFile(conf.V.Sealer.GenesisFile)
	if err != nil {
		return fmt.Errorf("Error reading genesis block file. %w", err)
	}
	genesisBlock := &common.Block{}
	if err := genesisBlock.Unmarshal(bootstrapFile); err != nil {
		return fmt.Errorf("Error unmarshalling genesis block. %w", err)
	}
	// raft共识模式下，raft集群中第一个节点创建系统链的创世块，需要往元数据里写入初始的raft集群节点信息：
	// 即写入自己的endpoint；若是"后加入节点"，则无需写入，因为后加入节点的genesisBlock不在这里append进账本
	if conf.V.Sealer.Raft.BootStrapEndPoint == "" {
		raftMetaData := &etcdraft.BlockMetadata{
			ConsenterEndpoints: []string{conf.V.Sealer.Raft.EndPoint},
			NextConsenterId:    uint64(2),
			PeerEndpoints:      []string{conf.V.Peer.Endpoint},
		}
		encodedMetadataValue := utils.MarshalOrPanic(raftMetaData)
		var metadataContents [][]byte
		for i := 0; i < len(common.BlockMetadataIndex_name); i++ {
			metadataContents = append(metadataContents, []byte{})
		}
		genesisBlock.Metadata = &common.BlockMetadata{Metadata: metadataContents}
		genesisBlock.Metadata.Metadata[common.BlockMetadataIndex_ORDERER] = utils.MarshalOrPanic(&common.Metadata{Value: encodedMetadataValue})
	}
	// 从块中的第一笔交易中获取链ID
	chainID, err := utils.GetChainIDFromBlock(genesisBlock)
	if err != nil {
		return fmt.Errorf("Failed to parse chain ID from genesis block. %w", err)
	}
	// 创建链账本
	l, err := ledgermgmt.LedgerProvider.CreateWithoutCommit(genesisBlock)
	if err != nil {
		return err
	}
	kvLedger, ok := l.(*kvledger.KvLedger)
	if !ok {
		return fmt.Errorf("PeeLedger cannot convert to KvLedger")
	}
	signaledLedger := &kvledger.SignaledLedger{KvLedger: kvLedger, Signal: make(chan struct{})}
	ledgermgmt.SignaledLedgers.Store(chainID, signaledLedger)
	// raft共识模式下，raft集群的后加入节点的系统链的创世块不在这里写入账本
	if conf.V.Sealer.Raft.BootStrapEndPoint != "" {
		log.Logger.Info("this peer is not the first added node in raft cluster")
		return nil
	}
	// 写入创世块数据
	if err = signaledLedger.Append(genesisBlock); err != nil {
		return fmt.Errorf("Could not write genesis block to ledger. %w", err)
	}

	return nil
}
