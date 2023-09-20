package comm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/rongzer/blockchain/common/conf"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type GRPCClient struct {
	// Options for setting up new connections
	dialOpts []grpc.DialOption
	// Duration for which to block while established a new connection
	timeout time.Duration
	// Maximum message size the client can receive
	maxRecvMsgSize int
	// Maximum message size the client can send
	maxSendMsgSize int
}

// ClientConfig defines the parameters for configuring a GRPCClient instance
type ClientConfig struct {
	// KaOpts defines the keepalive parameters
	KaOpts *KeepaliveOptions
	// Timeout specifies how long the client will block when attempting to
	// establish a connection
	Timeout time.Duration
	// AsyncConnect makes connection creation non blocking
	AsyncConnect bool
}

// Clone clones this ClientConfig
func (cc ClientConfig) Clone() ClientConfig {
	shallowClone := cc
	return shallowClone
}

// NewGRPCClient creates a new implementation of GRPCClient given an address
// and client configuration
func NewGRPCClient(config ClientConfig) (*GRPCClient, error) {
	client := &GRPCClient{}

	// keepalive options
	kap := keepalive.ClientParameters{
		Time:                config.KaOpts.ClientInterval,
		Timeout:             config.KaOpts.ClientTimeout,
		PermitWithoutStream: true,
	}
	// set keepalive
	client.dialOpts = append(client.dialOpts, grpc.WithKeepaliveParams(kap))
	// Unless asynchronous connect is set, make connection establishment blocking.
	if !config.AsyncConnect {
		client.dialOpts = append(client.dialOpts, grpc.WithBlock())
		client.dialOpts = append(client.dialOpts, grpc.FailOnNonTempDialError(true))
	}
	client.timeout = config.Timeout
	// set send/recv message size to package defaults
	client.maxRecvMsgSize = MaxRecvMsgSize()
	client.maxSendMsgSize = MaxSendMsgSize()

	return client, nil
}

// SetMaxRecvMsgSize sets the maximum message size the client can receive
func (client *GRPCClient) SetMaxRecvMsgSize(size int) {
	client.maxRecvMsgSize = size
}

// SetMaxSendMsgSize sets the maximum message size the client can send
func (client *GRPCClient) SetMaxSendMsgSize(size int) {
	client.maxSendMsgSize = size
}

type TLSOption func(tlsConfig *tls.Config)

// NewConnection returns a grpc.ClientConn for the target address and
// overrides the server name used to verify the hostname on the
// certificate returned by a server when using TLS
func (client *GRPCClient) NewConnection(address string) (*grpc.ClientConn, error) {

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, client.dialOpts...)
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(client.maxRecvMsgSize),
		grpc.MaxCallSendMsgSize(client.maxSendMsgSize),
	))
	if TLSEnabled() {
		InitMappingOfIpAndHost()
		// 构造server rootCAs的池，以验证服务端的tls证书是否由合法ca签发
		cp := x509.NewCertPool()
		for _, file := range conf.V.TLS.RootCAs {
			pem, err := ioutil.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to load Server RootCAs file '%s' %w", file, err)
			}
			if !cp.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("credentials: failed to append certificates of server rootCAs")
			}
		}
		// 构造客户端自己的x509证书（服务端和客户端共用一套tls证书和私钥）
		clientCertificate, err := ioutil.ReadFile(conf.V.TLS.Certificate)
		if err != nil {
			return nil, fmt.Errorf("failed to load Server Certificate file '%s'. %w", conf.V.TLS.Certificate, err)
		}
		clientKey, err := ioutil.ReadFile(conf.V.TLS.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load PrivateKey file '%s' %w", conf.V.TLS.PrivateKey, err)
		}
		cert, err := tls.X509KeyPair(clientCertificate, clientKey)
		if err != nil {
			return nil, err
		}
		serverName := GetHostNameFromIp(strings.Split(address, ":")[0])
		credentials := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   serverName,
			RootCAs:      cp,
		})
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, dialOpts...)
	if err != nil {
		return nil, errors.WithMessage(errors.WithStack(err),
			"failed to create new connection")
	}
	return conn, nil
}
