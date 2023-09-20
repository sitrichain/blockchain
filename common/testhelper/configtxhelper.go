package testhelper

import (
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/common/config"
	configtxmsp "github.com/rongzer/blockchain/common/config/msp"
	"github.com/rongzer/blockchain/common/configtx"
	"github.com/rongzer/blockchain/common/genesis"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	cb "github.com/rongzer/blockchain/protos/common"
	mspproto "github.com/rongzer/blockchain/protos/msp"
	genesisconfig "github.com/rongzer/blockchain/tool/configtxgen/localconfig"
	"github.com/rongzer/blockchain/tool/configtxgen/provisional"
)

// MakeGenesisBlock creates a genesis block using the test templates for the given chainID
func MakeGenesisBlock(chainID string) (*cb.Block, error) {
	return genesis.NewFactoryImpl(CompositeTemplate()).Block(chainID)
}

// MakeGenesisBlockWithMSPs creates a genesis block using the MSPs provided for the given chainID
func MakeGenesisBlockFromMSPs(chainID string, appMSPConf, ordererMSPConf *mspproto.MSPConfig,
	appOrgID, ordererOrgID string) (*cb.Block, error) {
	appOrgTemplate := configtx.NewSimpleTemplate(configtxmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.ApplicationGroupKey, appOrgID}, appMSPConf, true))
	ordererOrgTemplate := configtx.NewSimpleTemplate(configtxmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.OrdererGroupKey, ordererOrgID}, ordererMSPConf, true))
	composite := configtx.NewCompositeTemplate(OrdererTemplate(), appOrgTemplate, ApplicationOrgTemplate(), ordererOrgTemplate)
	return genesis.NewFactoryImpl(composite).Block(chainID)
}

// OrderererTemplate returns the test orderer template
func OrdererTemplate() configtx.Template {
	genConf := genesisconfig.OriginLoad(genesisconfig.SampleInsecureProfile)
	return provisional.New(genConf).ChannelTemplate()
}

// sampleOrgID apparently _must_ be set to DEFAULT or things break
// Beware when changing!
const sampleOrgID = "DEFAULT"

// ApplicationOrgTemplate returns the SAMPLE org with MSP template
func ApplicationOrgTemplate() configtx.Template {
	mspConf, err := msp.GetLocalMspConfig(conf.V.MSPDir, nil, sampleOrgID)
	if err != nil {
		log.Logger.Panicf("Could not load sample MSP config: %s", err)
	}
	return configtx.NewSimpleTemplate(configtxmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.ApplicationGroupKey, sampleOrgID}, mspConf, true))
}

// OrdererOrgTemplate returns the SAMPLE org with MSP template
func OrdererOrgTemplate() configtx.Template {
	mspConf, err := msp.GetLocalMspConfig(conf.V.MSPDir, nil, sampleOrgID)
	if err != nil {
		log.Logger.Panicf("Could not load sample MSP config: %s", err)
	}
	return configtx.NewSimpleTemplate(configtxmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.OrdererGroupKey, sampleOrgID}, mspConf, true))
}

// CompositeTemplate returns the composite template of peer, orderer, and MSP
func CompositeTemplate() configtx.Template {
	return configtx.NewCompositeTemplate(OrdererTemplate(), ApplicationOrgTemplate(), OrdererOrgTemplate())
}
