package comm

import (
	"github.com/rongzer/blockchain/common/log"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/conf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	// Is the configuration cached?
	configurationCached = false
	// Is TLS enabled
	tlsEnabled bool
	// Max send and receive bytes for grpc clients and servers
	maxRecvMsgSize = 100 * 1024 * 1024
	maxSendMsgSize = 100 * 1024 * 1024
	// Default keepalive options
	keepaliveOptions = &KeepaliveOptions{
		ClientInterval:    time.Duration(1) * time.Minute,  // 1 min
		ClientTimeout:     time.Duration(20) * time.Second, // 20 sec - gRPC default
		ServerInterval:    time.Duration(2) * time.Hour,    // 2 hours - gRPC default
		ServerTimeout:     time.Duration(20) * time.Second, // 20 sec - gRPC default
		ServerMinInterval: time.Duration(1) * time.Minute,  // match ClientInterval
	}
	mappingOfIpAndHost sync.Map // map[string]string{}
	onceConf           sync.Once
)

// KeepAliveOptions is used to set the gRPC keepalive settings for both
// clients and servers
type KeepaliveOptions struct {
	// ClientInterval is the duration after which if the client does not see
	// any activity from the server it pings the server to see if it is alive
	ClientInterval time.Duration
	// ClientTimeout is the duration the client waits for a response
	// from the server after sending a ping before closing the connection
	ClientTimeout time.Duration
	// ServerInterval is the duration after which if the server does not see
	// any activity from the client it pings the client to see if it is alive
	ServerInterval time.Duration
	// ServerTimeout is the duration the server waits for a response
	// from the client after sending a ping before closing the connection
	ServerTimeout time.Duration
	// ServerMinInterval is the minimum permitted time between client pings.
	// If clients send pings more frequently, the server will disconnect them
	ServerMinInterval time.Duration
}

func InitMappingOfIpAndHost() {
	onceConf.Do(func() {
		updateMappingOfIpAndHost()
		go func() {
			t := time.NewTicker(time.Second * 5)
			for {
				select {
				case <-t.C:
					updateMappingOfIpAndHost()
				}
			}
		}()
	})
}

func GetHostNameFromIp(ipAddr string) string {
	v, ok := mappingOfIpAndHost.Load(ipAddr)
	if !ok {
		return ""
	}
	return v.(string)
}

func updateMappingOfIpAndHost() {
	hostsPath := "/etc/hosts"
	fd, err := ioutil.ReadFile(hostsPath)
	if err != nil {
		log.Logger.Errorf("cannot read from /etc/hosts, err is: %v", err)
		return
	}
	entries := strings.Split(string(fd), "\n")
	for _, entry := range entries {
		if strings.HasSuffix(entry, "peer.com") {
			trimmedEntry := strings.Replace(entry, " ", "", strings.Count(entry, " ")-1)
			ipAndHost := strings.Split(trimmedEntry, " ")
			if len(ipAndHost) == 1 {
				log.Logger.Errorf("this entry: %v should contain at least one space, skip it", entry)
				continue
			}
			mappingOfIpAndHost.Store(ipAndHost[0], ipAndHost[1])
		}
	}
}

// cacheConfiguration caches common package scoped variables
func cacheConfiguration() {
	if !configurationCached {
		tlsEnabled = conf.V.TLS.Enabled
		configurationCached = true
	}
}

// TLSEnabled return cached value for "peer.tls.enabled" configuration value
func TLSEnabled() bool {
	if !configurationCached {
		cacheConfiguration()
	}
	return tlsEnabled
}

// MaxRecvMsgSize returns the maximum message size in bytes that gRPC clients
// and servers can receive
func MaxRecvMsgSize() int {
	return maxRecvMsgSize
}

// MaxSendMsgSize returns the maximum message size in bytes that gRPC clients
// and servers can send
func MaxSendMsgSize() int {
	return maxSendMsgSize
}

// DefaultKeepaliveOptions returns sane default keepalive settings for gRPC
// servers and clients
func DefaultKeepaliveOptions() *KeepaliveOptions {
	return keepaliveOptions
}

// ServerKeepaliveOptions returns gRPC keepalive options for server.  If
// opts is nil, the default keepalive options are returned
func ServerKeepaliveOptions(ka *KeepaliveOptions) []grpc.ServerOption {
	// use default keepalive options if nil
	if ka == nil {
		ka = keepaliveOptions
	}
	var serverOpts []grpc.ServerOption
	kap := keepalive.ServerParameters{
		Time:    ka.ServerInterval,
		Timeout: ka.ServerTimeout,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(kap))
	kep := keepalive.EnforcementPolicy{
		MinTime: ka.ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))
	return serverOpts
}

// ClientKeepaliveOptions returns gRPC keepalive options for clients.  If
// opts is nil, the default keepalive options are returned
func ClientKeepaliveOptions(ka *KeepaliveOptions) []grpc.DialOption {
	// use default keepalive options if nil
	if ka == nil {
		ka = keepaliveOptions
	}

	var dialOpts []grpc.DialOption
	kap := keepalive.ClientParameters{
		Time:                ka.ClientInterval,
		Timeout:             ka.ClientTimeout,
		PermitWithoutStream: true,
	}
	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(kap))
	return dialOpts
}
