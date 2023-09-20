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

package comm

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/rongzer/blockchain/common/log"
	util2 "github.com/rongzer/blockchain/common/util"
	common2 "github.com/rongzer/blockchain/peer/gossip/common"
	util3 "github.com/rongzer/blockchain/peer/gossip/util"
	proto "github.com/rongzer/blockchain/protos/gossip"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type handler func(message *proto.SignedGossipMessage)

type connFactory interface {
	createConnection(endpoint string, pkiID common2.PKIidType) (*connection, error)
}

type connectionStore struct {
	isClosing        bool                     // whether this connection store is shutting down
	connFactory      connFactory              // creates a connection to remote peer
	sync.RWMutex                              // synchronize access to shared variables
	pki2Conn         map[string]*connection   // mapping between pkiID to connections
	destinationLocks map[string]*sync.RWMutex //mapping between pkiIDs and locks,
	// used to prevent concurrent connection establishment to the same remote endpoint
}

func newConnStore(connFactory connFactory) *connectionStore {
	return &connectionStore{
		connFactory:      connFactory,
		isClosing:        false,
		pki2Conn:         make(map[string]*connection),
		destinationLocks: make(map[string]*sync.RWMutex),
	}
}

func (cs *connectionStore) getConnection(peer *RemotePeer) (*connection, error) {
	cs.RLock()
	isClosing := cs.isClosing
	cs.RUnlock()

	if isClosing {
		return nil, errors.New("Shutting down")
	}

	pkiID := peer.PKIID
	endpoint := peer.Endpoint

	cs.Lock()
	destinationLock, hasConnected := cs.destinationLocks[util2.BytesToString(pkiID)]
	if !hasConnected {
		destinationLock = &sync.RWMutex{}
		cs.destinationLocks[util2.BytesToString(pkiID)] = destinationLock
	}
	cs.Unlock()

	destinationLock.Lock()

	cs.RLock()
	conn, exists := cs.pki2Conn[util2.BytesToString(pkiID)]
	if exists {
		cs.RUnlock()
		destinationLock.Unlock()
		return conn, nil
	}
	cs.RUnlock()

	createdConnection, err := cs.connFactory.createConnection(endpoint, pkiID)

	destinationLock.Unlock()

	cs.RLock()
	isClosing = cs.isClosing
	cs.RUnlock()
	if isClosing {
		return nil, errors.New("ConnStore is closing")
	}

	cs.Lock()
	delete(cs.destinationLocks, util2.BytesToString(pkiID))
	defer cs.Unlock()

	// check again, maybe someone connected to us during the connection creation?
	conn, exists = cs.pki2Conn[util2.BytesToString(pkiID)]

	if exists {
		if createdConnection != nil {
			createdConnection.close()
		}
		return conn, nil
	}

	// no one connected to us AND we failed connecting!
	if err != nil {
		return nil, err
	}

	// at this point in the code, we created a connection to a remote peer
	conn = createdConnection
	cs.pki2Conn[util2.BytesToString(createdConnection.pkiID)] = conn

	go conn.serviceConnection()

	return conn, nil
}

func (cs *connectionStore) connNum() int {
	cs.RLock()
	defer cs.RUnlock()
	return len(cs.pki2Conn)
}

func (cs *connectionStore) closeConn(peer *RemotePeer) {
	cs.Lock()
	defer cs.Unlock()

	if conn, exists := cs.pki2Conn[util2.BytesToString(peer.PKIID)]; exists {
		conn.close()
		delete(cs.pki2Conn, util2.BytesToString(conn.pkiID))
	}
}

func (cs *connectionStore) shutdown() {
	cs.Lock()
	cs.isClosing = true
	pkiIds2conn := cs.pki2Conn

	var connections2Close []*connection
	for _, conn := range pkiIds2conn {
		connections2Close = append(connections2Close, conn)
	}
	cs.Unlock()

	wg := sync.WaitGroup{}
	for _, conn := range connections2Close {
		wg.Add(1)
		go func(conn *connection) {
			cs.closeByPKIid(conn.pkiID)
			wg.Done()
		}(conn)
	}
	wg.Wait()
}

func (cs *connectionStore) onConnected(serverStream proto.Gossip_GossipStreamServer, connInfo *proto.ConnectionInfo) *connection {
	cs.Lock()
	defer cs.Unlock()

	if c, exists := cs.pki2Conn[util2.BytesToString(connInfo.Identity)]; exists {
		c.close()
	}

	return cs.registerConn(connInfo, serverStream)
}

func (cs *connectionStore) registerConn(connInfo *proto.ConnectionInfo, serverStream proto.Gossip_GossipStreamServer) *connection {
	conn := newConnection(nil, nil, nil, serverStream)
	conn.pkiID = connInfo.ID
	conn.info = connInfo
	cs.pki2Conn[util2.BytesToString(connInfo.ID)] = conn
	return conn
}

func (cs *connectionStore) closeByPKIid(pkiID common2.PKIidType) {
	cs.Lock()
	defer cs.Unlock()
	if conn, exists := cs.pki2Conn[util2.BytesToString(pkiID)]; exists {
		conn.close()
		delete(cs.pki2Conn, util2.BytesToString(pkiID))
	}
}

func newConnection(cl proto.GossipClient, c *grpc.ClientConn, cs proto.Gossip_GossipStreamClient, ss proto.Gossip_GossipStreamServer) *connection {
	connection := &connection{
		outBuff:      make(chan *msgSending, util3.GetIntOrDefault("peer.gossip.sendBuffSize", defSendBuffSize)),
		cl:           cl,
		conn:         c,
		clientStream: cs,
		serverStream: ss,
		stopFlag:     int32(0),
		stopChan:     make(chan struct{}, 1),
	}

	return connection
}

type connection struct {
	cancel       context.CancelFunc
	info         *proto.ConnectionInfo
	outBuff      chan *msgSending
	pkiID        common2.PKIidType               // pkiID of the remote endpoint
	handler      handler                         // function to invoke upon a message reception
	conn         *grpc.ClientConn                // gRPC connection to remote endpoint
	cl           proto.GossipClient              // gRPC stub of remote endpoint
	clientStream proto.Gossip_GossipStreamClient // client-side stream to remote endpoint
	serverStream proto.Gossip_GossipStreamServer // server-side stream to remote endpoint
	stopFlag     int32                           // indicates whether this connection is in process of stopping
	stopChan     chan struct{}                   // a method to stop the server-side gRPC call from a different go-routine
	sync.RWMutex                                 // synchronizes access to shared variables
}

func (conn *connection) close() {
	if conn.toDie() {
		return
	}

	amIFirst := atomic.CompareAndSwapInt32(&conn.stopFlag, int32(0), int32(1))
	if !amIFirst {
		return
	}

	conn.stopChan <- struct{}{}

	conn.Lock()

	if conn.clientStream != nil {
		conn.clientStream.CloseSend()
	}
	if conn.conn != nil {
		conn.conn.Close()
	}

	if conn.cancel != nil {
		conn.cancel()
	}

	conn.Unlock()

}

func (conn *connection) toDie() bool {
	return atomic.LoadInt32(&(conn.stopFlag)) == int32(1)
}

func (conn *connection) send(msg *proto.SignedGossipMessage, onErr func(error)) {
	conn.Lock()
	defer conn.Unlock()

	if len(conn.outBuff) == util3.GetIntOrDefault("peer.gossip.sendBuffSize", defSendBuffSize) {
		return
	}

	m := &msgSending{
		envelope: msg.Envelope,
		onErr:    onErr,
	}

	conn.outBuff <- m
}

func (conn *connection) serviceConnection() error {
	errChan := make(chan error, 1)
	msgChan := make(chan *proto.SignedGossipMessage, util3.GetIntOrDefault("peer.gossip.recvBuffSize", defRecvBuffSize))
	defer close(msgChan)

	// Call stream.Recv() asynchronously in readFromStream(),
	// and wait for either the Recv() call to end,
	// or a signal to close the connection, which exits
	// the method and makes the Recv() call to fail in the
	// readFromStream() method
	go conn.readFromStream(errChan, msgChan)

	go conn.writeToStream()

	for !conn.toDie() {
		select {
		case stop := <-conn.stopChan:
			log.Logger.Warn("Closing reading from stream")
			conn.stopChan <- stop
			return nil
		case err := <-errChan:
			log.Logger.Errorf("errChan : %s", err)
			return err
		case msg := <-msgChan:
			conn.handler(msg)
		}
	}
	return nil
}

func (conn *connection) writeToStream() {
	for !conn.toDie() {
		stream := conn.getStream()
		if stream == nil {
			log.Logger.Error(conn.pkiID, " writeToStream aborting because Stream is nil")
			return
		}
		select {
		case m := <-conn.outBuff:
			err := stream.Send(m.envelope)
			if err != nil {
				go m.onErr(err)
				log.Logger.Errorf("conn err msg failed write back : %s connectin num : %d", err)
				return
			}
		case stop := <-conn.stopChan:
			log.Logger.Warn("Closing writing to stream")
			conn.stopChan <- stop
			return
		}
	}
}

func (conn *connection) readFromStream(errChan chan error, msgChan chan *proto.SignedGossipMessage) {
	defer func() {
		recover()
	}() // msgChan might be closed
	for !conn.toDie() {
		stream := conn.getStream()
		if stream == nil {
			log.Logger.Error(conn.pkiID, " readFromStream aborting because Stream is nil")
			errChan <- errors.New("Stream is nil")
			return
		}
		envelope, err := stream.Recv()
		if conn.toDie() {
			log.Logger.Warn(conn.pkiID, " readFromStream canceling read because closing")
			return
		}
		if err != nil {
			errChan <- err
			log.Logger.Error(conn.pkiID, " readFromStream Recv error, aborting:", err)
			return
		}
		msg, err := envelope.ToGossipMessage()
		if err != nil {
			errChan <- err
			log.Logger.Warn(conn.pkiID, " readFromStream ToGossipMessage error, aborting:", err)
		}
		msgChan <- msg
	}
}

func (conn *connection) getStream() stream {
	conn.Lock()
	defer conn.Unlock()

	if conn.clientStream != nil && conn.serverStream != nil {
		log.Logger.Error("Both client and server stream are not nil, something went wrong")
	}

	if conn.clientStream != nil {
		return conn.clientStream
	}

	if conn.serverStream != nil {
		return conn.serverStream
	}

	return nil
}

type msgSending struct {
	envelope *proto.Envelope
	onErr    func(error)
}
