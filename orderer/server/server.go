package server

import (
	"runtime/debug"

	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/chain"
	"github.com/rongzer/blockchain/orderer/clients"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"golang.org/x/net/context"
)

// 原子广播服务接口实例结构
type server struct {
	dh *deliverHandler        // 分发接口实例
	ph *processMessageHandler // 消息接口实例
}

// NewAtomicBroadcastServer 创建原子广播服务接口实例
func NewAtomicBroadcastServer(chainManager *chain.Manager, signer crypto.LocalSigner, clientManager *clients.Manager) ab.AtomicBroadcastServer {
	return &server{
		dh: newDeliverHandler(chainManager),
		ph: newProcessMessageHandler(
			chainManager,
			signer,
			clientManager),
	}
}

// Deliver 注册接口, 发送块数据流给客户端
func (s *server) Deliver(srv ab.AtomicBroadcast_DeliverServer) error {
	log.Logger.Debug("Starting new Deliver handler")
	defer func() {
		if r := recover(); r != nil {
			log.Logger.Errorf("Deliver client triggered panic: %s\n%s", r, debug.Stack())
		}
		log.Logger.Infof("Closing Deliver stream")
	}()
	return s.dh.handle(srv)
}

// ProcessMessage 注册接口, 接收客户端传入的RBCMessage消息
func (s *server) ProcessMessage(ctx context.Context, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Logger.Errorf("Process Message triggered panic: %s\n%s", r, debug.Stack())
		}
	}()
	return s.ph.Handle(ctx, message)
}
