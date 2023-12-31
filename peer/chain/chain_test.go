/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chain

import (
	"fmt"
	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/localmsp"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/common/testhelper"
	"github.com/rongzer/blockchain/light/gossip"
	ccp "github.com/rongzer/blockchain/peer/common/ccprovider"
	"github.com/rongzer/blockchain/peer/common/sysccprovider"
	"github.com/rongzer/blockchain/peer/deliverclient"
	"github.com/rongzer/blockchain/peer/deliverclient/blocksprovider"
	"github.com/rongzer/blockchain/peer/gossip/api"
	mocks "github.com/rongzer/blockchain/peer/gossip/security/securitytest"
	"github.com/rongzer/blockchain/peer/gossip/service"
	"github.com/rongzer/blockchain/peer/peertest/ccprovider"
	"github.com/rongzer/blockchain/peer/scc/scctest/scc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"os"
	"path/filepath"
	"testing"
)


type mockDeliveryClient struct {
}

func (*mockDeliveryClient) StartDeliverForChannel(chainID string, ledgerInfo blocksprovider.LedgerInfo, finalizer func()) error {
	panic("implement me")
}

func (*mockDeliveryClient) StopDeliverForChannel(chainID string) error {
	panic("implement me")
}

func (*mockDeliveryClient) Stop() {
	panic("implement me")
}

type mockDeliveryClientFactory struct {
}

func (*mockDeliveryClientFactory) Service(g service.GossipService, endpoints []string, mcs api.MessageCryptoService) (deliverclient.DeliverService, error) {
	return &mockDeliveryClient{}, nil
}

func TestCreatePeerServer(t *testing.T) {

	server, err := CreatePeerServer(":4050", comm.SecureServerConfig{})
	assert.NoError(t, err, "CreatePeerServer returned unexpected error")
	assert.Equal(t, "[::]:4050", server.Address(),
		"CreatePeerServer returned the wrong address")
	server.Stop()

	//_, err = CreatePeerServer("", comm.SecureServerConfig{})
	//log.Print(t, err, "expected CreatePeerServer to return error with missing address")
	//t.Fail()

}

func TestGetSecureConfig(t *testing.T) {

	// good config without TLS
	viper.Set("peer.tls.enabled", false)
	sc, _ := GetSecureConfig()
	assert.Equal(t, false, sc.UseTLS, "SecureConfig.UseTLS should be false")

	// good config with TLS
	viper.Set("peer.tls.enabled", true)
	viper.Set("peer.tls.cert.file", filepath.Join("testdata", "Org1-server1-cert.pem"))
	viper.Set("peer.tls.key.file", filepath.Join("testdata", "Org1-server1-key.pem"))
	viper.Set("peer.tls.rootcert.file", filepath.Join("testdata", "Org1-cert.pem"))
	sc, _ = GetSecureConfig()
	assert.Equal(t, true, sc.UseTLS, "SecureConfig.UseTLS should be true")

	// bad config with TLS
	viper.Set("peer.tls.rootcert.file", filepath.Join("testdata", "Org11-cert.pem"))
	_, err := GetSecureConfig()
	assert.Error(t, err, "GetSecureConfig should return error with bad root cert path")
	viper.Set("peer.tls.cert.file", filepath.Join("testdata", "Org11-cert.pem"))
	_, err = GetSecureConfig()
	assert.Error(t, err, "GetSecureConfig should return error with bad tls cert path")

	// disable TLS for remaining tests
	viper.Set("peer.tls.enabled", false)

}

func TestInitChain(t *testing.T) {

	chainId := "testChain"
	chainInitializer = func(cid string) {
		assert.Equal(t, chainId, cid, "chainInitializer received unexpected cid")
	}
	InitChain(chainId)
}

func TestInitialize(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/var/rongzer/test/")

	// we mock this because we can't import the chaincode package lest we create an import cycle
	ccp.RegisterChaincodeProviderFactory(&ccprovider.MockCcProviderFactory{})
	sysccprovider.RegisterSystemChaincodeProviderFactory(&scc.MocksccProviderFactory{})

	Initialize(nil)
}

func TestCreateChainFromBlock(t *testing.T) {
	viper.Set("peer.fileSystemPath", "/var/rongzer/test/")
	defer os.RemoveAll("/var/rongzer/test/")
	testChainID := "mytestchainid"
	block, err := testhelper.MakeGenesisBlock(testChainID)
	if err != nil {
		fmt.Printf("Failed to create a config block, err %s\n", err)
		t.FailNow()
	}

	// Initialize gossip service
	grpcServer := grpc.NewServer()
	socket, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "", 13611))
	assert.NoError(t, err)
	//go grpcServer.Serve(socket)
	defer grpcServer.Stop()

	testhelper.LoadMSPSetupForTesting()

	identity, _ := mgmt.GetLocalSigningIdentityOrPanic().Serialize()
	messageCryptoService := gossip.NewMCS(&mocks.ChannelPolicyManagerGetter{}, localmsp.NewSigner(), mgmt.NewDeserializersManager())
	secAdv := gossip.NewSecurityAdvisor(mgmt.NewDeserializersManager())
	var defaultSecureDialOpts = func() []grpc.DialOption {
		var dialOpts []grpc.DialOption
		dialOpts = append(dialOpts, grpc.WithInsecure())
		return dialOpts
	}
	err = service.InitGossipServiceCustomDeliveryFactory(
		identity, "localhost:13611", grpcServer,
		&mockDeliveryClientFactory{},
		messageCryptoService, secAdv, defaultSecureDialOpts)

	assert.NoError(t, err)

	go grpcServer.Serve(socket)

	err = CreateChainFromBlock(block)
	if err != nil {
		t.Fatalf("failed to create chain %s", err)
	}

	// Correct ledger
	ledger := GetLedger(testChainID)
	if ledger == nil {
		t.Fatalf("failed to get correct ledger")
	}

	// Get config block from ledger
	block, err = getCurrConfigBlockFromLedger(ledger)
	assert.NoError(t, err, "Failed to get config block from ledger")
	assert.NotNil(t, block, "Config block should not be nil")
	assert.Equal(t, uint64(0), block.Header.Number, "config block should have been block 0")

	// Bad ledger
	ledger = GetLedger("BogusChain")
	if ledger != nil {
		t.Fatalf("got a bogus ledger")
	}

	// Correct block
	block = GetCurrConfigBlock(testChainID)
	if block == nil {
		t.Fatalf("failed to get correct block")
	}

	// Bad block
	block = GetCurrConfigBlock("BogusBlock")
	if block != nil {
		t.Fatalf("got a bogus block")
	}

	// Correct PolicyManager
	pmgr := GetPolicyManager(testChainID)
	if pmgr == nil {
		t.Fatal("failed to get PolicyManager")
	}

	// Bad PolicyManager
	pmgr = GetPolicyManager("BogusChain")
	if pmgr != nil {
		t.Fatal("got a bogus PolicyManager")
	}

	// PolicyManagerGetter
	pmg := NewChannelPolicyManagerGetter()
	assert.NotNil(t, pmg, "PolicyManagerGetter should not be nil")

	pmgr, ok := pmg.Manager(testChainID)
	assert.NotNil(t, pmgr, "PolicyManager should not be nil")
	assert.Equal(t, true, ok, "expected Manage() to return true")

	// Chaos monkey test
	//Initialize(nil)
	//
	//SetCurrConfigBlock(block, testChainID)
	//
	//channels := GetChannelsInfo()
	//if len(channels) != 1 {
	//	t.Fatalf("incorrect number of channels")
	//}
}


func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	t.Log(ip)
}

func TestGetLedger(t *testing.T) {
	l := GetLedger("mytestchainid")
	assert.NotNil(t, l)
}

func TestGetPolicyManager(t *testing.T) {
	m := GetPolicyManager("mytestchainid")
	assert.NotNil(t, m)
}

func TestGetCurrConfigBlock(t *testing.T) {
	block := GetCurrConfigBlock("mytestchainid")
	assert.NotNil(t, block)
}

func TestGetMSPIDs(t *testing.T) {
	ids := GetMSPIDs("mytestchainid")
	assert.Empty(t, ids)
}

func TestGetChannelsInfo(t *testing.T) {
	infos := GetChannelsInfo()
	assert.Empty(t, infos)
}




