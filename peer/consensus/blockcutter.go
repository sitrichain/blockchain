package consensus

import (
	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
)

// Cutter 切块对象
type Cutter struct {
	ordererConfig         config.Orderer
	filters               filters.Set
	pendingBatch          []*cb.Envelope
	pendingBatchSizeBytes uint32
	pendingCommitters     []filters.Committer
}

// NewCutter 创建切块对象
func NewCutter(sharedConfig config.Orderer, filters filters.Set) *Cutter {
	return &Cutter{
		ordererConfig: sharedConfig,
		filters:       filters,
	}
}

// GetPendingBatchSize 获取累积消息的数量
func (r *Cutter) GetPendingBatchSize() int {
	return len(r.pendingBatch)
}

// Cut 返回当前累积的消息
func (r *Cutter) Cut() ([]*cb.Envelope, []filters.Committer) {
	batch := r.pendingBatch
	r.pendingBatch = nil

	committers := r.pendingCommitters
	r.pendingCommitters = nil

	r.pendingBatchSizeBytes = 0
	return batch, committers
}

// Ordered 丢入消息进行排队
func (r *Cutter) Ordered(message *cb.Envelope, committer filters.Committer) ([][]*cb.Envelope, [][]filters.Committer, bool) {
	// 无committer时消息要被再次过滤一遍, 以防配置发生了变更
	if committer == nil {
		var err error
		committer, err = r.filters.Apply(message)
		if err != nil {
			log.Logger.Errorf("Rejecting ordered message: %s", err)
			return nil, nil, false
		}
	}

	// 消息大小不包含附件
	messageSizeBytes := uint32(len(message.Payload) + len(message.Signature))

	// 如果标记了隔离, 或者消息大小超过落块限制, 则立即切割
	if committer.Isolated() || messageSizeBytes > r.ordererConfig.BatchSize().PreferredMaxBytes {
		if committer.Isolated() {
			log.Logger.Debugf("Found message which requested to be isolated, cutting into its own batch")
		} else {
			log.Logger.Debugf("The current message, with %v bytes, is larger than the preferred batch size of %v bytes and will be isolated.", messageSizeBytes, r.ordererConfig.BatchSize().PreferredMaxBytes)
		}

		var messageBatches [][]*cb.Envelope
		var committerBatches [][]filters.Committer

		// 如果有累积消息则先切出
		if len(r.pendingBatch) > 0 {
			messageBatch, committerBatch := r.Cut()
			messageBatches = append(messageBatches, messageBatch)
			committerBatches = append(committerBatches, committerBatch)
		}

		// 附加当前消息
		messageBatches = append(messageBatches, []*cb.Envelope{message})
		committerBatches = append(committerBatches, []filters.Committer{committer})

		return messageBatches, committerBatches, true
	}

	var messageBatches [][]*cb.Envelope
	var committerBatches [][]filters.Committer
	// 如果累积消息数据大小+当前消息大小大于落块限制, 则先切出累计消息
	if r.pendingBatchSizeBytes+messageSizeBytes > r.ordererConfig.BatchSize().PreferredMaxBytes {
		log.Logger.Debugf("Pending batch would overflow if current message is added, cutting batch now. The current message, with %v bytes, will overflow the pending batch of %v bytes.PreferredMaxBytes: %d", messageSizeBytes, r.pendingBatchSizeBytes, r.ordererConfig.BatchSize().PreferredMaxBytes)
		messageBatch, committerBatch := r.Cut()
		messageBatches = append(messageBatches, messageBatch)
		committerBatches = append(committerBatches, committerBatch)
	}

	// 当前消息压入pending进行累积
	r.pendingBatch = append(r.pendingBatch, message)
	r.pendingBatchSizeBytes += messageSizeBytes
	r.pendingCommitters = append(r.pendingCommitters, committer)

	// 压入当前消息后累积数量超过最大消息数, 则切出数据
	if len(r.pendingBatch) >= int(r.ordererConfig.BatchSize().MaxMessageCount) {
		log.Logger.Debugf("Batch size met, cutting batch. MaxMessageCount: %d | Current: %d", r.ordererConfig.BatchSize().MaxMessageCount, len(r.pendingBatch))
		messageBatch, committerBatch := r.Cut()
		messageBatches = append(messageBatches, messageBatch)
		committerBatches = append(committerBatches, committerBatch)
	}

	// 没有切出的数据时, 返回nil替代空slice
	if len(messageBatches) == 0 {
		return nil, nil, true
	}

	return messageBatches, committerBatches, true
}
