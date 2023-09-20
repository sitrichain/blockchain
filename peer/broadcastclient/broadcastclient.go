package broadcastclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/orderer"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	retryNumber = 3 // 发送重试次数
)

type client struct {
	ordererEndpoint string
	cc              *grpc.ClientConn
	abc             orderer.AtomicBroadcastClient
}

type CommunicateOrderer interface {
	SendToOrderer(rbcMessage *common.RBCMessage) (*ab.BroadcastResponse, error)
}

type BroadcastClient struct {
	connProd comm.ConnectionProducer
}

func NewBroadcastClient(sourceEndpoint string) *BroadcastClient {
	return &BroadcastClient{
		connProd: comm.NewConnectionProducer(DefaultConnectionFactory(), sourceEndpoint),
	}
}

func GetCommunicateOrderer(sourceEndpoint string) CommunicateOrderer {
	return NewBroadcastClient(sourceEndpoint)
}

func (b *BroadcastClient) getClient() (*client, error) {
	cc, endpoint, err := b.connProd.NewConnection()
	if err != nil {
		log.Logger.Errorf("connect to endPoint err:%s", err)
		return nil, err
	}
	return &client{
		ordererEndpoint: endpoint,
		cc:              cc,
		abc:             orderer.NewAtomicBroadcastClient(cc),
	}, nil
}

func (b *BroadcastClient) send(rbcMessage *common.RBCMessage, retry int) (*ab.BroadcastResponse, error) {
	client, err := b.getClient()
	if err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("try 3 times send msg %s to orderer fail:%s ", rbcMessage.TxID, err)
			return nil, err
		}

		time.Sleep(1 * time.Second)
		return b.send(rbcMessage, retry)
	}
	defer client.cc.Close()

	res, err := client.abc.ProcessMessage(context.TODO(), rbcMessage)
	if err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("processmessage : %s to endpoint : %s err :%s ", rbcMessage.TxID, client.ordererEndpoint, err)
			return nil, err
		}

		time.Sleep(1 * time.Second)
		return b.send(rbcMessage, retry)
	}

	return res, nil
}

func (b *BroadcastClient) SendToOrderer(rbcMessage *common.RBCMessage) (*ab.BroadcastResponse, error) {
	return b.send(rbcMessage, retryNumber)
}

func DefaultConnectionFactory() func(endpoint string) (*grpc.ClientConn, error) {
	return func(endpoint string) (*grpc.ClientConn, error) {
		dialOpts := []grpc.DialOption{grpc.WithBlock()}
		// set max send/recv msg sizes
		dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize()),
			grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize())))
		// set the keepalive options
		kaOpts := comm.DefaultKeepaliveOptions()
		dialOpts = append(dialOpts, comm.ClientKeepaliveOptions(kaOpts)...)

		if comm.TLSEnabled() {
			comm.InitMappingOfIpAndHost()
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
			serverName := comm.GetHostNameFromIp(strings.Split(endpoint, ":")[0])
			credentials := credentials.NewTLS(&tls.Config{
				Certificates: []tls.Certificate{cert},
				ServerName:   serverName,
				RootCAs:      cp,
			})
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials))
		} else {
			dialOpts = append(dialOpts, grpc.WithInsecure())
		}

		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, time.Second*3)
		return grpc.DialContext(ctx, endpoint, dialOpts...)
	}
}
