package broadcastclient

import (
	"context"
	"strings"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/orderer"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	retryNumber = 3 // 发送重试次数
)

var broadcastClient CommunicateOrderer

type CommunicateOrderer interface {
	SendToOrderer(rbcMessage *common.RBCMessage) (*ab.BroadcastResponse, error)
}

type BroadcastClient struct {
	ordererAddress []string
	ordererClient  orderer.AtomicBroadcastClient
	ordererEndpoint string
	connProd comm.ConnectionProducer
}

func NewBroadcastClient() *BroadcastClient {
	bc := &BroadcastClient{}
	ordererAddresses := viper.GetString("orderer.address")
	if ordererAddresses == "" {
		ordererAddresses = "127.0.0.1:7050"
	}

	tempAddressed := strings.Split(ordererAddresses, ",")
	bc.ordererAddress = tempAddressed
	bc.connProd = comm.NewConnectionProducer(DefaultConnectionFactory(), bc.ordererAddress)

	return bc
}

func GetCommunicateOrderer() CommunicateOrderer {
	if broadcastClient == nil {
		broadcastClient = NewBroadcastClient()
	}

	return broadcastClient
}

func (b *BroadcastClient) getClient(retry int) (orderer.AtomicBroadcastClient, error) {

	// 重连创建新链接
	if retry < 3 {
		conn, endpoint, err := b.connProd.NewConnection()
		if err != nil {
			log.Logger.Errorf("connect to endPoint %s,err:%s", b.ordererEndpoint, err)
			return nil, err
		}

		client := DefaultABCFactory(conn)
		b.ordererClient = client
		b.ordererEndpoint = endpoint

		return client, nil
	}

	// 为空创建连接
	if b.ordererClient == nil {
		conn, endpoint, err := b.connProd.NewConnection()
		if err != nil {
			log.Logger.Errorf("connect to endPoint %s,err:%s", endpoint, err)
			return nil, err
		}

		client := DefaultABCFactory(conn)
		b.ordererClient = client
		b.ordererEndpoint = endpoint
	}

	return b.ordererClient, nil
}

func (b *BroadcastClient) send(rbcMessage *common.RBCMessage, retry int) (*ab.BroadcastResponse, error) {

	client, err := b.getClient(retry)
	if err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("try 3 times send msg %s to orderer endpoint : %s fail:%s ", rbcMessage.TxID, b.ordererEndpoint, err)
			return nil, err
		}

		time.Sleep(1 * time.Second)

		return b.send(rbcMessage, retry)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := client.ProcessMessage(ctx, rbcMessage)
	if err != nil {
		retry --
		if retry <= 0 {
			log.Logger.Errorf("processmessage : %s to endpoint : %s err :%s ", rbcMessage.TxID, b.ordererEndpoint, err)
			return nil, err
		}
		log.Logger.Error(err)
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
			crtPath := viper.GetString("orderer.tls.cert")
			creds, err := credentials.NewClientTLSFromFile(crtPath, "orderer.orderer.com")
			if err != nil {
				dialOpts = append(dialOpts, grpc.WithInsecure())
			} else {
				dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
			}
		} else {
			dialOpts = append(dialOpts, grpc.WithInsecure())
		}
		grpc.EnableTracing = true
		ctx := context.Background()
		ctx, _ = context.WithTimeout(ctx, time.Second*3)
		return grpc.DialContext(ctx, endpoint, dialOpts...)
	}
}

func DefaultABCFactory(conn *grpc.ClientConn) orderer.AtomicBroadcastClient {
	return orderer.NewAtomicBroadcastClient(conn)
}
