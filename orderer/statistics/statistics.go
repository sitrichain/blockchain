package statistics

import (
	"time"

	"github.com/rongzer/blockchain/common/log"
	"go.uber.org/atomic"
)

var (
	ReceivedProposalCount atomic.Uint64 // 接收提案数量
	SentEndorseCount      atomic.Uint64 // 发送背书数量
	EndorseSuccessCount   atomic.Uint64 // 背书成功数量
	EndorseFailCount      atomic.Uint64 // 背书失败数量
)

// LoopPrint 循环打印统计信息
func LoopPrint() {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			log.Logger.Debugw("Statistics.",
				"Received Proposal Count", ReceivedProposalCount.Load(),
				"Sent Endorse Count", SentEndorseCount.Load(),
				"Endorse Success Count", EndorseSuccessCount.Load(),
				"Endorse Fail Count", EndorseFailCount.Load(),
			)
			ReceivedProposalCount.Store(0)
			SentEndorseCount.Store(0)
			EndorseSuccessCount.Store(0)
			EndorseFailCount.Store(0)
		}
	}
}
