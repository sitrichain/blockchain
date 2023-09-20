package endorser

import (
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/statistics"
	"github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/utils"
	"go.uber.org/zap"
	"strings"
)

var (
	successProcessMessageResponse            = &ab.BroadcastResponse{Status: common.Status_SUCCESS}
	badRequestProcessMessageResponse         = &ab.BroadcastResponse{Status: common.Status_BAD_REQUEST}
	notFoundProcessMessageResponse           = &ab.BroadcastResponse{Status: common.Status_NOT_FOUND}
	serviceUnavailableProcessMessageResponse = &ab.BroadcastResponse{Status: common.Status_SERVICE_UNAVAILABLE}
	unknownProcessMessageResponse            = &ab.BroadcastResponse{Status: common.Status_UNKNOWN}
	internalErrorProcessMessageResponse      = &ab.BroadcastResponse{Status: common.Status_INTERNAL_SERVER_ERROR}
)

// 创建链
func (e *Endorser) createChainImplement(logger *zap.SugaredLogger, message *common.RBCMessage) *ab.BroadcastResponse {
	msg := &common.Envelope{}
	if err := msg.Unmarshal(message.Data); err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message,error: %s", err)
		return badRequestProcessMessageResponse
	}
	payload := &common.Payload{}
	if err := payload.Unmarshal(msg.Payload); err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message, dropping connection: %s", err)
		return badRequestProcessMessageResponse
	}
	if payload.Header == nil {
		logger.Errorf("Rejecting createChain message because of received malformed message, with missing header, dropping connection")
		return badRequestProcessMessageResponse
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message (bad chain header): %s", err)
		return badRequestProcessMessageResponse
	}

	if chdr.Type == int32(common.HeaderType_CONFIG_UPDATE) {
		msg, err = e.processor.Process(msg)
		if err != nil {
			logger.Errorf("Rejecting createChain CONFIG_UPDATE because %s", err)
			return badRequestProcessMessageResponse
		}

		if err = payload.Unmarshal(msg.Payload); err != nil || payload.Header == nil {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing")
			return internalErrorProcessMessageResponse
		}

		chdr, err = utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing (bad chain header): %s", err)
			return internalErrorProcessMessageResponse
		}

		if chdr.ChannelId == "" {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing (empty chain ID)")
			return internalErrorProcessMessageResponse
		}
	}

	c, ok := e.chainManager.GetChain(chdr.ChannelId)
	if !ok {
		logger.Errorf("Rejecting createChain message because chain %s was not found", chdr.ChannelId)
		return notFoundProcessMessageResponse
	}
	// 创建链消息不进行过滤
	//committer, err := c.Filters().Apply(msg)
	//if err != nil {
	//	logger.Errorf("Rejecting createChain message because of msgprocessor error: %s", err)
	//	return badRequestProcessMessageResponse, nil
	//}

	// 给这条创建链的交易增加attach——firstNode，指明创建该链的第一个节点，即该链raft集群的第一个节点
	msg.Attachs = map[string]string{"firstNode": conf.V.Sealer.Raft.EndPoint}
	if !c.Enqueue(msg, nil) {
		logger.Errorf("Rejecting createChain message because of consenter instructed us to shut down")
		return serviceUnavailableProcessMessageResponse
	}

	return successProcessMessageResponse
}

// 背书提案
func (e *Endorser) endorseProposal(logger *zap.SugaredLogger, message *common.RBCMessage) *ab.BroadcastResponse {
	if len(message.Data) > 10_000_000 {
		logger.Errorf("Rejecting endorseProposal message because of payload length must less then 10M. Current: %s", len(message.Data))
		return badRequestProcessMessageResponse
	}
	statistics.ReceivedProposalCount.Inc()
	var newMsg *common.RBCMessage
	var target []string
	var err error
	if newMsg, target, err = e.distributor.SendToEndorse(message); err != nil {
		return badRequestProcessMessageResponse
	}
	go func(newMsg *common.RBCMessage, target []string) {
		// 若发送目标是自身节点，则不走grpc发送，直接塞入endorserChan
		if isInTarget(conf.V.Peer.Endpoint, target) {
			e.endorserChan <- newMsg
		}
		newTarget := excludeInTarget(conf.V.Peer.Endpoint, target)
		if len(newTarget) != 0 {
			e.distributor.GetClientManager().SendToPeers(newTarget, newMsg)
		}
	}(newMsg, target)

	return successProcessMessageResponse
}

func isInTarget(a string, target []string) bool {
	for _, b := range target {
		if a == b {
			return true
		}
	}
	return false
}

func excludeInTarget(a string, target []string) []string {
	var newTarget []string
	for _, b := range target {
		if a != b {
			newTarget = append(newTarget, b)
		}
	}
	return newTarget
}

// 背书成功回复
func (e *Endorser) endorseSuccessReply(message *common.RBCMessage) {
	statistics.EndorseSuccessCount.Inc()
	e.distributor.AddUpEndorseSuccess(message)
}

// 背书失败回复
func (e *Endorser) endorseFailReply(message *common.RBCMessage) {
	statistics.EndorseFailCount.Inc()
	errInfo := string(message.Data)
	// 背书超时不算背书失败
	if strings.Index(errInfo, "Timeout expired while executing endorse") >= 0 {
		log.Logger.Warn(errInfo)
	}

	detail, ok := e.distributor.MarkProposalFail(message)
	if ok {
		log.Logger.Errorf("Endorse fail. %s. %s", errInfo, detail)
	}
}

// 发送HTTP请求
func (e *Endorser) sendHttpRequestImplement(logger *zap.SugaredLogger, message *common.RBCMessage) *ab.BroadcastResponse {
	if message == nil {
		logger.Errorf("Rejecting sendHttpRequest message because message is nil")
		return badRequestProcessMessageResponse
	}

	httpRequest := &common.RBCHttpRequest{}
	if err := httpRequest.Unmarshal(message.Data); err != nil {
		logger.Errorf("Rejecting sendHttpRequest message because unmarshal http request err : %s", err)
		return badRequestProcessMessageResponse
	}
	// http请求类型的rbcMessage的转发逻辑
	buf, err := e.httpClient.Reqest(httpRequest)
	if err != nil {
		logger.Errorf("Rejecting sendHttpRequest message because request http error:%s", err)
		return badRequestProcessMessageResponse
	}
	return &ab.BroadcastResponse{Status: common.Status_SUCCESS, Data: buf}
}
