package state

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rongzer/blockchain/peer/gossip/discovery"
	"github.com/rongzer/blockchain/protos/msp"
	"github.com/rongzer/blockchain/protos/utils"
	"os"
	"strconv"
	"testing"
	"time"

	pb "github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/testhelper"
	"github.com/rongzer/blockchain/peer/committer"
	"github.com/rongzer/blockchain/peer/gossip/api"
	"github.com/rongzer/blockchain/peer/gossip/comm"
	"github.com/rongzer/blockchain/peer/gossip/common"
	"github.com/rongzer/blockchain/peer/gossip/gossip"
	"github.com/rongzer/blockchain/peer/gossip/identity"
	"github.com/rongzer/blockchain/peer/gossip/light"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	pcomm "github.com/rongzer/blockchain/protos/common"
	proto "github.com/rongzer/blockchain/protos/gossip"
	ppeer "github.com/rongzer/blockchain/protos/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	portPrefix = 7610
)

var orgID = []byte("ORG1")

type peerIdentityAcceptor func(identity api.PeerIdentityType) error

var noopPeerIdentityAcceptor = func(identity api.PeerIdentityType) error {
	return nil
}

type joinChanMsg struct {
}

// SequenceNumber returns the sequence number of the block that the message
// is derived from
func (*joinChanMsg) SequenceNumber() uint64 {
	return uint64(time.Now().UnixNano())
}

// Members returns the organizations of the channel
func (jcm *joinChanMsg) Members() []api.OrgIdentityType {
	return []api.OrgIdentityType{orgID}
}

// AnchorPeersOf returns the anchor peers of the given organization
func (jcm *joinChanMsg) AnchorPeersOf(org api.OrgIdentityType) []api.AnchorPeer {
	return []api.AnchorPeer{}
}

type orgCryptoService struct {
}

// OrgByPeerIdentity returns the OrgIdentityType
// of a given peer identity
func (*orgCryptoService) OrgByPeerIdentity(identity api.PeerIdentityType) api.OrgIdentityType {
	return orgID
}

// Verify verifies a JoinChannelMessage, returns nil on success,
// and an error on failure
func (*orgCryptoService) Verify(joinChanMsg api.JoinChannelMessage) error {
	return nil
}

type cryptoServiceMock struct {
	acceptor peerIdentityAcceptor
}

// GetPKIidOfCert returns the PKI-ID of a peer's identity
func (*cryptoServiceMock) GetPKIidOfCert(peerIdentity api.PeerIdentityType) common.PKIidType {
	return common.PKIidType(peerIdentity)
}

// VerifyBlock returns nil if the block is properly signed,
// else returns error
func (*cryptoServiceMock) VerifyBlock(chainID common.ChainID, seqNum uint64, signedBlock []byte) error {
	return nil
}

// Sign signs msg with this peer's signing key and outputs
// the signature if no error occurred.
func (*cryptoServiceMock) Sign(msg []byte) ([]byte, error) {
	clone := make([]byte, len(msg))
	copy(clone, msg)
	return clone, nil
}

// Verify checks that signature is a valid signature of message under a peer's verification key.
// If the verification succeeded, Verify returns nil meaning no error occurred.
// If peerCert is nil, then the signature is verified against this peer's verification key.
func (*cryptoServiceMock) Verify(peerIdentity api.PeerIdentityType, signature, message []byte) error {
	equal := bytes.Equal(signature, message)
	if !equal {
		return fmt.Errorf("Wrong signature:%v, %v", signature, message)
	}
	return nil
}

// VerifyByChannel checks that signature is a valid signature of message
// under a peer's verification key, but also in the context of a specific channel.
// If the verification succeeded, Verify returns nil meaning no error occurred.
// If peerIdentity is nil, then the signature is verified against this peer's verification key.
func (cs *cryptoServiceMock) VerifyByChannel(chainID common.ChainID, peerIdentity api.PeerIdentityType, signature, message []byte) error {
	return cs.acceptor(peerIdentity)
}

func (*cryptoServiceMock) ValidateIdentity(peerIdentity api.PeerIdentityType) error {
	return nil
}

func bootPeers(ids ...int) []string {
	peers := []string{}
	for _, id := range ids {
		peers = append(peers, fmt.Sprintf("localhost:%d", id+portPrefix))
	}
	return peers
}

// Simple presentation of peer which includes only
// communication module, gossip and state transfer
type peerNode struct {
	port   int
	g      gossip.Gossip
	s      GossipStateProvider
	cs     *cryptoServiceMock
	commit committer.Committer
}

// Shutting down all modules used
func (node *peerNode) shutdown() {
	node.s.Stop()
	node.g.Stop()
}

type mockCommitter struct {
	mock.Mock
}

func (mc *mockCommitter) Commit(block *pcomm.Block, i int) error {
	mc.Called(block)
	return nil
}

func (mc *mockCommitter) LedgerHeight() (uint64, error) {
	if mc.Called().Get(1) == nil {
		return mc.Called().Get(0).(uint64), nil
	}
	return mc.Called().Get(0).(uint64), mc.Called().Get(1).(error)
}

func (mc *mockCommitter) GetBlocks(blockSeqs []uint64) []*pcomm.Block {
	if mc.Called(blockSeqs).Get(0) == nil {
		return nil
	}
	return mc.Called(blockSeqs).Get(0).([]*pcomm.Block)
}

func (*mockCommitter) Close() {
}

type GossipMock struct {
	mock.Mock
}

func (*GossipMock) SuspectPeers(s api.PeerSuspector) {
	panic("implement me")
}

func (*GossipMock) Send(msg *proto.GossipMessage, peers ...*comm.RemotePeer) {
	panic("implement me")
}

func (*GossipMock) Peers() []discovery.NetworkMember {
	panic("implement me")
}

func (*GossipMock) PeersOfChannel(common.ChainID) []discovery.NetworkMember {
	return nil
}

func (*GossipMock) UpdateMetadata(metadata []byte) {
	panic("implement me")
}

func (*GossipMock) UpdateChannelMetadata(metadata []byte, chainID common.ChainID) {

}

func (*GossipMock) Gossip(msg *proto.GossipMessage) {
	panic("implement me")
}

func (g *GossipMock) Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *proto.GossipMessage, <-chan proto.ReceivedMessage) {
	args := g.Called(acceptor, passThrough)
	if args.Get(0) == nil {
		return nil, args.Get(1).(<-chan proto.ReceivedMessage)
	}
	return args.Get(0).(<-chan *proto.GossipMessage), nil
}

func (g *GossipMock) JoinChan(joinMsg api.JoinChannelMessage, chainID common.ChainID) {
}

func (*GossipMock) Stop() {
}

// MockValidator implements a mock validation useful for testing
type MockValidator struct {
}

// Validate does nothing, returning no error
func (m *MockValidator) Validate(block *pcomm.Block) error {
	return nil
}

// Default configuration to be used for gossip and communication modules
func newGossipConfig(id int, boot ...int) *gossip.Config {
	port := id + portPrefix
	return &gossip.Config{
		BindPort:                   port,
		BootstrapPeers:             bootPeers(boot...),
		ID:                         fmt.Sprintf("p%d", id),
		MaxBlockCountToStore:       0,
		MaxPropagationBurstLatency: time.Duration(10) * time.Millisecond,
		MaxPropagationBurstSize:    10,
		PropagateIterations:        1,
		PropagatePeerNum:           3,
		PullInterval:               time.Duration(4) * time.Second,
		PullPeerNum:                5,
		InternalEndpoint:           fmt.Sprintf("localhost:%d", port),
		PublishCertPeriod:          10 * time.Second,
		RequestStateInfoInterval:   4 * time.Second,
		PublishStateInfoInterval:   4 * time.Second,
	}
}

// Create gossip instance
func newGossipInstance(config *gossip.Config, mcs api.MessageCryptoService) gossip.Gossip {
	id := api.PeerIdentityType(config.InternalEndpoint)
	idMapper := identity.NewIdentityMapper(mcs, id)
	return gossip.NewGossipService(config, nil, &orgCryptoService{}, mcs,
		idMapper, id, nil)
}

// Create new instance of KVLedger to be used for testing
func newCommitter(id int) committer.Committer {
	cb, _ := testhelper.MakeGenesisBlock(strconv.Itoa(id))
	ledger, _ := ledgermgmt.CreateLedger(cb)
	return committer.NewLedgerCommitterReactive(ledger, &MockValidator{}, func(_ *pcomm.Block) error { return nil })
}

// Constructing pseudo peer node, simulating only gossip and state transfer part
func newPeerNodeWithGossip(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor, g gossip.Gossip) *peerNode {
	cs := &cryptoServiceMock{acceptor: acceptor}
	// Gossip component based on configuration provided and communication module
	if g == nil {
		g = newGossipInstance(config, &cryptoServiceMock{acceptor: noopPeerIdentityAcceptor})
	}

	log.Logger.Debug("Joinning channel", "testchainid")
	g.JoinChan(&joinChanMsg{}, common.ChainID("testchainid"))

	// Initialize pseudo peer simulator, which has only three
	// basic parts

	// 初始化memberCertData，模拟存储会员证书的世界状态数据库
	MemberCertData.Lock()
	defer MemberCertData.Unlock()
	MemberCertData.MapBuf["www"] = "www"

	sp := NewGossipStateProvider("testchainid", g, committer, cs)
	if sp == nil {
		return nil
	}

	return &peerNode{
		port:   config.BindPort,
		g:      g,
		s:      sp,
		commit: committer,
		cs:     cs,
	}
}

func newLightNodeWithGossip(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor, g gossip.Gossip, sampleTxMember string) *peerNode {
	cs := &cryptoServiceMock{acceptor: acceptor}
	// Gossip component based on configuration provided and communication module
	if g == nil {
		g = newGossipInstance(config, &cryptoServiceMock{acceptor: noopPeerIdentityAcceptor})
	}

	log.Logger.Debug("Joinning channel", "testchainid")
	g.JoinChan(&joinChanMsg{}, common.ChainID("testchainid"))

	// Initialize pseudo peer simulator, which has only three
	// basic parts
	viper.Set("peer.gossip.fromMembers", sampleTxMember)
	sp := light.NewGossipStateProvider("testchainid", g, committer, cs)
	if sp == nil {
		return nil
	}

	return &peerNode{
		port:   config.BindPort,
		g:      g,
		s:      sp,
		commit: committer,
		cs:     cs,
	}
}

// Constructing pseudo peer node, simulating only gossip and state transfer part
func newPeerNode(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor) *peerNode {
	return newPeerNodeWithGossip(config, committer, acceptor, nil)
}

func newLightNode(config *gossip.Config, committer committer.Committer, acceptor peerIdentityAcceptor, sampleTxMember string) *peerNode {
	return newLightNodeWithGossip(config, committer, acceptor, nil, sampleTxMember)
}

func newEnvBytes() []byte {
	// 模拟的交易数据
	id := utils.MarshalOrPanic(&msp.SerializedIdentity{Mspid: "DEFAULT", IdBytes: []byte("www")})
	nonce := utils.CreateNonceOrPanic()
	signatureHeader := utils.MarshalOrPanic(&pcomm.SignatureHeader{
		Creator: id,
		Nonce:   nonce,
	})
	taa := &ppeer.TransactionAction{Header: signatureHeader, Payload: nil}
	taas := make([]*ppeer.TransactionAction, 1)
	taas[0] = taa
	transaction := &ppeer.Transaction{Actions: taas}
	txBytes := utils.MarshalOrPanic(transaction)
	ccs := utils.MarshalOrPanic(&ppeer.ChaincodeSpec{ChaincodeId: &ppeer.ChaincodeID{Name: "testChaincode"}})
	channelHeader := utils.MarshalOrPanic(&pcomm.ChannelHeader{
		Extension: ccs,
	})
	header := &pcomm.Header{
		ChannelHeader:   channelHeader,
		SignatureHeader: signatureHeader, // 用相同的signatureHeader
	}
	payLoadBytes := utils.MarshalOrPanic(&pcomm.Payload{Header: header, Data: txBytes})
	envBytes := utils.MarshalOrPanic(&pcomm.Envelope{Payload: payLoadBytes, Signature: nil, Attachs: nil})

	return envBytes
}

func TestPeerGossipDirectMsg(t *testing.T) {
	mc := &mockCommitter{}
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	g := &GossipMock{}
	g.On("Accept", mock.Anything, false).Return(make(<-chan *proto.GossipMessage), nil)
	g.On("Accept", mock.Anything, true).Return(nil, make(<-chan proto.ReceivedMessage))
	p := newPeerNodeWithGossip(newGossipConfig(0), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
	p.s.(*GossipStateProviderImpl).handleStateRequest(nil)
	p.s.(*GossipStateProviderImpl).directMessage(nil)
	sMsg, _ := p.s.(*GossipStateProviderImpl).stateRequestMessage(uint64(10), uint64(8)).NoopSign()
	req := &comm.ReceivedMessageImpl{
		SignedGossipMessage: sMsg,
	}
	p.s.(*GossipStateProviderImpl).directMessage(req)
}

func TestPeerGossipReception(t *testing.T) {
	signalChan := make(chan struct{})
	rawblock := &pcomm.Block{
		Header: &pcomm.BlockHeader{
			Number: uint64(1),
		},
		Data: &pcomm.BlockData{
			Data: [][]byte{},
		},
	}
	b, _ := pb.Marshal(rawblock)

	createChan := func(signalChan chan struct{}) <-chan *proto.GossipMessage {
		c := make(chan *proto.GossipMessage)
		gMsg := &proto.GossipMessage{
			Channel: []byte("AAA"),
			Content: &proto.GossipMessage_DataMsg{
				DataMsg: &proto.DataMessage{
					Payload: &proto.Payload{
						SeqNum: 1,
						Data:   b,
					},
				},
			},
		}
		go func(c chan *proto.GossipMessage) {
			// Wait for Accept() to be called
			<-signalChan
			// Simulate a message reception from the gossip component with an invalid channel
			c <- gMsg
			gMsg.Channel = []byte("testchainid")
			// Simulate a message reception from the gossip component
			c <- gMsg
		}(c)
		return c
	}

	g := &GossipMock{}
	rmc := createChan(signalChan)
	g.On("Accept", mock.Anything, false).Return(rmc, nil).Run(func(_ mock.Arguments) {
		signalChan <- struct{}{}
	})
	g.On("Accept", mock.Anything, true).Return(nil, make(<-chan proto.ReceivedMessage))
	mc := &mockCommitter{}
	receivedChan := make(chan struct{})
	mc.On("Commit", mock.Anything).Run(func(arguments mock.Arguments) {
		block := arguments.Get(0).(*pcomm.Block)
		assert.Equal(t, uint64(1), block.Header.Number)
		receivedChan <- struct{}{}
	})
	mc.On("LedgerHeight", mock.Anything).Return(uint64(1), nil)
	p := newPeerNodeWithGossip(newGossipConfig(1), mc, noopPeerIdentityAcceptor, g)
	defer p.shutdown()
	select {
	case <-receivedChan:
	case <-time.After(time.Second * 15):
		assert.Fail(t, "Didn't commit a block within a timely manner")
	}
}

func TestPeerGossipAccessControl(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	testPath := "/tmp/rongzer/test/gossipstate"
	viper.Set("peer.fileSystemPath", testPath)
	// 账本管理器初始化
	func() {
		os.RemoveAll(testPath)
		ledgermgmt.Initialize()
	}()
	defer func() {
		ledgermgmt.Close()
		os.RemoveAll(testPath)
	}()
	bootstrapSetSize := 1
	bootstrapSet := make([]*peerNode, 0)

	authorizedPeers := map[string]struct{}{
		"localhost:5610": {},
		"localhost:5615": {},
		"localhost:5618": {},
		"localhost:5621": {},
	}

	blockPullPolicy := func(identity api.PeerIdentityType) error {
		if _, isAuthorized := authorizedPeers[string(identity)]; isAuthorized {
			return nil
		}
		return errors.New("Not authorized")
	}

	for i := 0; i < bootstrapSetSize; i++ {
		commit := newCommitter(i)
		bootstrapSet = append(bootstrapSet, newPeerNode(newGossipConfig(10+i), commit, blockPullPolicy))
	}

	defer func() {
		for _, p := range bootstrapSet {
			p.shutdown()
		}
	}()

	msgCount := 1

	for i := 1; i <= msgCount; i++ {
		rawblock := pcomm.NewBlock(uint64(i), []byte{})
		if b, err := pb.Marshal(rawblock); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootstrapSet[0].s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	assert.Equal(t, msgCount, bootstrapSet[0].s.GetPayloadSize())

	standardPeerSetSize := 1
	peersSet := make([]*peerNode, 0)

	for i := 0; i < standardPeerSetSize; i++ {
		commit := newCommitter(bootstrapSetSize + i)
		peersSet = append(peersSet, newPeerNode(newGossipConfig(20+i, 10), commit, blockPullPolicy))
	}

	defer func() {
		for _, p := range peersSet {
			p.shutdown()
		}
	}()

	waitUntilTrueOrTimeout(t, func() bool {
		for _, p := range peersSet {
			if len(p.g.PeersOfChannel(common.ChainID("testchainid"))) != bootstrapSetSize+standardPeerSetSize-1 {
				log.Logger.Error("Peer discovery has not finished yet")
				return false
			}
		}
		log.Logger.Info("All peer discovered each other!!!")
		return true
	}, 120*time.Second)

	log.Logger.Debug("Waiting for all blocks to arrive.")
	waitUntilTrueOrTimeout(t, func() bool {
		log.Logger.Debug("Trying to see all authorized peers get all blocks, and all non-authorized didn't")
		for _, p := range peersSet {
			height, err := p.commit.LedgerHeight()
			id := fmt.Sprintf("localhost:%d", p.port)
			if _, isAuthorized := authorizedPeers[id]; isAuthorized {
				if height != uint64(msgCount+1) || err != nil {
					return false
				}
			} else {
				if err == nil && height > 1 {
					assert.Fail(t, "Peer", id, "got message but isn't authorized! Height:", height)
				}
			}
		}
		log.Logger.Info("All peers have same ledger height!!!")
		return true
	}, 120*time.Second)
}

func TestPeerGossip_SendingManyMessages(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	testPath := "/tmp/rongzer/test/gossipstate1"
	viper.Set("peer.fileSystemPath", testPath)
	// 账本管理器初始化
	func() {
		os.RemoveAll(testPath)
		ledgermgmt.Initialize()
	}()
	defer func() {
		ledgermgmt.Close()
		os.RemoveAll(testPath)
	}()

	bootstrapSetSize := 1
	bootstrapSet := make([]*peerNode, 0)

	for i := 0; i < bootstrapSetSize; i++ {
		commit := newCommitter(i)
		bootstrapSet = append(bootstrapSet, newPeerNode(newGossipConfig(i+30), commit, noopPeerIdentityAcceptor))
	}

	defer func() {
		for _, p := range bootstrapSet {
			p.shutdown()
		}
	}()

	msgCount := 3

	// 创建一个"模拟块"生成器
	bg, _ := testhelper.NewBlockGenerator(t, strconv.Itoa(0), false)

	for i := 1; i <= msgCount; i++ {
		block := bg.NextBlock([][]byte{})
		if b, err := pb.Marshal(block); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootstrapSet[0].s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	standartPeersSize := 2
	peersSet := make([]*peerNode, 0)

	for i := 0; i < standartPeersSize; i++ {
		commit := newCommitter(bootstrapSetSize + i)
		peersSet = append(peersSet, newPeerNode(newGossipConfig(40+i, 30), commit, noopPeerIdentityAcceptor))
	}

	defer func() {
		for _, p := range peersSet {
			p.shutdown()
		}
	}()

	waitUntilTrueOrTimeout(t, func() bool {
		for _, p := range peersSet {
			if len(p.g.PeersOfChannel(common.ChainID("testchainid"))) != bootstrapSetSize+standartPeersSize-1 {
				log.Logger.Debug("Peer discovery has not finished yet")
				return false
			}
		}
		log.Logger.Debug("All peer discovered each other!!!")
		return true
	}, 120*time.Second)

	log.Logger.Debug("Waiting for all blocks to arrive.")
	waitUntilTrueOrTimeout(t, func() bool {
		log.Logger.Debug("Trying to see all peers get all blocks")
		for _, p := range peersSet {
			height, err := p.commit.LedgerHeight()
			if height != uint64(msgCount+1) || err != nil {
				return false
			}
		}
		log.Logger.Debug("All peers have same ledger height!!!")
		return true
	}, 120*time.Second)
}

// Start one bootstrap peer and submit defAntiEntropyBatchSize + 5 messages into
// local ledger, next spawning a new peer waiting for anti-entropy procedure to
// complete missing blocks. Since state transfer messages now batched, it is expected
// to see _exactly_ two messages with state transfer response.
func TestPeerGossip_StateRequest(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	testPath := "/tmp/rongzer/test/gossipstate2"
	viper.Set("peer.fileSystemPath", testPath)
	// 账本管理器初始化
	func() {
		os.RemoveAll(testPath)
		ledgermgmt.Initialize()
	}()
	defer func() {
		ledgermgmt.Close()
		os.RemoveAll(testPath)
	}()

	bootPeer := newPeerNode(newGossipConfig(50), newCommitter(0), noopPeerIdentityAcceptor)
	defer bootPeer.shutdown()

	msgCount := 3
	expectedMessagesCnt := 1

	// 创建一个"模拟块"生成器
	bg, _ := testhelper.NewBlockGenerator(t, strconv.Itoa(0), false)

	for i := 1; i <= msgCount; i++ {
		rawblock := bg.NextBlock([][]byte{})
		if b, err := pb.Marshal(rawblock); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootPeer.s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	peer := newPeerNode(newGossipConfig(51, 50), newCommitter(1), noopPeerIdentityAcceptor)
	defer peer.shutdown()

	naiveStateMsgPredicate := func(message interface{}) bool {
		return message.(proto.ReceivedMessage).GetGossipMessage().IsRemoteStateMessage()
	}
	_, peerCh := peer.g.Accept(naiveStateMsgPredicate, true)

	messageCh := make(chan struct{})
	stopWaiting := make(chan struct{})

	// Number of submitted messages is defAntiEntropyBatchSize + 5, therefore
	// expected number of batches is expectedMessagesCnt = 2. Following go routine
	// makes sure it receives expected amount of messages and sends signal of success
	// to continue the test
	go func(expected int) {
		cnt := 0
		for cnt < expected {
			select {
			case <-peerCh:
				{
					cnt++
				}

			case <-stopWaiting:
				{
					return
				}
			}
		}

		messageCh <- struct{}{}
	}(expectedMessagesCnt)

	// Waits for message which indicates that expected number of message batches received
	// otherwise timeouts after 2 * defAntiEntropyInterval + 1 seconds
	select {
	case <-messageCh:
		{
			// Once we got message which indicate of two batches being received,
			// making sure messages indeed committed.
			waitUntilTrueOrTimeout(t, func() bool {
				if len(peer.g.PeersOfChannel(common.ChainID("testchainid"))) != 1 {
					log.Logger.Debug("Peer discovery has not finished yet")
					return false
				}
				log.Logger.Debug("All peer discovered each other!!!")
				return true
			}, 120*time.Second)

			log.Logger.Debug("Waiting for all blocks to arrive.")
			waitUntilTrueOrTimeout(t, func() bool {
				log.Logger.Debug("Trying to see all peers get all blocks")
				height, err := peer.commit.LedgerHeight()
				if height != uint64(msgCount+1) || err != nil {
					return false
				}
				log.Logger.Debug("All peers have same ledger height!!!")
				return true
			}, 120*time.Second)
		}
	case <-time.After(120 * time.Second):
		{
			close(stopWaiting)
			t.Fatal("Expected to receive two batches with missing payloads")
		}
	}
}

func TestPeerGossip_LightWeightRequest(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	testPath := "/tmp/rongzer/test/gossipstate3"
	viper.Set("peer.fileSystemPath", testPath)
	// 账本管理器初始化
	func() {
		os.RemoveAll(testPath)
		ledgermgmt.Initialize()
	}()
	defer func() {
		ledgermgmt.Close()
		os.RemoveAll(testPath)
	}()

	bootPeer := newPeerNode(newGossipConfig(60), newCommitter(0), noopPeerIdentityAcceptor)
	defer bootPeer.shutdown()

	msgCount := 3
	expectedMessagesCnt := 1
	// 创建一个"模拟块"生成器
	bg, _ := testhelper.NewBlockGenerator(t, strconv.Itoa(0), false)

	for i := 1; i <= msgCount; i++ {
		block := bg.NextBlockWithSpecEnvBytes(newEnvBytes())
		if b, err := pb.Marshal(block); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootPeer.s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}
	// light节点去同步bootPeer
	peer := newLightNode(newGossipConfig(61, 60), newCommitter(1), noopPeerIdentityAcceptor, "www")
	defer peer.shutdown()

	naiveStateMsgPredicate := func(message interface{}) bool {
		return message.(proto.ReceivedMessage).GetGossipMessage().IsRemoteStateMessage()
	}
	_, peerCh := peer.g.Accept(naiveStateMsgPredicate, true)

	messageCh := make(chan struct{})
	stopWaiting := make(chan struct{})

	// Number of submitted messages is defAntiEntropyBatchSize + 5, therefore
	// expected number of batches is expectedMessagesCnt = 2. Following go routine
	// makes sure it receives expected amount of messages and sends signal of success
	// to continue the test
	go func(expected int) {
		cnt := 0
		for cnt < expected {
			select {
			case <-peerCh:
				{
					cnt++
				}

			case <-stopWaiting:
				{
					return
				}
			}
		}

		messageCh <- struct{}{}
	}(expectedMessagesCnt)

	// Waits for message which indicates that expected number of message batches received
	// otherwise timeouts after 2 * defAntiEntropyInterval + 1 seconds
	select {
	case <-messageCh:
		{
			// Once we got message which indicate of two batches being received,
			// making sure messages indeed committed.
			waitUntilTrueOrTimeout(t, func() bool {
				if len(peer.g.PeersOfChannel(common.ChainID("testchainid"))) != 1 {
					log.Logger.Debug("Peer discovery has not finished yet")
					return false
				}
				log.Logger.Debug("All peer discovered each other!!!")
				return true
			}, 120*time.Second)

			log.Logger.Debug("Waiting for all blocks to arrive.")
			waitUntilTrueOrTimeout(t, func() bool {
				log.Logger.Debug("Trying to see all peers get all blocks")
				height, err := peer.commit.LedgerHeight()
				if height != uint64(msgCount+1) || err != nil {
					return false
				}
				log.Logger.Debug("All peers have same ledger height!!!")
				return true
			}, 120*time.Second)
		}
	case <-time.After(120 * time.Second):
		{
			close(stopWaiting)
			t.Fatal("Expected to receive two batches with missing payloads")
		}
	}
}

func TestPeerGossip_FailedLightWeightRequest(t *testing.T) {
	// 读取core.yaml配置文件
	testhelper.SetupCoreYAMLConfig()
	testPath := "/tmp/rongzer/test/gossipstate4"
	viper.Set("peer.fileSystemPath", testPath)
	// 账本管理器初始化
	func() {
		os.RemoveAll(testPath)
		ledgermgmt.Initialize()
	}()
	defer func() {
		ledgermgmt.Close()
		os.RemoveAll(testPath)
	}()

	bootPeer := newPeerNode(newGossipConfig(70), newCommitter(0), noopPeerIdentityAcceptor)
	defer bootPeer.shutdown()

	msgCount := 2
	// 创建一个"模拟块"生成器
	bg, _ := testhelper.NewBlockGenerator(t, strconv.Itoa(0), false)

	for i := 1; i <= msgCount; i++ {
		block := bg.NextBlockWithSpecEnvBytes(newEnvBytes())
		if b, err := pb.Marshal(block); err == nil {
			payload := &proto.Payload{
				SeqNum: uint64(i),
				Data:   b,
			}
			bootPeer.s.AddPayload(payload)
		} else {
			t.Fail()
		}
	}

	// light节点去同步bootPeer
	peer := newLightNode(newGossipConfig(71, 70), newCommitter(1), noopPeerIdentityAcceptor, "zzz")
	defer peer.shutdown()

	waitUntilTrueOrTimeout(t, func() bool {
		if len(peer.g.PeersOfChannel(common.ChainID("testchainid"))) != 1 {
			log.Logger.Debug("Peer discovery has not finished yet")
			return false
		}
		log.Logger.Debug("All peer discovered each other!!!")
		return true
	}, 120*time.Second)

	log.Logger.Debug("Waiting for all blocks to arrive.")
	time.Sleep(30 * time.Second)
}

func waitUntilTrueOrTimeout(t *testing.T, predicate func() bool, timeout time.Duration) {
	ch := make(chan struct{})
	go func() {
		log.Logger.Debug("Started to spin off, until predicate will be satisfied.")
		for !predicate() {
			time.Sleep(1 * time.Second)
		}
		ch <- struct{}{}
		log.Logger.Debug("Done.")
	}()

	select {
	case <-ch:
		break
	case <-time.After(timeout):
		t.Fatal("Timeout has expired")
		break
	}
	log.Logger.Debug("Stop waiting until timeout or true")
}
