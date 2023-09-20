package genesis

import (
	"github.com/rongzer/blockchain/common/configtx"
	cb "github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/utils"
)

// Factory facilitates the creation of genesis blocks.
type Factory interface {
	// Block returns a genesis block for a given channel ID.
	Block(channelID string) (*cb.Block, error)
}

type factory struct {
	template configtx.Template
}

// NewFactoryImpl creates a new Factory.
func NewFactoryImpl(template configtx.Template) Factory {
	return &factory{template: template}
}

// Block constructs and returns a genesis block for a given channel ID.
func (f *factory) Block(channelID string) (*cb.Block, error) {
	configEnv, err := f.template.Envelope(channelID)
	if err != nil {
		return nil, err
	}

	configUpdate := &cb.ConfigUpdate{}
	if err = configUpdate.Unmarshal(configEnv.ConfigUpdate); err != nil {
		return nil, err
	}

	payloadChannelHeader := utils.MakeChannelHeader(cb.HeaderType_CONFIG, channelID)
	payloadSignatureHeader := utils.MakeSignatureHeader(nil, utils.CreateNonceOrPanic())
	utils.SetTxID(payloadChannelHeader, payloadSignatureHeader)
	payloadHeader := utils.MakePayloadHeader(payloadChannelHeader, payloadSignatureHeader)
	payload := &cb.Payload{Header: payloadHeader, Data: utils.MarshalOrPanic(&cb.ConfigEnvelope{Config: &cb.Config{ChannelGroup: configUpdate.WriteSet}})}
	envelope := &cb.Envelope{Payload: utils.MarshalOrPanic(payload), Signature: nil}

	block := cb.NewBlock(0, nil)
	block.Data = &cb.BlockData{Data: [][]byte{utils.MarshalOrPanic(envelope)}}
	block.Header.DataHash = block.Data.Hash()
	block.Metadata.Metadata[cb.BlockMetadataIndex_LAST_CONFIG] = utils.MarshalOrPanic(&cb.Metadata{
		Value: utils.MarshalOrPanic(&cb.LastConfig{Index: 0}),
	})
	return block, nil
}
