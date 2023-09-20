package filter

import (
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/orderer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
)

// 策略过滤器, 确认签名符合策略
type policyFilter struct {
	policySource  string
	policyManager policies.Manager
}

// NewPolicyFilter 创建策略过滤器
func NewPolicyFilter(policySource string, policyManager policies.Manager) filters.Filter {
	return &policyFilter{
		policySource:  policySource,
		policyManager: policyManager,
	}
}

// Apply 验证消息签名属于策略范围
func (pf *policyFilter) Apply(message *cb.Envelope) (filters.Action, filters.Committer) {
	// 获取策略
	policy, ok := pf.policyManager.GetPolicy(pf.policySource)
	if !ok {
		log.Logger.Warnf("Could not find policy %s", pf.policySource)
		return filters.Reject, nil
	}
	// 获取签名数据
	signedData, err := message.AsSignedData()
	if err != nil {
		return filters.Reject, nil
	}
	// 验签
	if err = policy.Evaluate(signedData); err == nil {
		return filters.Forward, nil
	}

	return filters.Reject, nil
}
