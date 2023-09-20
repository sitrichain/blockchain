package filters

import (
	"fmt"

	ab "github.com/rongzer/blockchain/protos/common"
)

// Set 过滤规则集合
type Set []Filter

// Apply 按顺序执行过滤规则
func (s Set) Apply(message *ab.Envelope) (Committer, error) {
	for _, rule := range s {
		action, committer := rule.Apply(message)
		switch action {
		case Accept:
			return committer, nil
		case Reject:
			return nil, fmt.Errorf("Rejected by rule: %T", rule)
		default:
		}
	}
	return nil, fmt.Errorf("No matching filter found")
}
