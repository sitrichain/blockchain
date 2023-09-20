package server

import (
	"io"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/filters"
	"github.com/rongzer/blockchain/peer/filters/filter"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
)

type deliverHandler struct {
	chainManager  *chain.Manager
	policyFilters sync.Map // map[string]*PolicyFilter
}

// newDeliverHandler 创建分发接口Handler实现
func newDeliverHandler(chainManager *chain.Manager) *deliverHandler {
	return &deliverHandler{chainManager: chainManager}
}

// handle 接收请求
func (dh *deliverHandler) handle(srv ab.AtomicBroadcast_DeliverServer) error {
	for {
		// 接收请求
		envelope, err := srv.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Logger.Errorf("Error reading deliver message from stream: %s", err)
			return err
		}
		// 处理请求
		if err := dh.deliverBlocks(srv, envelope); err != nil {
			return err
		}
	}
}

var (
	successDeliverResponse            = &ab.DeliverResponse{Type: &ab.DeliverResponse_Status{Status: cb.Status_SUCCESS}}
	badRequestDeliverResponse         = &ab.DeliverResponse{Type: &ab.DeliverResponse_Status{Status: cb.Status_BAD_REQUEST}}
	notFoundDeliverResponse           = &ab.DeliverResponse{Type: &ab.DeliverResponse_Status{Status: cb.Status_NOT_FOUND}}
	forbiddenDeliverResponse          = &ab.DeliverResponse{Type: &ab.DeliverResponse_Status{Status: cb.Status_FORBIDDEN}}
	serviceUnavailableDeliverResponse = &ab.DeliverResponse{Type: &ab.DeliverResponse_Status{Status: cb.Status_SERVICE_UNAVAILABLE}}
)

// 处理请求
func (dh *deliverHandler) deliverBlocks(srv ab.AtomicBroadcast_DeliverServer, envelope *cb.Envelope) error {
	// 反序列化payload
	payload := &cb.Payload{}
	if err := payload.Unmarshal(envelope.Payload); err != nil {
		log.Logger.Errorf("Rejecting deliver request because of received an envelope with no payload: %s", err)
		return srv.Send(badRequestDeliverResponse)
	}
	if payload.Header == nil {
		log.Logger.Error("Rejecting deliver request because of malformed envelope received with bad header")
		return srv.Send(badRequestDeliverResponse)
	}
	// 反序列化payload的ChannelHeader
	chdr := &cb.ChannelHeader{}
	if err := chdr.Unmarshal(payload.Header.ChannelHeader); err != nil {
		log.Logger.Errorf("Rejecting deliver request because of failed to unmarshal chain header: %s", err)
		return srv.Send(badRequestDeliverResponse)
	}
	// 获取对应的链
	c, ok := dh.chainManager.GetChain(chdr.ChannelId)
	if !ok {
		// Note, we log this at DEBUG because SDKs will poll waiting for chains to be created
		// So we would expect our log to be somewhat flooded with these
		log.Logger.Debugf("Rejecting deliver because chain %s not found", chdr.ChannelId)
		return srv.Send(notFoundDeliverResponse)
	}
	// 获取链在共识层的错误信号
	erroredChan := c.Errored()
	select {
	case <-erroredChan:
		log.Logger.Errorf("[chain: %s] Rejecting deliver request because of consenter error", chdr.ChannelId)
		return srv.Send(serviceUnavailableDeliverResponse)
	default:
	}

	// 记录一次当前最新配置序号
	lastConfigSequence := c.Sequence()

	// 通过策略过滤器检查该次请求是否满足策略
	policyFilter, ok := dh.policyFilters.Load(chdr.ChannelId)
	if !ok {
		policyFilter = filter.NewPolicyFilter(policies.ChannelReaders, c.PolicyManager())
		dh.policyFilters.Store(chdr.ChannelId, policyFilter)
	}
	result, _ := policyFilter.(filters.Filter).Apply(envelope)
	if result != filters.Forward {
		log.Logger.Errorf("[chain: %s] Received unauthorized deliver request", chdr.ChannelId)
		return srv.Send(forbiddenDeliverResponse)
	}
	// 反序列化查找信息
	seekInfo := &ab.SeekInfo{}
	if err := seekInfo.Unmarshal(payload.Data); err != nil {
		log.Logger.Errorf("[chain: %s] Received a signed deliver request with malformed seekInfo payload: %s", chdr.ChannelId, err)
		return srv.Send(badRequestDeliverResponse)
	}
	if seekInfo.Start == nil || seekInfo.Stop == nil {
		log.Logger.Errorf("[chain: %s] Received seekInfo message with missing start or stop %v, %v", chdr.ChannelId, seekInfo.Start, seekInfo.Stop)
		return srv.Send(badRequestDeliverResponse)
	}
	// 获取账本数据迭代器
	cursor, number := c.Ledger().Iterator(seekInfo.Start)
	// 根据查找信息确定结束的块序号
	var stopNum uint64
	switch stop := seekInfo.Stop.Type.(type) {
	case *ab.SeekPosition_Oldest:
		stopNum = number
	case *ab.SeekPosition_Newest:
		stopNum = c.Ledger().Height() - 1
	case *ab.SeekPosition_Specified:
		stopNum = stop.Specified.Number
		if stopNum < number {
			log.Logger.Errorf("[chain: %s] Received invalid seekInfo message: start number %d greater than stop number %d", chdr.ChannelId, number, stopNum)
			return srv.Send(badRequestDeliverResponse)
		}
	}

	// 循环迭代器获取块数据
	for {
		if seekInfo.Behavior == ab.SeekInfo_BLOCK_UNTIL_READY {
			select {
			case <-erroredChan:
				log.Logger.Errorf("[chain: %s] Aborting deliver request because of consenter error", chdr.ChannelId)
				return srv.Send(serviceUnavailableDeliverResponse)
			case <-cursor.ReadyChan():
				// SeekInfo_BLOCK_UNTIL_READY指定如果数据还未准备好则一直卡在此处等待
			}
		} else {
			select {
			case <-cursor.ReadyChan():
			default:
				// 未指定等待时, 当块不存在时直接返回数据未找到
				return srv.Send(notFoundDeliverResponse)
			}
		}

		// 再拉取一次最新配置序号, 如有变化就重新获取一次策略并检查, 确保请求依旧满足策略
		currentConfigSequence := c.Sequence()
		if currentConfigSequence > lastConfigSequence {
			lastConfigSequence = currentConfigSequence

			policyFilter := filter.NewPolicyFilter(policies.ChannelReaders, c.PolicyManager())
			dh.policyFilters.Store(chdr.ChannelId, policyFilter)
			result, _ := policyFilter.Apply(envelope)
			if result != filters.Forward {
				log.Logger.Errorf("[chain: %s] Client authorization revoked for deliver request", chdr.ChannelId)
				return srv.Send(forbiddenDeliverResponse)
			}
		}

		// 从迭代器中获取下一个块数据
		block, status, err := cursor.Next()
		if err != nil {
			log.Logger.Errorf("[chain: %s] Error reading from chain, status: %v error: %v", chdr.ChannelId, status, err)
			switch status {
			case cb.Status_SERVICE_UNAVAILABLE:
				return srv.Send(serviceUnavailableDeliverResponse)
			case cb.Status_NOT_FOUND:
				return srv.Send(notFoundDeliverResponse)
			default:
				return srv.Send(notFoundDeliverResponse)
			}
		}

		// 回复块数据
		if err := srv.Send(&ab.DeliverResponse{Type: &ab.DeliverResponse_Block{Block: block}}); err != nil {
			log.Logger.Errorf("[chain: %s] Error sending to stream: %v", chdr.ChannelId, err)
			return err
		}
		// 达到结束块序号就跳出
		if stopNum == block.Header.Number {
			break
		}
	}

	// 最终发送完成状态信息
	if err := srv.Send(successDeliverResponse); err != nil {
		log.Logger.Errorf("[chain: %s] Error sending to stream: %s", chdr.ChannelId, err)
		return err
	}
	return nil
}
