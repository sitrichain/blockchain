package server

import (
	"context"
	"strconv"
	"strings"

	"github.com/rongzer/blockchain/common/comm"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/chain"
	"github.com/rongzer/blockchain/orderer/clients"
	"github.com/rongzer/blockchain/orderer/endorse"
	"github.com/rongzer/blockchain/orderer/statistics"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/utils"
	"go.uber.org/zap"
)

type processMessageHandler struct {
	chainManager  *chain.Manager
	clientManager *clients.Manager
	processor     *configUpdateProcessor
	distributor   *endorse.Distributor
	hc            *comm.HttpClient
}

// newProcessMessageHandler 创建接口
func newProcessMessageHandler(chainManager *chain.Manager, signer crypto.LocalSigner, clientManager *clients.Manager) *processMessageHandler {
	return &processMessageHandler{
		chainManager:  chainManager,
		clientManager: clientManager,
		processor:     newConfigUpdateProcessor(chainManager, signer),
		distributor:   endorse.NewDistributor(clientManager, chainManager, signer),
		hc:            comm.NewHttpClient(),
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
	case 0: // 创建链
		return ph.createChain(logger, message)
	case 1: // 获取块数据
		return ph.getBlock(logger, message)
	case 3: // 背书交易
		return ph.endorseProposal(logger, message)
	case 5: // 背书成功回复
		return ph.endorseSuccessReply(message)
	case 6: // 背书失败回复
		return ph.endorseFailReply(logger, message)
	case 11: // 更新Peer状态
		return ph.updatePeerStatus(logger, message)
	case 12: // 获取Peer列表
		return ph.getPeerList(logger, message)
	case 21: // 发送HTTP请求
		return ph.sendHttpRequest(logger, message)
	case 22: // 多种方式获取数据(块,交易)
		return ph.getData(logger, message)
	case 23: // 获取未背书完成的数量
		return ph.getUnendorsedCount()
	case 24: // 获取附件数据
		return ph.getAttach(logger, message)
	default:
		return badRequestProcessMessageResponse, nil
	}
}

// 创建链
func (ph *processMessageHandler) createChain(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	msg := &cb.Envelope{}
	if err := msg.Unmarshal(message.Data); err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message,error: %s", err)
		return badRequestProcessMessageResponse, nil
	}
	payload := &cb.Payload{}
	if err := payload.Unmarshal(msg.Payload); err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message, dropping connection: %s", err)
		return badRequestProcessMessageResponse, nil
	}
	if payload.Header == nil {
		logger.Errorf("Rejecting createChain message because of received malformed message, with missing header, dropping connection")
		return badRequestProcessMessageResponse, nil
	}
	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		logger.Errorf("Rejecting createChain message because of received malformed message (bad chain header): %s", err)
		return badRequestProcessMessageResponse, nil
	}

	if chdr.Type == int32(cb.HeaderType_CONFIG_UPDATE) {
		msg, err = ph.processor.Process(msg)
		if err != nil {
			logger.Errorf("Rejecting createChain CONFIG_UPDATE because %s", err)
			return badRequestProcessMessageResponse, nil
		}

		if err = payload.Unmarshal(msg.Payload); err != nil || payload.Header == nil {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing")
			return internalErrorProcessMessageResponse, nil
		}

		chdr, err = utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing (bad chain header): %s", err)
			return internalErrorProcessMessageResponse, nil
		}

		if chdr.ChannelId == "" {
			logger.Errorf("Rejecting createChain message because of generated bad transaction after CONFIG_UPDATE processing (empty chain ID)")
			return internalErrorProcessMessageResponse, nil
		}
	}

	c, ok := ph.chainManager.GetChain(chdr.ChannelId)
	if !ok {
		logger.Errorf("Rejecting createChain message because chain %s was not found", chdr.ChannelId)
		return notFoundProcessMessageResponse, nil
	}

	committer, err := c.Filters().Apply(msg)
	if err != nil {
		logger.Errorf("Rejecting createChain message because of msgprocessor error: %s", err)
		return badRequestProcessMessageResponse, nil
	}

	if !c.Enqueue(msg, committer) {
		logger.Errorf("Rejecting createChain message because of consenter instructed us to shut down")
		return serviceUnavailableProcessMessageResponse, nil
	}

	return successProcessMessageResponse, nil
}

// 获取块数据
func (ph *processMessageHandler) getBlock(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	c, ok := ph.chainManager.GetChain(message.ChainID)
	if !ok {
		logger.Errorf("Rejecting getBlock message because chain %s was not found", message.ChainID)
		return notFoundProcessMessageResponse, nil
	}

	num, err := strconv.ParseUint(message.Extend, 10, 64)
	if err != nil {
		num = c.LegderHeight() - 1 // 缺省获取最后一个块数据
	}

	// 超限
	currentHeight := c.LegderHeight()
	if num < 0 || num >= currentHeight {
		logger.Errorf("Rejecting getBlock message because num %d out of range current height %d", num, currentHeight)
		return notFoundProcessMessageResponse, nil
	}

	block, err := c.GetBlockByNumber(num)
	if err != nil {
		logger.Errorf("Rejecting getBlock message because not found block num %d", num)
		return notFoundProcessMessageResponse, nil
	}

	buf, err := block.Marshal()
	if err != nil {
		logger.Errorf("Rejecting getBlock message because unmarshal block err : %s", err)
		return badRequestProcessMessageResponse, nil
	}

	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
}

// 背书提案
func (ph *processMessageHandler) endorseProposal(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	if len(message.Data) > 10_000_000 {
		logger.Errorf("Rejecting endorseProposal message because of payload length must less then 10M. Current: %s", len(message.Data))
		return badRequestProcessMessageResponse, nil
	}
	statistics.ReceivedProposalCount.Inc()
	if err := ph.distributor.SendToEndorse(message); err != nil {
		return badRequestProcessMessageResponse, nil
	}

	return successProcessMessageResponse, nil
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

// 更新Peer状态
func (ph *processMessageHandler) updatePeerStatus(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	info := &cb.PeerInfo{}
	if err := info.Unmarshal(message.Data); err != nil {
		logger.Errorf("Rejecting updatePeerStatus message because unmarshal peer info err : %s", err)
		return badRequestProcessMessageResponse, nil
	}

	ph.distributor.UpdatePeerStatus(message.ChainID, info)
	return successProcessMessageResponse, nil
}

// 获取Peer列表
func (ph *processMessageHandler) getPeerList(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	buf, err := ph.distributor.MarshalPeerList(message.ChainID)
	if err != nil {
		logger.Errorf("Rejecting getPeerList message because %s", err)
		return badRequestProcessMessageResponse, nil
	}
	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
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

// 多种方式获取数据(块,交易)
func (ph *processMessageHandler) getData(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	c, ok := ph.chainManager.GetChain(message.ChainID)
	if !ok {
		logger.Errorf("Rejecting getData message because chain was not found")
		return notFoundProcessMessageResponse, nil
	}

	// 获取块信息
	switch message.Extend {
	case "0":
		info, err := c.GetBlockchainInfo()
		if err != nil {
			logger.Errorf("Rejecting getData message because get chain info error: %s", err)
			return notFoundProcessMessageResponse, nil
		}
		buf, err := info.Marshal()
		if err != nil {
			logger.Errorf("Rejecting getData message because chain info marshal error: %s", err)
			return badRequestProcessMessageResponse, nil
		}

		return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
	case "1": // 通过区块号获取区块
		num, err := strconv.ParseUint(string(message.Data), 10, 64)
		if err != nil {
			logger.Errorf("Rejecting getData message because invalid block num: %s", string(message.Data))
			return badRequestProcessMessageResponse, nil
		}

		block, err := c.GetBlockByNumber(num)
		if err != nil {
			logger.Errorf("Rejecting getData message because not found block num: %d", num)
			return notFoundProcessMessageResponse, nil
		}

		buf, err := block.Marshal()
		if err != nil {
			logger.Errorf("Rejecting getData message because marshal block error: %s", err)
			return badRequestProcessMessageResponse, nil
		}

		return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
	case "2": // 通过hash获取块
		block, err := c.GetBlockByHash(message.Data)
		if err != nil {
			logger.Errorf("Rejecting getData message because not found block hash: %s", message.Data)
			return notFoundProcessMessageResponse, nil
		}

		buf, err := block.Marshal()
		if err != nil {
			logger.Errorf("Rejecting getData message because marshal block error: %s", err)
			return badRequestProcessMessageResponse, nil
		}

		return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
	case "3": // 通过 txid 获取块
		block, err := c.GetBlockByTxID(string(message.Data))
		if err != nil {
			logger.Errorf("Rejecting getData message because not found block by txid: %s", message.Data)
			return notFoundProcessMessageResponse, nil
		}

		buf, err := block.Marshal()
		if err != nil {
			logger.Errorf("Rejecting getData message because marshal block error: %s", err)
			return badRequestProcessMessageResponse, nil
		}

		return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
	case "4": //通过 txid 获取 tx
		tx, err := c.GetTxByID(string(message.Data))
		if err != nil {
			logger.Errorf("Rejecting getData message because not found tx by txid: %s", message.Data)
			return notFoundProcessMessageResponse, nil
		}

		buf, err := tx.Marshal()
		if err != nil {
			logger.Errorf("Rejecting getData message because marshal tx error: %s", err)
			return badRequestProcessMessageResponse, nil
		}

		return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
	default:
		logger.Errorf("Rejecting getData message because of unknown extend: %s", message.Extend)
		return unknownProcessMessageResponse, nil
	}
}

// 获取未背书完成的数量
func (ph *processMessageHandler) getUnendorsedCount() (*ab.BroadcastResponse, error) {
	buf := []byte(strconv.Itoa(ph.distributor.GetUnendorseCount()))
	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
}

// 获取附件数据
func (ph *processMessageHandler) getAttach(logger *zap.SugaredLogger, message *cb.RBCMessage) (*ab.BroadcastResponse, error) {
	c, ok := ph.chainManager.GetChain(message.ChainID)
	if !ok {
		logger.Errorf("Rejecting getAttach message because of not found chain %s", message.ChainID)
		return notFoundProcessMessageResponse, nil
	}

	buf := []byte(c.GetAttach(string(message.Data)))
	return &ab.BroadcastResponse{Status: cb.Status_SUCCESS, Data: buf}, nil
}
