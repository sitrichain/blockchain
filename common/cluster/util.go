package cluster

import (
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/protos/common"
	"google.golang.org/grpc"
	"sync"
	"sync/atomic"
)

// ConnByCertMap maps certificates represented as strings
// to gRPC connections
type ConnByCertMap map[string]*grpc.ClientConn

// Lookup looks up a certificate and returns the connection that was mapped
// to the certificate, and whether it was found or not
func (cbc ConnByCertMap) Lookup(endPoint string) (*grpc.ClientConn, bool) {
	conn, ok := cbc[endPoint]
	return conn, ok
}

// Put associates the given connection to the certificate
func (cbc ConnByCertMap) Put(endPoint string, conn *grpc.ClientConn) {
	cbc[endPoint] = conn
}

// Remove removes the connection that is associated to the given certificate
func (cbc ConnByCertMap) Remove(endPoint string) {
	delete(cbc, endPoint)
}

func (cbc ConnByCertMap) Size() int {
	return len(cbc)
}

// MemberMapping defines NetworkMembers by their ID
type MemberMapping map[uint64]*Stub

// Put inserts the given stub to the MemberMapping
func (mp MemberMapping) Put(stub *Stub) {
	mp[stub.ID] = stub
}

// ByID retrieves the Stub with the given ID from the MemberMapping
func (mp MemberMapping) ByID(ID uint64) *Stub {
	return mp[ID]
}

// LookupByClientCert retrieves a Stub with the given endpoint
func (mp MemberMapping) LookupByEndPoint(endpoint string) *Stub {
	for _, stub := range mp {
		if stub.Endpoint == endpoint {
			return stub
		}
	}
	return nil
}

// ServerCertificates returns a set of the server certificates
// represented as strings
func (mp MemberMapping) EndPoints() StringSet {
	res := make(StringSet)
	for _, member := range mp {
		res[string(member.Endpoint)] = struct{}{}
	}
	return res
}

// StringSet is a set of strings
type StringSet map[string]struct{}

// union adds the elements of the given set to the StringSet
func (ss StringSet) union(set StringSet) {
	for k := range set {
		ss[k] = struct{}{}
	}
}

// subtract removes all elements in the given set from the StringSet
func (ss StringSet) subtract(set StringSet) {
	for k := range set {
		delete(ss, k)
	}
}

// PredicateDialer creates gRPC connections
// that are only established if the given predicate
// is fulfilled
type PredicateDialer struct {
	lock   sync.RWMutex
	Config comm.ClientConfig
}

// Dial creates a new gRPC connection that can only be established, if the remote node's
// certificate chain satisfy verifyFunc
func (dialer *PredicateDialer) Dial(address string) (*grpc.ClientConn, error) {
	dialer.lock.RLock()
	cfg := dialer.Config.Clone()
	dialer.lock.RUnlock()

	client, err := comm.NewGRPCClient(cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return client.NewConnection(address)
}

// StandardDialer wraps an ClientConfig, and provides
// a means to connect according to given EndpointCriteria.
type StandardDialer struct {
	Config comm.ClientConfig
}

// Dial dials an address according to the given EndpointCriteria
func (dialer *StandardDialer) Dial(endpointCriteria EndpointCriteria) (*grpc.ClientConn, error) {
	cfg := dialer.Config.Clone()

	client, err := comm.NewGRPCClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating gRPC client")
	}

	return client.NewConnection(endpointCriteria.Endpoint)
}

// Dialer creates a gRPC connection to a remote address
type Dialer interface {
	Dial(endpointCriteria EndpointCriteria) (*grpc.ClientConn, error)
}

var errNotAConfig = errors.New("not a config block")

// EndpointCriteria defines criteria of how to connect to a remote orderer node.
type EndpointCriteria struct {
	Endpoint string // Endpoint of the form host:port
}

// String returns a string representation of this EndpointCriteria
func (ep EndpointCriteria) String() string {
	return ep.Endpoint
}

// BlockCommitFunc signals a block commit.
type BlockCommitFunc func(block *common.Block, channel string)


// BlockRetriever retrieves blocks
type BlockRetriever interface {
	// Block returns a block with the given number,
	// or nil if such a block doesn't exist.
	GetBlockByNumber(blockNumber uint64) (*common.Block, error)
}

// StreamCountReporter reports the number of streams currently connected to this node
type StreamCountReporter struct {
	count   uint32
}

func (scr *StreamCountReporter) Increment() {
	atomic.AddUint32(&scr.count, 1)
}

func (scr *StreamCountReporter) Decrement() {
	atomic.AddUint32(&scr.count, ^uint32(0))
}
