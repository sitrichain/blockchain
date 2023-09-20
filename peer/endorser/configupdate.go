package endorser

import (
	"fmt"

	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/chain"
	"github.com/rongzer/blockchain/peer/consensus"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

type configUpdateProcessor struct {
	signer  crypto.LocalSigner
	manager *chain.Manager
}

func newConfigUpdateProcessor(manager *chain.Manager, signer crypto.LocalSigner) *configUpdateProcessor {
	return &configUpdateProcessor{
		manager: manager,
		signer:  signer,
	}
}

func getChainID(env *cb.Envelope) (string, error) {
	envPayload, err := utils.UnmarshalPayload(env.Payload)
	if err != nil {
		return "", fmt.Errorf("Failing to process config update because of payload unmarshaling error: %s", err)
	}

	if envPayload.Header == nil /* || envPayload.Header.ChannelHeader == nil */ {
		return "", fmt.Errorf("Failing to process config update because no chain ID was set")
	}

	chdr, err := utils.UnmarshalChannelHeader(envPayload.Header.ChannelHeader)
	if err != nil {
		return "", fmt.Errorf("Failing to process config update because of chain header unmarshaling error: %s", err)
	}

	if chdr.ChannelId == "" {
		return "", fmt.Errorf("Failing to process config update because no chain ID was set")
	}

	return chdr.ChannelId, nil
}

// Process takes in an envelope of type CONFIG_UPDATE and proceses it
// to transform it either into to a new chain creation request, or
// into a chain CONFIG transaction (or errors on failure)
func (p *configUpdateProcessor) Process(envConfigUpdate *cb.Envelope) (*cb.Envelope, error) {
	chainID, err := getChainID(envConfigUpdate)
	if err != nil {
		return nil, err
	}

	c, ok := p.manager.GetChain(chainID)
	if ok {
		log.Logger.Debugf("Processing chain reconfiguration request for chain %s", chainID)
		return p.existingChannelConfig(envConfigUpdate, chainID, c)
	}

	log.Logger.Debugf("Processing chain creation request for chain %s", chainID)
	return p.newChannelConfig(chainID, envConfigUpdate)
}

func (p *configUpdateProcessor) existingChannelConfig(envConfigUpdate *cb.Envelope, chainID string, chain consensus.Chain) (*cb.Envelope, error) {
	configEnvelope, err := chain.ProposeConfigUpdate(envConfigUpdate)
	if err != nil {
		return nil, err
	}

	return utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, chainID, p.signer, configEnvelope)
}

func (p *configUpdateProcessor) proposeNewChannelToSystemChannel(newChannelEnvConfig *cb.Envelope) (*cb.Envelope, error) {
	return utils.CreateSignedEnvelope(cb.HeaderType_ORDERER_TRANSACTION, p.manager.SystemChainID, p.signer, newChannelEnvConfig)
}

func (p *configUpdateProcessor) newChannelConfig(chainID string, envConfigUpdate *cb.Envelope) (*cb.Envelope, error) {
	ctxm, err := p.manager.NewChainConfigManager(envConfigUpdate)
	if err != nil {
		return nil, err
	}

	newChannelConfigEnv, err := ctxm.ProposeConfigUpdate(envConfigUpdate)
	if err != nil {
		return nil, err
	}

	newChannelEnvConfig, err := utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, chainID, p.signer, newChannelConfigEnv)
	if err != nil {
		return nil, err
	}

	return p.proposeNewChannelToSystemChannel(newChannelEnvConfig)
}