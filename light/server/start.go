package server

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/localmsp"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	lightgossip "github.com/rongzer/blockchain/light/gossip"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/chaincode"
	"github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/config"
	"github.com/rongzer/blockchain/peer/endorser"
	"github.com/rongzer/blockchain/peer/events/producer"
	"github.com/rongzer/blockchain/peer/gossip/service"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/peer/scc"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

//function used by chaincode support
type ccEndpointFunc func() (*pb.PeerEndpoint, error)

//start chaincodes
func initSysCCs() {
	//deploy system chaincodes
	scc.DeploySysCCs("")
	log.Logger.Infof("Deployed system chaincodess")
}

func Serve(chaincodeDevMode bool) error {
	ledgermgmt.Initialize()
	// Parameter overrides must be processed before any parameters are
	// cached. Failures to cache cause the server to terminate immediately.
	if chaincodeDevMode {
		log.Logger.Info("Running in chaincode development mode")
		log.Logger.Info("Disable loading validity system chaincode")

		viper.Set("chaincode.mode", chaincode.DevModeUserRunsChaincode)

	}

	if err := chain.CacheConfiguration(); err != nil {
		return err
	}

	peerEndpoint, err := chain.GetPeerEndpoint()
	if err != nil {
		err = fmt.Errorf("Failed to get Peer Endpoint: %s", err)
		return err
	}

	listenAddr := viper.GetString("peer.listenAddress")

	secureConfig, err := chain.GetSecureConfig()
	if err != nil {
		log.Logger.Fatalf("Error loading secure config for peer (%s)", err)
	}
	peerServer, err := chain.CreatePeerServer(listenAddr, secureConfig)
	if err != nil {
		log.Logger.Fatalf("Failed to create peer server (%s)", err)
	}

	log.Logger.Infof("peerServer listenAddr : %s", listenAddr)

	if secureConfig.UseTLS {
		log.Logger.Info("Starting peer with TLS enabled")
		// set up CA support
		caSupport := comm.GetCASupport()
		caSupport.ServerRootCAs = secureConfig.ServerRootCAs
	}

	//TODO - do we need different SSL material for events ?
	ehubGrpcServer, err := createEventHubServer(secureConfig)
	if err != nil {
		grpclog.Fatalf("Failed to create ehub server: %v", err)
	}

	log.Logger.Infof("ehubGrpcServer listenAddr : %s", viper.GetString("peer.events.address"))

	// enable the cache of chaincode info
	ccprovider.EnableCCInfoCache()

	ccSrv, ccEpFunc := createChaincodeServer(peerServer, listenAddr)
	registerChaincodeSupport(ccSrv.Server(), ccEpFunc)

	log.Logger.Debugf("Running peer")

	// Register the Endorser server
	serverEndorser := endorser.NewEndorserServer()
	pb.RegisterEndorserServer(peerServer.Server(), serverEndorser)

	// Initialize gossip component
	bootstrap := viper.GetStringSlice("peer.gossip.bootstrap")

	serializedIdentity, err := mgmt.GetLocalSigningIdentityOrPanic().Serialize()
	if err != nil {
		log.Logger.Panicf("Failed serializing self identity: %v", err)
	}

	messageCryptoService := lightgossip.NewMCS(
		chain.NewChannelPolicyManagerGetter(),
		localmsp.NewSigner(),
		mgmt.NewDeserializersManager())
	secAdv := lightgossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())

	// callback function for secure dial options for gossip service
	secureDialOpts := func() []grpc.DialOption {
		var dialOpts []grpc.DialOption
		// set max send/recv msg sizes
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize()),
			grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize())))
		// set the keepalive options
		dialOpts = append(dialOpts, comm.ClientKeepaliveOptions(nil)...)

		if comm.TLSEnabled() {
			tlsCert := peerServer.ServerCertificate()
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(comm.GetCASupport().GetPeerCredentials(tlsCert)))
		} else {
			dialOpts = append(dialOpts, grpc.WithInsecure())
		}
		return dialOpts
	}
	err = service.InitLightGossipService(serializedIdentity, peerEndpoint.Address, peerServer.Server(),
		messageCryptoService, secAdv, secureDialOpts, bootstrap...)
	if err != nil {
		return err
	}
	defer service.GetLightGossipService().Stop()

	go ccSrv.Start()

	//initialize system chaincodes
	initSysCCs()

	//this brings up all the chains (including testchainid)
	chain.Initialize(func(cid string) {
		log.Logger.Debugf("Deploying system CC, for chain <%s>", cid)
		scc.DeploySysCCs(cid)
	})

	log.Logger.Infof("Starting peer with ID=[%s], network ID=[%s], address=[%s]",
		peerEndpoint.Id, viper.GetString("peer.networkId"), peerEndpoint.Address)

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

	if err := writePid(config.GetPath("peer.fileSystemPath")+"/peer.pid", os.Getpid()); err != nil {
		return err
	}

	// Start the event hub server
	if ehubGrpcServer != nil {
		go ehubGrpcServer.Start()
	}

	log.Logger.Infof("Started peer with ID=[%s], network ID=[%s], address=[%s]",
		peerEndpoint.Id, viper.GetString("peer.networkId"), peerEndpoint.Address)

	// Block until grpc server exits
	return <-serve
}

//create a CC listener using peer.chaincodeListenAddress (and if that's not set use peer.peerAddress)
func createChaincodeServer(peerServer comm.GRPCServer, peerListenAddress string) (comm.GRPCServer, ccEndpointFunc) {
	cclistenAddress := viper.GetString("peer.chaincodeListenAddress")

	var srv comm.GRPCServer
	var ccEpFunc ccEndpointFunc

	//use the chaincode address endpoint function..
	//three cases
	// -  peer.chaincodeListenAddress not specied (use peer's server)
	// -  peer.chaincodeListenAddress identical to peer.listenAddress (use peer's server)
	// -  peer.chaincodeListenAddress different and specified (create chaincode server)
	if cclistenAddress == "" {
		//...but log a warning
		log.Logger.Warnf("peer.chaincodeListenAddress is not set, use peer.listenAddress %s", peerListenAddress)

		//we are using peer address, use peer endpoint
		ccEpFunc = chain.GetPeerEndpoint
		srv = peerServer
	} else if cclistenAddress == peerListenAddress {
		//using peer's endpoint...log a  warning
		log.Logger.Warnf("peer.chaincodeListenAddress is identical to peer.listenAddress %s", cclistenAddress)

		//we are using peer address, use peer endpoint
		ccEpFunc = chain.GetPeerEndpoint
		srv = peerServer
	} else {
		c, err := chain.GetSecureConfig()
		if err != nil {
			panic(err)
		}

		srv, err = comm.NewGRPCServer(cclistenAddress, c)
		if err != nil {
			panic(err)
		}
		ccEpFunc = getChaincodeAddressEndpoint
	}

	return srv, ccEpFunc
}

//NOTE - when we implment JOIN we will no longer pass the chainID as param
//The chaincode support will come up without registering system chaincodes
//which will be registered only during join phase.
func registerChaincodeSupport(grpcServer *grpc.Server, ccEpFunc ccEndpointFunc) {
	//get user mode
	userRunsCC := chaincode.IsDevMode()

	//get chaincode startup timeout
	ccStartupTimeout := viper.GetDuration("chaincode.startuptimeout")
	if ccStartupTimeout < time.Duration(5)*time.Second {
		log.Logger.Warnf("Invalid chaincode startup timeout value %s (should be at least 5s); defaulting to 5s", ccStartupTimeout)
		ccStartupTimeout = time.Duration(5) * time.Second
	} else {
		log.Logger.Debugf("Chaincode startup timeout value set to %s", ccStartupTimeout)
	}

	ccSrv := chaincode.NewChaincodeSupport(ccEpFunc, userRunsCC, ccStartupTimeout)

	//Now that chaincode is initialized, register all system chaincodes.
	scc.RegisterSysCCs()

	pb.RegisterChaincodeSupportServer(grpcServer, ccSrv)
}

func getChaincodeAddressEndpoint() (*pb.PeerEndpoint, error) {
	//need this for the ID to create chaincode endpoint
	peerEndpoint, err := chain.GetPeerEndpoint()
	if err != nil {
		return nil, err
	}

	ccendpoint := viper.GetString("peer.chaincodeListenAddress")
	if ccendpoint == "" {
		return nil, fmt.Errorf("peer.chaincodeListenAddress not specified")
	}

	if _, _, err = net.SplitHostPort(ccendpoint); err != nil {
		return nil, err
	}

	return &pb.PeerEndpoint{
		Id:      peerEndpoint.Id,
		Address: ccendpoint,
	}, nil
}

func createEventHubServer(secureConfig comm.SecureServerConfig) (comm.GRPCServer, error) {
	var lis net.Listener
	var err error
	lis, err = net.Listen("tcp", viper.GetString("peer.events.address"))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer, err := comm.NewGRPCServerFromListener(lis, secureConfig)
	if err != nil {
		log.Logger.Errorf("Failed to return new GRPC server: %s", err)
		return nil, err
	}
	ehServer := producer.NewEventsServer(
		uint(viper.GetInt("peer.events.buffersize")),
		viper.GetDuration("peer.events.timeout"))

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
