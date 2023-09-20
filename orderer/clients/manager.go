package clients

import (
	"context"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	retryNumber = 3 // 发送重试次数
)

// peer连接管理器
type Manager struct {
	signer   msp.SigningIdentity
	identity []byte
	peers    sync.Map
}

// NewManager 初始化管理器
func NewManager() (*Manager, error) {
	s, err := mspmgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		log.Logger.Errorf("New peer clients fail because get signer error :%s", err)
		return nil, err
	}
	id, err := s.Serialize()
	if err != nil {
		log.Logger.Errorf("New peer clients fail because get signer identity error :%s", err)
		return nil, err
	}

	return &Manager{
		signer:   s,
		identity: id,
	}, nil
}

// SendToPeers 向peer发送消息
func (m *Manager) SendToPeers(peers []string, message *common.RBCMessage) {
	if len(peers) == 0 {
		log.Logger.Errorf("Send message to peer fail because no peer input")
		return
	}
	// 对消息序列化后签名
	message.Extend = ""
	message.Creator = nil
	message.Sign = nil
	buf, err := message.Marshal()
	if err != nil {
		log.Logger.Errorf("Send message to peer fail because message marshal error :%s", err)
		return
	}
	signature, err := m.signer.Sign(buf)
	if err != nil {
		log.Logger.Errorf("Send message to peer fail because message signature error :%s", err)
		return
	}
	message.Creator = m.identity
	message.Sign = signature

	// 发送消息
	var wg sync.WaitGroup
	for i := range peers {
		wg.Add(1)
		go func(p string) {
			_ = m.send(message, p, retryNumber)
			wg.Done()
		}(peers[i])
	}
	wg.Wait()
}

// 获取指定peer的连接
func (m *Manager) GetClient(endpoint string, retry int) (peer.EndorserClient, error) {
	if retry < 3 {
		// 重试时获取连接每次都创建新连接
		conn, err := m.newClientConnection(endpoint)
		if err != nil {
			return nil, err
		}
		client := peer.NewEndorserClient(conn)
		m.peers.Store(endpoint, client)
		return client, nil
	}
	c, ok := m.peers.Load(endpoint)
	if ok {
		return c.(peer.EndorserClient), nil
	}

	// 创建新连接
	conn, err := m.newClientConnection(endpoint)
	if err != nil {
		return nil, err
	}
	client := peer.NewEndorserClient(conn)
	m.peers.Store(endpoint, client)
	return client, nil
}

// 发送消息
func (m *Manager) send(message *common.RBCMessage, endpoint string, retry int) error {
	client, err := m.GetClient(endpoint, retry)
	if err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("try 3 times send msg %s to %s fail:%s ", message.TxID, endpoint, err)
			return err
		}

		time.Sleep(time.Second)
		return m.send(message, endpoint, retry)
	}

	if _, err := client.SendTransaction(context.TODO(), message); err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("try 3 times send msg %s to %s fail:%s ", message.TxID, endpoint, err)
			return err
		}

		time.Sleep(time.Second)
		return m.send(message, endpoint, retry)
	}
	return nil
}

// 创建grpc连接
func (m *Manager) newClientConnection(endpoint string) (*grpc.ClientConn, error) {
	// 设置阻塞等待连接成功
	opts := []grpc.DialOption{grpc.WithBlock()}
	// 设置最大接收/发送消息的大小, 当前为10M
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(comm.MaxRecvMsgSize()),
		grpc.MaxCallSendMsgSize(comm.MaxSendMsgSize()),
	))
	// 设置keepalive相关设置
	opts = append(opts, comm.ClientKeepaliveOptions(comm.DefaultKeepaliveOptions())...)
	// TLS设置开启时, 附加证书使用TLS进行连接
	if comm.TLSEnabled() {
		crtPath := viper.GetString("peer.tls.cert")
		creds, err := credentials.NewClientTLSFromFile(crtPath, "peer0.peer.com")
		if err != nil {
			log.Logger.Errorf("Failed obtaining credentials: %v", err)
			opts = append(opts, grpc.WithInsecure())
		} else {
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	// 3秒连接超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	return grpc.DialContext(ctx, endpoint, opts...)
}
