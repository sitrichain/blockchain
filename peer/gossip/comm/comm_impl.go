package comm

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rongzer/blockchain/common/log"
	api2 "github.com/rongzer/blockchain/peer/gossip/api"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	identity2 "github.com/rongzer/blockchain/peer/gossip/identity"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
	proto "github.com/rongzer/blockchain/protos/gossip"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const (
	defDialTimeout  = time.Second * time.Duration(3)
	defConnTimeout  = time.Second * time.Duration(2)
	defRecvBuffSize = 20
	defSendBuffSize = 20
)

func (c *commImpl) SetDialOpts(opts ...grpc.DialOption) {
	if len(opts) == 0 {
		log.Logger.Warn("Given an empty set of grpc.DialOption, aborting")
		return
	}
	c.opts = opts
}

// NewCommInstanceWithServer creates a comm instance that creates an underlying gRPC server
func NewCommInstanceWithServer(port int, idMapper identity2.Mapper, peerIdentity api2.PeerIdentityType,
	secureDialOpts api2.PeerSecureDialOpts, dialOpts ...grpc.DialOption) (Comm, error) {

	var ll net.Listener
	var s *grpc.Server
	var certHash []byte

	if len(dialOpts) == 0 {
		dialOpts = []grpc.DialOption{grpc.WithTimeout(util2.GetDurationOrDefault("core.peer.gossip.dialTimeout", defDialTimeout))}
	}

	if port > 0 {
		s, ll, secureDialOpts, certHash = createGRPCLayer(port)
	}

	commInst := &commImpl{
		selfCertHash:   certHash,
		PKIID:          idMapper.GetPKIidOfCert(peerIdentity),
		idMapper:       idMapper,
		peerIdentity:   peerIdentity,
		opts:           dialOpts,
		secureDialOpts: secureDialOpts,
		port:           port,
		lsnr:           ll,
		gSrv:           s,
		msgPublisher:   NewChannelDemultiplexer(),
		lock:           &sync.RWMutex{},
		deadEndpoints:  make(chan common2.PKIidType, 100),
		stopping:       int32(0),
		exitChan:       make(chan struct{}, 1),
		subscriptions:  make([]chan proto.ReceivedMessage, 0),
	}
	commInst.connStore = newConnStore(commInst)

	if port > 0 {
		commInst.stopWG.Add(1)
		go func() {
			defer commInst.stopWG.Done()
			s.Serve(ll)
		}()
		proto.RegisterGossipServer(s, commInst)
	}

	return commInst, nil
}

// NewCommInstance creates a new comm instance that binds itself to the given gRPC server
func NewCommInstance(s *grpc.Server, cert *tls.Certificate, idStore identity2.Mapper,
	peerIdentity api2.PeerIdentityType, secureDialOpts api2.PeerSecureDialOpts,
	dialOpts ...grpc.DialOption) (Comm, error) {

	dialOpts = append(dialOpts, grpc.WithTimeout(util2.GetDurationOrDefault("core.peer.gossip.dialTimeout", defDialTimeout)))
	commInst, err := NewCommInstanceWithServer(-1, idStore, peerIdentity, secureDialOpts, dialOpts...)
	if err != nil {
		return nil, err
	}

	if cert != nil {
		inst := commInst.(*commImpl)
		if len(cert.Certificate) == 0 {
			log.Logger.Panic("Certificate supplied but certificate chain is empty")
		} else {
			inst.selfCertHash = certHashFromRawCert(cert.Certificate[0])
		}
	}

	proto.RegisterGossipServer(s, commInst.(*commImpl))

	return commInst, nil
}

type commImpl struct {
	selfCertHash   []byte
	peerIdentity   api2.PeerIdentityType
	idMapper       identity2.Mapper
	opts           []grpc.DialOption
	secureDialOpts func() []grpc.DialOption
	connStore      *connectionStore
	PKIID          []byte
	deadEndpoints  chan common2.PKIidType
	msgPublisher   *ChannelDeMultiplexer
	lock           *sync.RWMutex
	lsnr           net.Listener
	gSrv           *grpc.Server
	exitChan       chan struct{}
	stopWG         sync.WaitGroup
	subscriptions  []chan proto.ReceivedMessage
	port           int
	stopping       int32
}

func (c *commImpl) createConnection(endpoint string, expectedPKIID common2.PKIidType) (*connection, error) {
	var err error
	var cc *grpc.ClientConn
	var stream proto.Gossip_GossipStreamClient
	var pkiID common2.PKIidType
	var connInfo *proto.ConnectionInfo
	var dialOpts []grpc.DialOption

	log.Logger.Debug("Entering", endpoint, expectedPKIID)
	defer log.Logger.Debug("Exiting")

	if c.isStopping() {
		return nil, errors.New("Stopping")
	}
	dialOpts = append(dialOpts, c.secureDialOpts()...)
	dialOpts = append(dialOpts, grpc.WithBlock())
	dialOpts = append(dialOpts, c.opts...)

	cc, err = grpc.Dial(endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}

	cl := proto.NewGossipClient(cc)

	if _, err = cl.Ping(context.Background(), &proto.Empty{}); err != nil {
		cc.Close()
		return nil, err
	}

	ctx, cf := context.WithCancel(context.Background())
	if stream, err = cl.GossipStream(ctx); err == nil {
		connInfo, err = c.authenticateRemotePeer(stream)
		if err == nil {
			pkiID = connInfo.ID
			if expectedPKIID != nil && !bytes.Equal(pkiID, expectedPKIID) {
				// PKIID is nil when we don't know the remote PKI id's
				log.Logger.Warn("Remote endpoint claims to be a different peer, expected ", expectedPKIID, " but got ", pkiID)
				cc.Close()
				return nil, errors.New("Authentication failure")
			}
			conn := newConnection(cl, cc, stream, nil)
			conn.info = connInfo
			conn.cancel = cf

			h := func(m *proto.SignedGossipMessage) {
				log.Logger.Debug("Got message:", m)
				c.msgPublisher.DeMultiplex(&ReceivedMessageImpl{
					conn:                conn,
					lock:                conn,
					SignedGossipMessage: m,
					connInfo:            connInfo,
				})
			}
			conn.handler = h
			return conn, nil
		}
		log.Logger.Warn("Authentication failed:", err)
	}
	cc.Close()
	return nil, err
}

func (c *commImpl) Send(msg *proto.SignedGossipMessage, peers ...*RemotePeer) {
	if c.isStopping() || len(peers) == 0 {
		return
	}

	log.Logger.Debug("Entering, sending", msg, "to ", len(peers), "peers")

	for _, p := range peers {
		go func(p *RemotePeer, msg *proto.SignedGossipMessage) {
			c.sendToEndpoint(p, msg)
		}(p, msg)
	}
}

func (c *commImpl) sendToEndpoint(peer *RemotePeer, msg *proto.SignedGossipMessage) {
	if c.isStopping() {
		return
	}
	log.Logger.Debug("Entering, Sending to", peer.Endpoint, ", msg:", msg)
	defer log.Logger.Debug("Exiting")
	var err error

	conn, err := c.connStore.getConnection(peer)
	if err == nil {
		disConnectOnErr := func(err error) {
			log.Logger.Warn(peer, " isn't responsive: ", err)
			c.disconnect(peer.PKIID)
		}
		conn.send(msg, disConnectOnErr)
		return
	}
	log.Logger.Warn("Failed obtaining connection for ", peer, " reason: ", err)
	c.disconnect(peer.PKIID)
}

func (c *commImpl) isStopping() bool {
	return atomic.LoadInt32(&c.stopping) == int32(1)
}

func (c *commImpl) Probe(remotePeer *RemotePeer) error {
	var dialOpts []grpc.DialOption
	endpoint := remotePeer.Endpoint
	pkiID := remotePeer.PKIID
	if c.isStopping() {
		return errors.New("Stopping")
	}
	log.Logger.Debug("Entering, endpoint: ", endpoint, " PKIID:", pkiID)
	dialOpts = append(dialOpts, c.secureDialOpts()...)
	dialOpts = append(dialOpts, grpc.WithBlock())
	dialOpts = append(dialOpts, c.opts...)

	cc, err := grpc.Dial(remotePeer.Endpoint, dialOpts...)
	if err != nil {
		log.Logger.Debug("Returning", err)
		return err
	}
	defer cc.Close()
	cl := proto.NewGossipClient(cc)
	_, err = cl.Ping(context.Background(), &proto.Empty{})
	log.Logger.Debug("Returning", err)
	return err
}

func (c *commImpl) Handshake(remotePeer *RemotePeer) (api2.PeerIdentityType, error) {
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, c.secureDialOpts()...)
	dialOpts = append(dialOpts, grpc.WithBlock())
	dialOpts = append(dialOpts, c.opts...)

	cc, err := grpc.Dial(remotePeer.Endpoint, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer cc.Close()

	cl := proto.NewGossipClient(cc)
	if _, err = cl.Ping(context.Background(), &proto.Empty{}); err != nil {
		return nil, err
	}

	stream, err := cl.GossipStream(context.Background())
	if err != nil {
		return nil, err
	}
	connInfo, err := c.authenticateRemotePeer(stream)
	if err != nil {
		log.Logger.Warn("Authentication failed:", err)
		return nil, err
	}
	if len(remotePeer.PKIID) > 0 && !bytes.Equal(connInfo.ID, remotePeer.PKIID) {
		return nil, errors.New("PKI-ID of remote peer doesn't match expected PKI-ID")
	}
	return connInfo.Identity, nil
}

func (c *commImpl) Accept(acceptor common2.MessageAcceptor) <-chan proto.ReceivedMessage {
	genericChan := c.msgPublisher.AddChannel(acceptor)
	specificChan := make(chan proto.ReceivedMessage, 10)

	if c.isStopping() {
		log.Logger.Warn("Accept() called but comm module is stopping, returning empty channel")
		return specificChan
	}

	c.lock.Lock()
	c.subscriptions = append(c.subscriptions, specificChan)
	c.lock.Unlock()

	go func() {
		defer log.Logger.Debug("Exiting Accept() loop")
		defer func() {
			recover()
		}()

		c.stopWG.Add(1)
		defer c.stopWG.Done()

		for {
			select {
			case msg := <-genericChan:
				specificChan <- msg.(*ReceivedMessageImpl)
			case s := <-c.exitChan:
				c.exitChan <- s
				return
			}
		}
	}()
	return specificChan
}

func (c *commImpl) PresumedDead() <-chan common2.PKIidType {
	return c.deadEndpoints
}

func (c *commImpl) CloseConn(peer *RemotePeer) {
	log.Logger.Debug("Closing connection for", peer)
	c.connStore.closeConn(peer)
}

func (c *commImpl) emptySubscriptions() {
	c.lock.Lock()
	defer c.lock.Unlock()
	for _, ch := range c.subscriptions {
		close(ch)
	}
}

func (c *commImpl) Stop() {
	if c.isStopping() {
		return
	}
	atomic.StoreInt32(&c.stopping, int32(1))
	log.Logger.Info("Stopping")
	defer log.Logger.Info("Stopped")
	if c.gSrv != nil {
		c.gSrv.Stop()
	}
	if c.lsnr != nil {
		c.lsnr.Close()
	}
	c.connStore.shutdown()
	log.Logger.Debug("Shut down connection store, connection count:", c.connStore.connNum())
	c.exitChan <- struct{}{}
	c.msgPublisher.Close()
	log.Logger.Debug("Shut down publisher")
	c.emptySubscriptions()
	log.Logger.Debug("Closed subscriptions, waiting for goroutines to stop...")
	c.stopWG.Wait()
}

func (c *commImpl) GetPKIid() common2.PKIidType {
	return c.PKIID
}

func extractRemoteAddress(stream stream) string {
	var remoteAddress string
	p, ok := peer.FromContext(stream.Context())
	if ok {
		if address := p.Addr; address != nil {
			remoteAddress = address.String()
		}
	}
	return remoteAddress
}

func (c *commImpl) authenticateRemotePeer(stream stream) (*proto.ConnectionInfo, error) {
	ctx := stream.Context()
	remoteAddress := extractRemoteAddress(stream)
	remoteCertHash := extractCertificateHashFromContext(ctx)
	var err error
	var cMsg *proto.SignedGossipMessage
	var signer proto.Signer
	useTLS := c.selfCertHash != nil

	// If TLS is enabled, sign the connection message in order to bind
	// the TLS session to the peer's identity
	if useTLS {
		signer = func(msg []byte) ([]byte, error) {
			return c.idMapper.Sign(msg)
		}
	} else { // If we don't use TLS, we have no unique text to sign,
		//  so don't sign anything
		signer = func(msg []byte) ([]byte, error) {
			return msg, nil
		}
	}

	// TLS enabled but not detected on other side
	if useTLS && len(remoteCertHash) == 0 {
		log.Logger.Warnf("%s didn't send TLS certificate", remoteAddress)
		return nil, errors.New("No TLS certificate")
	}

	cMsg, err = c.createConnectionMsg(c.PKIID, c.selfCertHash, c.peerIdentity, signer)
	if err != nil {
		return nil, err
	}

	log.Logger.Debug("Sending", cMsg, "to", remoteAddress)
	stream.Send(cMsg.Envelope)
	m, err := readWithTimeout(stream, util2.GetDurationOrDefault("core.peer.gossip.connTimeout", defConnTimeout), remoteAddress)
	if err != nil {
		log.Logger.Warnf("Failed reading messge from %s, reason: %v", remoteAddress, err)
		return nil, err
	}
	receivedMsg := m.GetConn()
	if receivedMsg == nil {
		log.Logger.Warn("Expected connection message from", remoteAddress, "but got", receivedMsg)
		return nil, errors.New("Wrong type")
	}

	if receivedMsg.PkiId == nil {
		log.Logger.Warn("%s didn't send a pkiID", remoteAddress)
		return nil, errors.New("No PKI-ID")
	}

	log.Logger.Debug("Received", receivedMsg, "from", remoteAddress)
	err = c.idMapper.Put(receivedMsg.PkiId, receivedMsg.Identity)
	if err != nil {
		log.Logger.Warn("Identity store rejected", remoteAddress, ":", err)
		return nil, err
	}

	connInfo := &proto.ConnectionInfo{
		ID:       receivedMsg.PkiId,
		Identity: receivedMsg.Identity,
		Endpoint: remoteAddress,
	}

	// if TLS is enabled and detected, verify remote peer
	if useTLS {
		// If the remote peer sent its TLS certificate, make sure it actually matches the TLS cert
		// that the peer used.
		if !bytes.Equal(remoteCertHash, receivedMsg.TlsCertHash) {
			return nil, fmt.Errorf("Expected %v in remote hash of TLS cert, but got %v", remoteCertHash, receivedMsg.TlsCertHash)
		}
		verifier := func(peerIdentity []byte, signature, message []byte) error {
			pkiID := c.idMapper.GetPKIidOfCert(peerIdentity)
			return c.idMapper.Verify(pkiID, signature, message)
		}
		err = m.Verify(receivedMsg.Identity, verifier)
		if err != nil {
			log.Logger.Error("Failed verifying signature from", remoteAddress, ":", err)
			return nil, err
		}
		connInfo.Auth = &proto.AuthInfo{
			Signature:  m.Signature,
			SignedData: m.Payload,
		}
	}

	log.Logger.Debug("Authenticated", remoteAddress)

	return connInfo, nil
}

func (c *commImpl) GossipStream(stream proto.Gossip_GossipStreamServer) error {
	if c.isStopping() {
		return errors.New("Shutting down")
	}
	connInfo, err := c.authenticateRemotePeer(stream)
	if err != nil {
		log.Logger.Error("Authentication failed:", err)
		return err
	}
	log.Logger.Debug("Servicing", extractRemoteAddress(stream))

	conn := c.connStore.onConnected(stream, connInfo)

	// if connStore denied the connection, it means we already have a connection to that peer
	// so close this stream
	if conn == nil {
		return nil
	}

	h := func(m *proto.SignedGossipMessage) {
		c.msgPublisher.DeMultiplex(&ReceivedMessageImpl{
			conn:                conn,
			lock:                conn,
			SignedGossipMessage: m,
			connInfo:            connInfo,
		})
	}

	conn.handler = h

	defer func() {
		log.Logger.Debug("Client", extractRemoteAddress(stream), " disconnected")
		c.connStore.closeByPKIid(connInfo.ID)
		conn.close()
	}()

	return conn.serviceConnection()
}

func (c *commImpl) Ping(context.Context, *proto.Empty) (*proto.Empty, error) {
	return &proto.Empty{}, nil
}

func (c *commImpl) disconnect(pkiID common2.PKIidType) {
	if c.isStopping() {
		return
	}
	c.deadEndpoints <- pkiID
	c.connStore.closeByPKIid(pkiID)
}

func readWithTimeout(stream interface{}, timeout time.Duration, address string) (*proto.SignedGossipMessage, error) {
	incChan := make(chan *proto.SignedGossipMessage, 1)
	errChan := make(chan error, 1)
	go func() {
		if srvStr, isServerStr := stream.(proto.Gossip_GossipStreamServer); isServerStr {
			if m, err := srvStr.Recv(); err == nil {
				msg, err := m.ToGossipMessage()
				if err != nil {
					errChan <- err
					return
				}
				incChan <- msg
			}
		} else if clStr, isClientStr := stream.(proto.Gossip_GossipStreamClient); isClientStr {
			if m, err := clStr.Recv(); err == nil {
				msg, err := m.ToGossipMessage()
				if err != nil {
					errChan <- err
					return
				}
				incChan <- msg
			}
		} else {
			panic(fmt.Errorf("Stream isn't a GossipStreamServer or a GossipStreamClient, but %v. Aborting", reflect.TypeOf(stream)))
		}
	}()
	select {
	case <-time.NewTicker(timeout).C:
		return nil, fmt.Errorf("Timed out waiting for connection message from %s", address)
	case m := <-incChan:
		return m, nil
	case err := <-errChan:
		return nil, err
	}
}

func (c *commImpl) createConnectionMsg(pkiID common2.PKIidType, certHash []byte, cert api2.PeerIdentityType, signer proto.Signer) (*proto.SignedGossipMessage, error) {
	m := &proto.GossipMessage{
		Tag:   proto.GossipMessage_EMPTY,
		Nonce: 0,
		Content: &proto.GossipMessage_Conn{
			Conn: &proto.ConnEstablish{
				TlsCertHash: certHash,
				Identity:    cert,
				PkiId:       pkiID,
			},
		},
	}
	sMsg := &proto.SignedGossipMessage{
		GossipMessage: m,
	}
	_, err := sMsg.Sign(signer)
	return sMsg, err
}

type stream interface {
	Send(envelope *proto.Envelope) error
	Recv() (*proto.Envelope, error)
	grpc.Stream
}

func createGRPCLayer(port int) (*grpc.Server, net.Listener, api2.PeerSecureDialOpts, []byte) {
	var returnedCertHash []byte
	var s *grpc.Server
	var ll net.Listener
	var err error
	var serverOpts []grpc.ServerOption
	var dialOpts []grpc.DialOption

	cert := GenerateCertificatesOrPanic()
	returnedCertHash = certHashFromRawCert(cert.Certificate[0])

	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		ClientAuth:         tls.RequestClientCert,
		InsecureSkipVerify: true,
	}
	serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConf)))
	ta := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	})
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(ta))

	listenAddress := fmt.Sprintf("%s:%d", "", port)
	ll, err = net.Listen("tcp", listenAddress)

	if err != nil {
		panic(err)
	}
	secureDialOpts := func() []grpc.DialOption {
		return dialOpts
	}
	s = grpc.NewServer(serverOpts...)
	return s, ll, secureDialOpts, returnedCertHash
}
