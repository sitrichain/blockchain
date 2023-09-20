package chain

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/configtx"
	configtxapi "github.com/rongzer/blockchain/common/configtx/config"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/peer/broadcastclient"
	"github.com/rongzer/blockchain/peer/committer"
	"github.com/rongzer/blockchain/peer/committer/txvalidator"
	"github.com/rongzer/blockchain/peer/gossip/api"
	"github.com/rongzer/blockchain/peer/gossip/service"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
)

var peerServer comm.GRPCServer

// singleton instance to manage CAs for the peer across channel config changes
var rootCASupport = comm.GetCASupport()

type chainSupport struct {
	configtxapi.Manager
	config.Application
	ledger ledger.PeerLedger
	broadcastclient.CommunicateOrderer
}

func (cs *chainSupport) Ledger() ledger.PeerLedger {
	return cs.ledger
}

func (cs *chainSupport) GetMSPIDs(cid string) []string {
	return GetMSPIDs(cid)
}

// chain is a local struct to manage objects in a chain
type chain struct {
	cs        *chainSupport
	cb        *common.Block
	committer committer.Committer
}

// chains is a local map of chainID->chainObject
var chains = struct {
	sync.RWMutex
	list map[string]*chain
}{list: make(map[string]*chain)}

var chainInitializer func(string)

func ChainsInit() {
	chains.list = nil
	chains.list = make(map[string]*chain)
	chainInitializer = func(string) { return }
}

func AddChain(cid string, ledger ledger.PeerLedger, manager configtxapi.Manager) {
	chains.Lock()
	defer chains.Unlock()

	chains.list[cid] = &chain{
		cs: &chainSupport{
			Manager: manager,
			ledger:  ledger},
	}
}

// Initialize sets up any chains that the peer has from the persistence. This
// function should be called at the start up when the ledger and gossip
// ready
func Initialize(init func(string)) {
	chainInitializer = init
	var cb *common.Block
	ledgermgmt.Initialize()
	ledgerIds, err := ledgermgmt.GetLedgerIDs()
	if err != nil {
		panic(fmt.Errorf("Error in initializing ledgermgmt: %s", err))
	}
	for _, cid := range ledgerIds {
		log.Logger.Infof("Loading chain %s", cid)
		ldger, err := ledgermgmt.OpenLedger(cid)
		if err != nil {
			log.Logger.Warnf("Failed to load ledger %s(%s)", cid, err)
			log.Logger.Debugf("Error while loading ledger %s with message %s. We continue to the next ledger rather than abort.", cid, err)
			continue
		}
		if cb, err = getCurrConfigBlockFromLedger(ldger); err != nil {
			log.Logger.Warnf("Failed to find config block on ledger %s(%s)", cid, err)
			log.Logger.Debugf("Error while looking for config block on ledger %s with message %s. We continue to the next ledger rather than abort.", cid, err)
			continue
		}
		// Create a chain if we get a valid ledger with config block
		if err = createChain(cid, ldger, cb); err != nil {
			log.Logger.Warnf("Failed to load chain %s(%s)", cid, err)
			log.Logger.Debugf("Error reloading chain %s with message %s. We continue to the next chain rather than abort.", cid, err)
			continue
		}

		InitChain(cid)
	}
}

// Take care to initialize chain after peer joined, for example deploys system CCs
func InitChain(cid string) {
	if chainInitializer != nil {
		// Initialize chaincode, namely deploy system CC
		log.Logger.Infof("Init chain %s", cid)
		chainInitializer(cid)
	}
}

func getCurrConfigBlockFromLedger(ledger ledger.PeerLedger) (*common.Block, error) {
	log.Logger.Debugf("Getting config block")

	// get last block.  Last block number is Height-1
	blockchainInfo, err := ledger.GetBlockchainInfo()
	if err != nil {
		return nil, err
	}
	lastBlock, err := ledger.GetBlockByNumber(blockchainInfo.Height - 1)
	if err != nil {
		return nil, err
	}

	// get most recent config block location from last block metadata
	configBlockIndex, err := utils.GetLastConfigIndexFromBlock(lastBlock)
	if err != nil {
		return nil, err
	}

	// get most recent config block
	configBlock, err := ledger.GetBlockByNumber(configBlockIndex)
	if err != nil {
		return nil, err
	}

	log.Logger.Debugf("Got config block[%d]", configBlockIndex)
	return configBlock, nil
}

// createChain creates a new chain object and insert it into the chains
func createChain(cid string, ledger ledger.PeerLedger, cb *common.Block) error {

	envelopeConfig, err := utils.ExtractEnvelope(cb, 0)
	if err != nil {
		return err
	}

	configtxInitializer := configtx.NewInitializer()

	gossipEventer := service.GetGossipService().NewConfigEventer()

	gossipCallbackWrapper := func(cm configtxapi.Manager) {
		ac, ok := configtxInitializer.ApplicationConfig()
		if !ok {
			// TODO, handle a missing ApplicationConfig more gracefully
			ac = nil
		}
		gossipEventer.ProcessConfigUpdate(&chainSupport{
			Manager:     cm,
			Application: ac,
		})
		service.GetGossipService().SuspectPeers(func(identity api.PeerIdentityType) bool {
			// TODO: this is a place-holder that would somehow make the MSP layer suspect
			// that a given certificate is revoked, or its intermediate CA is revoked.
			// In the meantime, before we have such an ability, we return true in order
			// to suspect ALL identities in order to validate all of them.
			return true
		})
	}

	trustedRootsCallbackWrapper := func(cm configtxapi.Manager) {
		updateTrustedRoots(cm)
	}

	configtxManager, err := configtx.NewConfigManager(
		envelopeConfig,
		configtxInitializer,
		[]func(cm configtxapi.Manager){gossipCallbackWrapper, trustedRootsCallbackWrapper},
	)
	if err != nil {
		return err
	}

	// TODO remove once all references to mspmgmt are gone from peer code
	mspmgmt.XXXSetMSPManager(cid, configtxManager.MSPManager())

	// 创建bradcastClient

	bc := broadcastclient.NewBroadcastClient()

	ac, ok := configtxInitializer.ApplicationConfig()
	if !ok {
		ac = nil
	}
	cs := &chainSupport{
		Manager:            configtxManager,
		Application:        ac, // TODO, refactor as this is accessible through Manager
		ledger:             ledger,
		CommunicateOrderer: bc,
	}

	c := committer.NewLedgerCommitterReactive(ledger, txvalidator.NewTxValidator(cs), func(block *common.Block) error {
		chainID, err := utils.GetChainIDFromBlock(block)
		if err != nil {
			return err
		}
		return SetCurrConfigBlock(block, chainID)
	})

	ordererAddresses := configtxManager.ChannelConfig().OrdererAddresses()
	if len(ordererAddresses) == 0 {
		return errors.New("No ordering service endpoint provided in configuration block")
	}
	// 重载orderer地址配置
	ordererAddress := viper.GetString("orderer.address")
	if len(ordererAddress) > 0 {
		ordererAddresses = strings.Split(ordererAddress, ",")
	}
	service.GetGossipService().InitializeChannel(cs.ChainID(), c, ordererAddresses)

	chains.Lock()
	defer chains.Unlock()
	chains.list[cid] = &chain{
		cs:        cs,
		cb:        cb,
		committer: c,
	}

	// 在这启动定时器向orderer发消息
	go sendOrdererTimer(cs.ChainID())
	return nil
}

// CreateChainFromBlock creates a new chain from config block
func CreateChainFromBlock(cb *common.Block) error {
	cid, err := utils.GetChainIDFromBlock(cb)
	if err != nil {
		return err
	}

	var l ledger.PeerLedger
	if l, err = ledgermgmt.CreateLedger(cb); err != nil {
		return fmt.Errorf("Cannot create ledger from genesis block, due to %s", err)
	}

	return createChain(cid, l, cb)
}

// GetLedger returns the ledger of the chain with chain ID. Note that this
// call returns nil if chain cid has not been created.
func GetLedger(cid string) ledger.PeerLedger {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs.ledger
	}
	return nil
}

// GetPolicyManager returns the policy manager of the chain with chain ID. Note that this
// call returns nil if chain cid has not been created.
func GetPolicyManager(cid string) policies.Manager {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cs.PolicyManager()
	}
	return nil
}

// GetCurrConfigBlock returns the cached config block of the specified chain.
// Note that this call returns nil if chain cid has not been created.
func GetCurrConfigBlock(cid string) *common.Block {
	chains.RLock()
	defer chains.RUnlock()
	if c, ok := chains.list[cid]; ok {
		return c.cb
	}
	return nil
}

// updates the trusted roots for the peer based on updates to channels
func updateTrustedRoots(cm configtxapi.Manager) {
	// this is triggered on per channel basis so first update the roots for the channel
	log.Logger.Debugf("Updating trusted root authorities for channel %s", cm.ChainID())
	var secureConfig comm.SecureServerConfig
	var err error
	// only run is TLS is enabled
	secureConfig, err = GetSecureConfig()
	if err == nil && secureConfig.UseTLS {
		buildTrustedRootsForChain(cm)

		// now iterate over all roots for all app and orderer chains
		var trustedRoots [][]byte
		rootCASupport.RLock()
		defer rootCASupport.RUnlock()
		for _, roots := range rootCASupport.AppRootCAsByChain {
			trustedRoots = append(trustedRoots, roots...)
		}
		// also need to append statically configured root certs
		if len(secureConfig.ClientRootCAs) > 0 {
			trustedRoots = append(trustedRoots, secureConfig.ClientRootCAs...)
		}
		if len(secureConfig.ServerRootCAs) > 0 {
			trustedRoots = append(trustedRoots, secureConfig.ServerRootCAs...)
		}

		server := GetPeerServer()
		// now update the client roots for the peerServer
		if server != nil {
			err := server.SetClientRootCAs(trustedRoots)
			if err != nil {
				msg := "Failed to update trusted roots for peer from latest config " +
					"block.  This peer may not be able to communicate " +
					"with members of channel %s (%s)"
				log.Logger.Warnf(msg, cm.ChainID(), err)
			}
		}
	}
}

// populates the appRootCAs and orderRootCAs maps by getting the
// root and intermediate certs for all msps associated with the MSPManager
func buildTrustedRootsForChain(cm configtxapi.Manager) {
	rootCASupport.Lock()
	defer rootCASupport.Unlock()

	var appRootCAs [][]byte
	var ordererRootCAs [][]byte
	appOrgMSPs := make(map[string]struct{})
	ac, ok := cm.ApplicationConfig()
	if ok {
		//loop through app orgs and build map of MSPIDs
		for _, appOrg := range ac.Organizations() {
			appOrgMSPs[appOrg.MSPID()] = struct{}{}
		}
	}

	cid := cm.ChainID()
	log.Logger.Debugf("updating root CAs for channel [%s]", cid)
	msps, err := cm.MSPManager().GetMSPs()
	if err != nil {
		log.Logger.Errorf("Error getting root CAs for channel %s (%s)", cid, err)
	}
	if err == nil {
		for k, v := range msps {
			// check to see if this is a BLOCKCHAIN MSP
			if v.GetType() == msp.BLOCKCHAIN {
				for _, root := range v.GetTLSRootCerts() {
					// check to see of this is an app org MSP
					if _, ok := appOrgMSPs[k]; ok {
						log.Logger.Debugf("adding app root CAs for MSP [%s]", k)
						appRootCAs = append(appRootCAs, root)
					} else {
						log.Logger.Debugf("adding orderer root CAs for MSP [%s]", k)
						ordererRootCAs = append(ordererRootCAs, root)
					}
				}
				for _, intermediate := range v.GetTLSIntermediateCerts() {
					// check to see of this is an app org MSP
					if _, ok := appOrgMSPs[k]; ok {
						log.Logger.Debugf("adding app root CAs for MSP [%s]", k)
						appRootCAs = append(appRootCAs, intermediate)
					} else {
						log.Logger.Debugf("adding orderer root CAs for MSP [%s]", k)
						ordererRootCAs = append(ordererRootCAs, intermediate)
					}
				}
			}
		}
		rootCASupport.AppRootCAsByChain[cid] = appRootCAs
		rootCASupport.OrdererRootCAsByChain[cid] = ordererRootCAs
	}
}

// GetMSPIDs returns the ID of each application MSP defined on this chain
func GetMSPIDs(cid string) []string {
	chains.RLock()
	defer chains.RUnlock()

	if c, ok := chains.list[cid]; ok {
		if c == nil || c.cs == nil {
			return nil
		}
		ac, ok := c.cs.ApplicationConfig()
		if !ok || ac.Organizations() == nil {
			return nil
		}

		orgs := ac.Organizations()
		toret := make([]string, len(orgs))
		i := 0
		for _, org := range orgs {
			toret[i] = org.MSPID()
			i++
		}

		return toret
	}
	return nil
}

// SetCurrConfigBlock sets the current config block of the specified chain
func SetCurrConfigBlock(block *common.Block, cid string) error {
	chains.Lock()
	defer chains.Unlock()
	if c, ok := chains.list[cid]; ok {
		c.cb = block
		return nil
	}
	return fmt.Errorf("Chain %s doesn't exist on the peer", cid)
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// GetChannelsInfo returns an array with information about all channels for
// this peer
func GetChannelsInfo() []*pb.ChannelInfo {
	// array to store metadata for all channels
	var channelInfoArray []*pb.ChannelInfo

	chains.RLock()
	defer chains.RUnlock()
	for key := range chains.list {
		channelInfo := &pb.ChannelInfo{ChannelId: key}

		// add this specific chaincode's metadata to the array of all chaincodes
		channelInfoArray = append(channelInfoArray, channelInfo)
	}

	return channelInfoArray
}

// NewChannelPolicyManagerGetter returns a new instance of ChannelPolicyManagerGetter
func NewChannelPolicyManagerGetter() policies.ChannelPolicyManagerGetter {
	return &channelPolicyManagerGetter{}
}

type channelPolicyManagerGetter struct{}

func (c *channelPolicyManagerGetter) Manager(channelID string) (policies.Manager, bool) {
	policyManager := GetPolicyManager(channelID)
	return policyManager, policyManager != nil
}

// CreatePeerServer creates an instance of comm.GRPCServer
// This server is used for peer communications
func CreatePeerServer(listenAddress string,
	secureConfig comm.SecureServerConfig) (comm.GRPCServer, error) {

	var err error
	peerServer, err = comm.NewGRPCServer(listenAddress, secureConfig)
	if err != nil {
		log.Logger.Errorf("Failed to create peer server (%s)", err)
		return nil, err
	}
	return peerServer, nil
}

// GetPeerServer returns the peer server instance
func GetPeerServer() comm.GRPCServer {
	return peerServer
}

// 发送消息至orderer
func sendOrdererTimer(channelID string) {
	timer1 := time.NewTicker(5 * time.Second)
	id := viper.GetString("peer.id")
	address := viper.GetString("peer.address")
	for {
		select {
		case <-timer1.C:
			sendOrdererTimerFunc(channelID, id, address)
		}
	}
}

func sendOrdererTimerFunc(channelID, id, address string) {
	go func() {
		peerInfo := &common.PeerInfo{}
		peerInfo.Id = id
		peerInfo.Address = address

		targetLedger := GetLedger(channelID)

		binfo, err := targetLedger.GetBlockchainInfo()
		if err != nil {
			log.Logger.Errorf("Failed to get block info with error %s", err)
		} else {
			peerInfo.BlockInfo = binfo
		}

		buf, err := peerInfo.Marshal()
		if err != nil {
			log.Logger.Errorf("marshal peerinfo error: %s ", err)
			return
		}

		rbcMessage := &common.RBCMessage{ChainID: channelID, Type: 11, Data: buf}
		_, err = broadcastclient.GetCommunicateOrderer().SendToOrderer(rbcMessage)
		if err != nil {
			log.Logger.Errorf("send peer status info message To Orderer error: %s", err)
			return
		}

		log.Logger.Debugf("[Chain: %s] Send peer status info message To Orderer", channelID)
	}()
}
