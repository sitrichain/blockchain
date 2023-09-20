package filter

import (
	"fmt"
	"reflect"

	"github.com/rongzer/blockchain/common/configtx"
	configtxapi "github.com/rongzer/blockchain/common/configtx/config"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// chainCreator 抽象接口, 提供链相关操作
type chainCreator interface {
	NewChainConfigManager(envConfigUpdate *cb.Envelope) (configtxapi.Manager, error)
	NewChain(configTx *cb.Envelope)
	ChainsCount() int
}

// 创建链消息过滤器, 系统链才过滤并处理创建链消息
type createChainFilter struct {
	maxChainsCount uint64
	chainCreator   chainCreator
}

func NewCreateChainFilter(maxChainsCount uint64, chainCreator chainCreator) filters.Filter {
	return &createChainFilter{
		maxChainsCount: maxChainsCount,
		chainCreator:   chainCreator,
	}
}

func (ccf *createChainFilter) Apply(message *cb.Envelope) (filters.Action, filters.Committer) {
	msgData := &cb.Payload{}

	if err := msgData.Unmarshal(message.Payload); err != nil {
		return filters.Forward, nil
	}

	if msgData.Header == nil {
		return filters.Forward, nil
	}

	chdr, err := utils.UnmarshalChannelHeader(msgData.Header.ChannelHeader)
	if err != nil {
		return filters.Forward, nil
	}

	// HeaderType_ORDERER_TRANSACTION类型表示消息为创建链消息
	if chdr.Type != int32(cb.HeaderType_ORDERER_TRANSACTION) {
		return filters.Forward, nil
	}

	// 如果配置的最多链数大于0,则需要判断并限制当前链数
	if ccf.maxChainsCount > 0 {
		if uint64(ccf.chainCreator.ChainsCount()) > ccf.maxChainsCount {
			log.Logger.Warnf("Rejecting chain creation because has reached the maximum chains number %d", ccf.maxChainsCount)
			return filters.Reject, nil
		}
	}

	configTx := &cb.Envelope{}
	if err = configTx.Unmarshal(msgData.Data); err != nil {
		return filters.Reject, nil
	}

	if err = ccf.authorizeAndInspect(configTx); err != nil {
		log.Logger.Debugf("Rejecting chain creation because %s", err)
		return filters.Reject, nil
	}

	return filters.Accept, &createChainCommitter{
		chainCreator: ccf.chainCreator,
		configTx:     configTx,
	}
}

func (ccf *createChainFilter) authorize(configEnvelope *cb.ConfigEnvelope) (configtxapi.Manager, error) {
	if configEnvelope.LastUpdate == nil {
		return nil, fmt.Errorf("Must include a config update")
	}

	configManager, err := ccf.chainCreator.NewChainConfigManager(configEnvelope.LastUpdate)
	if err != nil {
		return nil, fmt.Errorf("Error constructing new chain config from update: %s", err)
	}

	newChannelConfigEnv, err := configManager.ProposeConfigUpdate(configEnvelope.LastUpdate)
	if err != nil {
		return nil, err
	}

	if err = configManager.Apply(newChannelConfigEnv); err != nil {
		return nil, err
	}

	return configManager, nil
}

func (ccf *createChainFilter) inspect(proposedManager, configManager configtxapi.Manager) error {
	proposedEnv := proposedManager.ConfigEnvelope()
	actualEnv := configManager.ConfigEnvelope()
	if !reflect.DeepEqual(proposedEnv.Config, actualEnv.Config) {
		return fmt.Errorf("The config proposed by the chain creation request did not match the config received with the chain creation request")
	}
	return nil
}

func (ccf *createChainFilter) authorizeAndInspect(configTx *cb.Envelope) error {
	payload := &cb.Payload{}

	if err := payload.Unmarshal(configTx.Payload); err != nil {
		return fmt.Errorf("Rejecting chain proposal: Error unmarshaling envelope payload: %s", err)
	}

	if payload.Header == nil {
		return fmt.Errorf("Rejecting chain proposal: Not a config transaction")
	}

	chdr, err := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	if err != nil {
		return fmt.Errorf("Rejecting chain proposal: Error unmarshaling chain header: %s", err)
	}

	if chdr.Type != int32(cb.HeaderType_CONFIG) {
		return fmt.Errorf("Rejecting chain proposal: Not a config transaction")
	}

	configEnvelope := &cb.ConfigEnvelope{}

	if err = configEnvelope.Unmarshal(payload.Data); err != nil {
		return fmt.Errorf("Rejecting chain proposal: Error unmarshalling config envelope from payload: %s", err)
	}

	// Make sure that the config was signed by the appropriate authorized entities
	proposedManager, err := ccf.authorize(configEnvelope)
	if err != nil {
		return err
	}

	initializer := configtx.NewInitializer()
	configManager, err := configtx.NewConfigManager(configTx, initializer, nil)
	if err != nil {
		return fmt.Errorf("Failed to create config manager and handlers: %s", err)
	}

	// Make sure that the config does not modify any of the orderer
	return ccf.inspect(proposedManager, configManager)
}

type createChainCommitter struct {
	chainCreator chainCreator
	configTx     *cb.Envelope
}

// Commit 创建新的链
func (ccc *createChainCommitter) Commit() {
	ccc.chainCreator.NewChain(ccc.configTx)
}

// Isolated 标记该交易数据需单独成块
func (ccc *createChainCommitter) Isolated() bool {
	return true
}
