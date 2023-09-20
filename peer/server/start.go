package server

import (
	"fmt"
	"github.com/rongzer/blockchain/peer/ledger/kvledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/peer/scc"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/chaincode"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/endorser"
	"github.com/rongzer/blockchain/peer/events/producer"
	pb "github.com/rongzer/blockchain/protos/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

//function used by chaincode support
type ccEndpointFunc func() (*pb.PeerEndpoint, error)

// GetSecureConfig returns the secure server configuration for the peer
func GetSecureConfig() (comm.SecureServerConfig, error) {
	secureConfig := comm.SecureServerConfig{
		UseTLS: conf.V.TLS.Enabled,
	}
	if secureConfig.UseTLS {
		// get the certs from the file system
		serverKey, err := ioutil.ReadFile(conf.V.TLS.PrivateKey)
		serverCert, err := ioutil.ReadFile(conf.V.TLS.Certificate)
		// must have both key and cert file
		if err != nil {
			return secureConfig, fmt.Errorf("Error loading TLS key and/or certificate (%s)", err)
		}
		secureConfig.ServerCertificate = serverCert
		secureConfig.ServerKey = serverKey
		// check for root cert
		if len(conf.V.TLS.RootCAs) > 0 {
			rootCert, err := ioutil.ReadFile(conf.V.TLS.RootCAs[0])
			if err != nil {
				return secureConfig, fmt.Errorf("Error loading TLS root certificate (%s)", err)
			}
			secureConfig.ServerRootCAs = [][]byte{rootCert}
		}
		return secureConfig, nil
	}
	return secureConfig, nil
}

func Serve() error {
	// Parameter overrides must be processed before any parameters are
	// cached. Failures to cache cause the server to terminate immediately.
	if conf.V.Peer.Chaincode.Mode == chaincode.DevModeUserRunsChaincode {
		log.Logger.Info("Running in chaincode development mode")
		log.Logger.Info("Disable loading validity system chaincode")
	}

	peerEndpoint := &pb.PeerEndpoint{Id: &pb.PeerID{Name: conf.V.Peer.ID}, Address: conf.V.Peer.Endpoint}

	secureConfig, err := GetSecureConfig()
	if err != nil {
		log.Logger.Fatalf("Error loading secure config for peer (%s)", err)
		return err
	}
	// 创建交互模块的grpc service
	peerServer, err := comm.NewGRPCServer(conf.V.Peer.ListenAddress, secureConfig)
	if err != nil {
		log.Logger.Fatalf("Failed to create peer server (%s)", err)
		return err
	}

	log.Logger.Infof("peerServer listenAddr : %s", conf.V.Peer.ListenAddress)

	if secureConfig.UseTLS {
		log.Logger.Info("Starting peer with TLS enabled")
		// set up CA support
		caSupport := comm.GetCASupport()
		caSupport.ServerRootCAs = secureConfig.ServerRootCAs
	}

	// 创建交互模块的eventhub service
	ehubGrpcServer, err := createEventHubServer(secureConfig)
	if err != nil {
		grpclog.Fatalf("Failed to create ehub server: %v", err)
		return err
	}

	log.Logger.Infof("eventhubServer listenAddr : %s", conf.V.Peer.Events.Address)

	// enable the cache of chaincode info
	ccprovider.EnableCCInfoCache()

	// 创建交互模块的chaincode service
	ccSrv, ccEpFunc := createChaincodeServer(peerServer, conf.V.Peer.ListenAddress)
	registerChaincodeSupport(ccSrv.Server(), ccEpFunc)

	// 创建交互模块的Endorser server
	serverEndorser := endorser.NewEndorserServer()
	pb.RegisterEndorserServer(peerServer.Server(), serverEndorser)

	go ccSrv.Start()

	// 初始化所有链的系统链码
	ledgerIds, err := ledgermgmt.GetLedgerIDs()
	if err != nil {
		log.Logger.Fatalf("Error in initializing ledgermgmt: %s", err)
		return err
	}
	chainInitializer := func(cid string) {
		log.Logger.Debugf("Deploying system CC, for chain <%s>", cid)
		scc.DeploySysCCs(cid)
	}
	chain.SetChainInitializer(chainInitializer)
	for _, cid := range ledgerIds {
		chain.InitChain(cid)
	}

	log.Logger.Infof("Starting peer with ID=[%s], network ID=[%s], address=[%s]",
		peerEndpoint.Id, conf.V.Peer.NetworkId, peerEndpoint.Address)

	// Start the grpc server. Done in a goroutine so we can deploy the
	// genesis block if needed.
	serve := make(chan error)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Logger.Debugf("sig: %s", sig)
		serve <- nil
	}()

	go func() {
		var grpcErr error
		if grpcErr = peerServer.Start(); grpcErr != nil {
			grpcErr = fmt.Errorf("grpc server exited with error: %s", grpcErr)
			log.Logger.Errorf("grpc server exited with error: %s", grpcErr)
		} else {
			log.Logger.Info("peer server exited")
		}
		serve <- grpcErr
	}()

	if err := writePid(conf.V.FileSystemPath+"/peer.pid", os.Getpid()); err != nil {
		return err
	}

	// Start the event hub server
	if ehubGrpcServer != nil {
		go ehubGrpcServer.Start()
	}

	log.Logger.Infof("Started peer with ID=[%s], network ID=[%s], address=[%s]",
		peerEndpoint.Id, conf.V.Peer.NetworkId, peerEndpoint.Address)

	// Block until grpc server exits
	return <-serve
}

//create a CC listener using peer.chaincodeListenAddress (and if that's not set use peer.peerAddress)
func createChaincodeServer(peerServer comm.GRPCServer, peerListenAddress string) (comm.GRPCServer, ccEndpointFunc) {
	var srv comm.GRPCServer
	var ccEpFunc ccEndpointFunc
	ccEpFunc = func() (*pb.PeerEndpoint, error) {
		return &pb.PeerEndpoint{Id: &pb.PeerID{Name: conf.V.Peer.ID}, Address: conf.V.Peer.Endpoint}, nil
	}
	srv = peerServer

	return srv, ccEpFunc
}

//NOTE - when we implment JOIN we will no longer pass the chainID as param
//The chaincode support will come up without registering system chaincodes
//which will be registered only during join phase.
func registerChaincodeSupport(grpcServer *grpc.Server, ccEpFunc ccEndpointFunc) {
	//get user mode
	userRunsCC := chaincode.IsDevMode()

	//get chaincode startup timeout
	ccStartupTimeout := conf.V.Peer.Chaincode.Startuptimeout
	if ccStartupTimeout < time.Duration(5)*time.Second {
		log.Logger.Warnf("Invalid chaincode startup timeout value %s (should be at least 5s); defaulting to 5s", ccStartupTimeout)
		ccStartupTimeout = time.Duration(5) * time.Second
	} else {
		log.Logger.Debugf("Chaincode startup timeout value set to %s", ccStartupTimeout)
	}

	ccSrv := chaincode.NewChaincodeSupport(ccEpFunc, userRunsCC, ccStartupTimeout)
	// 在peer/ledger/kvledger包中标识theChaincodeSupport已初始化
	kvledger.IsTheChaincodeSupportInitialized = true
	//Now that chaincode is initialized, register all system chaincodes.
	scc.RegisterSysCCs()

	pb.RegisterChaincodeSupportServer(grpcServer, ccSrv)
}

func createEventHubServer(secureConfig comm.SecureServerConfig) (comm.GRPCServer, error) {
	var lis net.Listener
	var err error
	lis, err = net.Listen("tcp", conf.V.Peer.Events.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer, err := comm.NewGRPCServerFromListener(lis, secureConfig)
	if err != nil {
		log.Logger.Errorf("Failed to return new GRPC server: %s", err)
		return nil, err
	}
	ehServer := producer.NewEventsServer(
		uint(conf.V.Peer.Events.BufferSize),
		conf.V.Peer.Events.Timeout)

	pb.RegisterEventsServer(grpcServer.Server(), ehServer)
	return grpcServer, nil
}

func writePid(fileName string, pid int) error {
	err := os.MkdirAll(filepath.Dir(fileName), 0755)
	if err != nil {
		return err
	}

	buf := strconv.Itoa(pid)
	if err = ioutil.WriteFile(fileName, []byte(buf), 0644); err != nil {
		log.Logger.Errorf("Cannot write pid to %s (err:%s)", fileName, err)
		return err
	}

	return nil
}
