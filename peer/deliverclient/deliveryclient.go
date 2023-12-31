package deliverclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/deliverclient/blocksprovider"
	"github.com/rongzer/blockchain/peer/gossip/api"
	"github.com/rongzer/blockchain/protos/orderer"
	"google.golang.org/grpc"
)

var (
	connTimeout               = time.Second * 3
	reConnectBackoffThreshold = float64(time.Hour)
)

// DeliverService used to communicate with orderers to obtain
// new blocks and send them to the committer service
type DeliverService interface {
	// StartDeliverForChannel dynamically starts delivery of new blocks from ordering service
	// to channel peers.
	StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error

	// StopDeliverForChannel dynamically stops delivery of new blocks from ordering service
	// to channel peers.
	StopDeliverForChannel(chainID string) error

	// Stop terminates delivery service and closes the connection
	Stop()
}

// deliverServiceImpl the implementation of the delivery service
// maintains connection to the ordering service and maps of
// blocks providers
type deliverServiceImpl struct {
	conf           *Config
	blockProviders map[string]blocksprovider.BlocksProvider
	lock           sync.RWMutex
	stopping       bool
}

// Config dictates the DeliveryService's properties,
// namely how it connects to an ordering service endpoint,
// how it verifies messages received from it,
// and how it disseminates the messages to other peers
type Config struct {
	// ConnFactory returns a function that creates a connection to an endpoint
	ConnFactory func(channelID string) func(endpoint string) (*grpc.ClientConn, error)
	// ABCFactory creates an AtomicBroadcastClient out of a connection
	ABCFactory func(*grpc.ClientConn) orderer.AtomicBroadcastClient
	// CryptoSvc performs cryptographic actions like message verification and signing
	// and identity validation
	CryptoSvc api.MessageCryptoService
	// Gossip enables to enumerate peers in the channel, send a message to peers,
	// and add a block to the gossip state transfer layer
	Gossip blocksprovider.GossipServiceAdapter
	// Endpoints specifies the endpoints of the ordering service
	Endpoints []string
}

// NewDeliverService construction function to create and initialize
// delivery service instance. It tries to establish connection to
// the specified in the configuration ordering service, in case it
// fails to dial to it, return nil
func NewDeliverService(conf *Config) (DeliverService, error) {
	ds := &deliverServiceImpl{
		conf:           conf,
		blockProviders: make(map[string]blocksprovider.BlocksProvider),
	}
	if err := ds.validateConfiguration(); err != nil {
		return nil, err
	}
	return ds, nil
}

func (d *deliverServiceImpl) validateConfiguration() error {
	conf := d.conf
	if len(conf.Endpoints) == 0 {
		return errors.New("No endpoints specified")
	}
	if conf.Gossip == nil {
		return errors.New("No gossip provider specified")
	}
	if conf.ABCFactory == nil {
		return errors.New("No AtomicBroadcast factory specified")
	}
	if conf.ConnFactory == nil {
		return errors.New("No connection factory specified")
	}
	if conf.CryptoSvc == nil {
		return errors.New("No crypto service specified")
	}
	return nil
}

// StartDeliverForChannel starts blocks delivery for channel
// initializes the grpc stream for given chainID, creates blocks provider instance
// that spawns in go routine to read new blocks starting from the position provided by ledger
// info instance.
func (d *deliverServiceImpl) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.stopping {
		errMsg := fmt.Sprintf("Delivery service is stopping cannot join a new channel %s", chainID)
		log.Logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	if _, exist := d.blockProviders[chainID]; exist {
		errMsg := fmt.Sprintf("Delivery service - block provider already exists for %s found, can't start delivery", chainID)
		log.Logger.Errorf(errMsg)
		return errors.New(errMsg)
	} else {
		client := d.newClient(chainID, ledgerInfo)
		log.Logger.Debug("This peer will pass blocks from orderer service to other peers for channel", chainID)

		requester := &blocksRequester{
			chainID: chainID,
		}
		rFunc := func(ledgerInfoProvider blocksprovider.LedgerInfo) error {
			log.Logger.Infof("RequestBlocks1")
			return requester.RequestBlocks(ledgerInfoProvider)
		}

		d.blockProviders[chainID] = blocksprovider.NewBlocksProvider(chainID, client, d.conf.Gossip, d.conf.CryptoSvc, rFunc, ledgerInfo)
		go d.launchBlockProvider(chainID, finalizer)
	}
	return nil
}

func (d *deliverServiceImpl) launchBlockProvider(chainID string, finalizer func()) {
	d.lock.RLock()
	pb := d.blockProviders[chainID]
	d.lock.RUnlock()
	if pb == nil {
		log.Logger.Info("Block delivery for channel", chainID, "was stopped before block provider started")
		return
	}
	pb.DeliverBlocks()
	finalizer()
}

// StopDeliverForChannel stops blocks delivery for channel by stopping channel block provider
func (d *deliverServiceImpl) StopDeliverForChannel(chainID string) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.stopping {
		errMsg := fmt.Sprintf("Delivery service is stopping, cannot stop delivery for channel %s", chainID)
		log.Logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	if client, exist := d.blockProviders[chainID]; exist {
		client.Stop()
		delete(d.blockProviders, chainID)
		log.Logger.Debug("This peer will stop pass blocks from orderer service to other peers")
	} else {
		errMsg := fmt.Sprintf("Delivery service - no block provider for %s found, can't stop delivery", chainID)
		log.Logger.Errorf(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

// Stop all service and release resources
func (d *deliverServiceImpl) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()
	// Marking flag to indicate the shutdown of the delivery service
	d.stopping = true

	for _, client := range d.blockProviders {
		client.Stop()
	}
}

func (d *deliverServiceImpl) newClient(chainID string, ledgerInfoProvider blocksprovider.LedgerInfo) *broadcastClient {
	requester := &blocksRequester{
		chainID: chainID,
	}
	broadcastSetup := func(bd blocksprovider.BlocksDeliverer) error {
		log.Logger.Debug("RequestBlocks2")
		return requester.RequestBlocks(ledgerInfoProvider)
	}
	backoffPolicy := func(attemptNum int, elapsedTime time.Duration) (time.Duration, bool) {
		// 不中止重连请求
		if attemptNum >= 10 {
			//每分钟尝试一次
			attemptNum = 6
			log.Logger.Warnf("Delivery Attempts (%d) or elapsed time (%v) exhausted", attemptNum, elapsedTime)
		}
		//修改结束
		sleepIncrement := float64(time.Millisecond * 500)
		attempt := float64(attemptNum)
		return time.Duration(math.Min(math.Pow(2, attempt)*sleepIncrement, reConnectBackoffThreshold)), true
	}
	connProd := comm.NewConnectionProducer(d.conf.ConnFactory(chainID), d.conf.Endpoints)
	bClient := NewBroadcastClient(connProd, d.conf.ABCFactory, broadcastSetup, backoffPolicy)
	requester.client = bClient
	return bClient
}

func DefaultConnectionFactory(channelID string) func(endpoint string) (*grpc.ClientConn, error) {
	return func(endpoint string) (*grpc.ClientConn, error) {
		dialOpts := []grpc.DialOption{grpc.WithBlock()}
		// set max send/recv msg sizes
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize()),
			grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize())))
		// set the keepalive options
		kaOpts := comm.DefaultKeepaliveOptions()
		dialOpts = append(dialOpts, comm.ClientKeepaliveOptions(kaOpts)...)

		if comm.TLSEnabled() {
			creds, err := comm.GetCASupport().GetDeliverServiceCredentials(channelID)
			if err != nil {
				log.Logger.Errorf("Failed obtaining credentials for channel %s: %v", channelID, err)
				dialOpts = append(dialOpts, grpc.WithInsecure())
			} else {
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
			}
		} else {
			dialOpts = append(dialOpts, grpc.WithInsecure())
		}
		grpc.EnableTracing = true
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, connTimeout)
		return grpc.DialContext(ctx, endpoint, dialOpts...)
	}
}

func DefaultABCFactory(conn *grpc.ClientConn) orderer.AtomicBroadcastClient {
	return orderer.NewAtomicBroadcastClient(conn)
}
