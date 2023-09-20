package server

import (
	"context"
	"strings"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/dispatcher"
	"github.com/rongzer/blockchain/peer/endorserclient"
	"github.com/rongzer/blockchain/peer/statistics"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"go.uber.org/zap"
)

type processMessageHandler struct {
	distributor dispatcher.Distributor
	hc          *comm.HttpClient
}

// newProcessMessageHandler 创建接口
func newProcessMessageHandler(chainManager *chain.Manager, signer crypto.LocalSigner, clientManager *endorserclient.Manager) *processMessageHandler {
	return &processMessageHandler{
		distributor: dispatcher.NewDistributor(clientManager, chainManager, signer),
		hc:          comm.NewHttpClient(),
	}
}

var (
	successProcessMessageResponse            = &ab.BroadcastResponse{Status: cb.Status_SUCCESS}
	badRequestProcessMessageResponse         = &ab.BroadcastResponse{Status: cb.Status_BAD_REQUEST}
	notFoundProcessMessageResponse           = &ab.BroadcastResponse{Status: cb.Status_NOT_FOUND}
	serviceUnavailableProcessMessageResponse = &ab.BroadcastResponse{Status: cb.Status_SERVICE_UNAVAILABLE}
	unknownProcessMessageResponse            = &ab.BroadcastResponse{Status: cb.Status_UNKNOWN}
	internalErrorProcessMessageResponse      = &ab.BroadcastResponse{Status: cb.Status_INTERNAL_SERVER_ERROR}
)

// Handle 处理接收到的消息
func (ph *processMessageHandler) Handle(_ context.Context, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	logger := log.Logger.With(zap.String("TxID", message.TxID), zap.Int32("type", message.Type))

	switch message.Type {
	case 5: // 背书成功回复
		return ph.endorseSuccessReply(message)
	case 6: // 背书失败回复
		return ph.endorseFailReply(logger, message)
	case 21: // 发送HTTP请求
		return ph.sendHttpRequest(logger, message)
	default:
		return badRequestProcessMessageResponse, nil
	}
}

// 背书成功回复
func (ph *processMessageHandler) endorseSuccessReply(message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	statistics.EndorseSuccessCount.Inc()
	ph.distributor.AddUpEndorseSuccess(message)
	return successProcessMessageResponse, nil
}

// 背书失败回复
func (ph *processMessageHandler) endorseFailReply(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	statistics.EndorseFailCount.Inc()
	errInfo := string(message.Data)
	// 背书超时不算背书失败
	if strings.Index(errInfo, "Timeout expired while executing endorse") >= 0 {
		log.Logger.Warn(errInfo)
		return successProcessMessageResponse, nil
	}

	detail, ok := ph.distributor.MarkProposalFail(message)
	if ok {
		logger.Errorf("Endorse fail. %s. %s", errInfo, detail)
	}

	return successProcessMessageResponse, nil
}

// 发送HTTP请求
func (ph *processMessageHandler) sendHttpRequest(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	if message == nil {
		logger.Errorf("Rejecting sendHttpRequest message because message is nil")
		return badRequestProcessMessageResponse, nil
	}

	httpRequest := &cb.RBCHttpRequest{}
	if err := httpRequest.Unmarshal(message.Data); err != nil {
		logger.Errorf("Rejecting sendHttpRequest message because unmarshal http request err : %s", err)
		return badRequestProcessMessageResponse, nil
	}
	buf, err := ph.hc.Reqest(httpRequest)
	if err != nil {
		logger.Errorf("Rejecting sendHttpRequest message because request http error:%s", err)
		return badRequestProcessMessageResponse, nil
	}
	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
}
