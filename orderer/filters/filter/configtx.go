package filter

import (
	"fmt"

	configtxapi "github.com/rongzer/blockchain/common/configtx/config"
	"github.com/rongzer/blockchain/orderer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
)

// 配置消息过滤器
type configFilter struct {
	configManager configtxapi.Manager
}

// NewConfigFilter 创建基于指定配置管理器的配置过滤器
func NewConfigFilter(manager configtxapi.Manager) filters.Filter {
	return &configFilter{
		configManager: manager,
	}
}

// Apply 执行
func (cf *configFilter) Apply(message *cb.Envelope) (filters.Action, filters.Committer) {
	// 反序列化payload
	payload := &cb.Payload{}
	if err := payload.Unmarshal(message.Payload); err != nil {
		return filters.Forward, nil
	}
	// 反序列化ChannelHeader判断其中类型
	if payload.Header == nil {
		return filters.Forward, nil
	}
	chdr := &cb.ChannelHeader{}
	if err := chdr.Unmarshal(payload.Header.ChannelHeader); err != nil {
		return filters.Forward, nil
	}
	if chdr.Type != int32(cb.HeaderType_CONFIG) {
		return filters.Forward, nil
	}
	// 按配置交易数据反序列化
	configEnvelope := &cb.ConfigEnvelope{}
	if err := configEnvelope.Unmarshal(payload.Data); err != nil {
		return filters.Reject, nil
	}
	// 验证配置数据是否有效, 会在落块时应用新配置
	if err := cf.configManager.Validate(configEnvelope); err != nil {
		return filters.Reject, nil
	}

	return filters.Accept, &configCommitter{
		configManager:  cf.configManager,
		configEnvelope: configEnvelope,
	}
}

type configCommitter struct {
	configManager  configtxapi.Manager
	configEnvelope *cb.ConfigEnvelope
}

func (cc *configCommitter) Commit() {
	// 应用配置数据作为新配置
	if err := cc.configManager.Apply(cc.configEnvelope); err != nil {
		panic(fmt.Errorf("Could not apply config transaction which should have already been validated: %s", err))
	}
}

func (cc *configCommitter) Isolated() bool {
	return true
}
