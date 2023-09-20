package endorserclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/rongzer/blockchain/peer/events/producer"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	mspmgmt "github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	retryNumber = 3 // 发送重试次数
)

type client struct {
	cc *grpc.ClientConn
	ec peer.EndorserClient
}

// peer连接管理器
type Manager struct {
	signer   msp.SigningIdentity
	identity []byte
}

// NewManager 初始化管理器
func NewManager() (*Manager, error) {
	s, err := mspmgmt.GetLocalMSPOfSealer().GetDefaultSigningIdentity()
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
			defer wg.Done()
			// 若轮询挑选出的背书节点是自己，则不走grpc发送
			if p == conf.V.Peer.Endpoint {
				var err error
				if string(message.Data) == "success" {
					err = producer.Send(producer.CreateSuccessEvent(message.ChainID, message.TxID))
				} else {
					err = producer.Send(producer.CreateRejectionEvent(message.ChainID, message.TxID, string(message.Data)))
				}
				if err != nil {
					log.Logger.Errorf("producer send err : %s", err)
				}
				return
			}
			m.send(message, p, retryNumber)
		}(peers[i])
	}
	wg.Wait()
}

// 获取指定peer的连接
func (m *Manager) getClient(endpoint string) (*client, error) {
	cc, err := m.newClientConnection(endpoint)
	if err != nil {
		return nil, err
	}
	return &client{cc: cc, ec: peer.NewEndorserClient(cc)}, nil
}

// 发送消息
func (m *Manager) send(message *common.RBCMessage, endpoint string, retry int) error {
	client, err := m.getClient(endpoint)
	if err != nil {
		retry--
		if retry <= 0 {
			log.Logger.Errorf("try 3 times send msg %s to %s fail:%s ", message.TxID, endpoint, err)
			return err
		}

		time.Sleep(time.Second)
		return m.send(message, endpoint, retry)
	}
	defer client.cc.Close()

	if _, err := client.ec.SendTransaction(context.TODO(), message); err != nil {
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
		opts = append(opts, grpc.WithTransportCredentials(credentials))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// 3秒连接超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	return grpc.DialContext(ctx, endpoint, opts...)
}
