package provisional

import (
	"fmt"

	"github.com/rongzer/blockchain/common/cauthdsl"
	"github.com/rongzer/blockchain/common/config"
	configvaluesmsp "github.com/rongzer/blockchain/common/config/msp"
	"github.com/rongzer/blockchain/common/configtx"
	"github.com/rongzer/blockchain/common/genesis"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp"
	"github.com/rongzer/blockchain/common/policies"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
	genesisconfig "github.com/rongzer/blockchain/tool/configtxgen/localconfig"
)

// Generator can either create an orderer genesis block or config template
type Generator interface {
	GenesisBlock() *cb.Block

	// ChannelTemplate returns a template which can be used to help initialize a channel
	ChannelTemplate() configtx.Template

	// GenesisBlockForChannel TODO
	GenesisBlockForChannel(channelID string) *cb.Block
}

const (
	// ConsensusTypeSolo identifies the solo consensus implementation.
	ConsensusTypeSolo = "solo"
	// ConsensusTypeKafka identifies the Kafka-based consensus implementation.
	ConsensusTypeKafka = "kafka"

	// TestChainID is the default value of ChainID. It is used by all testing
	// networks. It it necessary to set and export this variable so that test
	// clients can connect without being rejected for targeting a chain which
	// does not exist.
	TestChainID = "testchainid"

	// BlockValidationPolicyKey TODO
	BlockValidationPolicyKey = "BlockValidation"

	// OrdererAdminsPolicy is the absolute path to the orderer admins policy
	OrdererAdminsPolicy = "/Channel/Orderer/Admins"
)

type bootstrapper struct {
	channelGroups     []*cb.ConfigGroup
	ordererGroups     []*cb.ConfigGroup
	applicationGroups []*cb.ConfigGroup
	consortiumsGroups []*cb.ConfigGroup
}

// New returns a new provisional bootstrap helper.
func New(conf *genesisconfig.Profile) Generator {
	bs := &bootstrapper{
		channelGroups: []*cb.ConfigGroup{
			// Chain Config Types
			config.DefaultHashingAlgorithm(),
			config.DefaultBlockDataHashingStructure(),

			// Default policies
			policies.TemplateImplicitMetaAnyPolicy([]string{}, configvaluesmsp.ReadersPolicyKey),
			policies.TemplateImplicitMetaAnyPolicy([]string{}, configvaluesmsp.WritersPolicyKey),
			policies.TemplateImplicitMetaMajorityPolicy([]string{}, configvaluesmsp.AdminsPolicyKey),
		},
	}

	if conf.Orderer != nil {
		// Orderer addresses
		oa := config.TemplateOrdererAddresses(conf.Orderer.Addresses)
		oa.Values[config.OrdererAddressesKey].ModPolicy = OrdererAdminsPolicy

		bs.ordererGroups = []*cb.ConfigGroup{
			oa,

			// Orderer Config Types
			config.TemplateConsensusType(conf.Orderer.OrdererType),
			config.TemplateBatchSize(&ab.BatchSize{
				MaxMessageCount:   conf.Orderer.BatchSize.MaxMessageCount,
				AbsoluteMaxBytes:  conf.Orderer.BatchSize.AbsoluteMaxBytes,
				PreferredMaxBytes: conf.Orderer.BatchSize.PreferredMaxBytes,
			}),
			config.TemplateBatchTimeout(conf.Orderer.BatchTimeout.String()),
			config.TemplateChannelRestrictions(conf.Orderer.MaxChannels),

			// Initialize the default Reader/Writer/Admins orderer policies, as well as block validation policy
			policies.TemplateImplicitMetaPolicyWithSubPolicy([]string{config.OrdererGroupKey}, BlockValidationPolicyKey, configvaluesmsp.WritersPolicyKey, cb.ImplicitMetaPolicy_ANY),
			policies.TemplateImplicitMetaAnyPolicy([]string{config.OrdererGroupKey}, configvaluesmsp.ReadersPolicyKey),
			policies.TemplateImplicitMetaAnyPolicy([]string{config.OrdererGroupKey}, configvaluesmsp.WritersPolicyKey),
			policies.TemplateImplicitMetaMajorityPolicy([]string{config.OrdererGroupKey}, configvaluesmsp.AdminsPolicyKey),
		}

		for _, org := range conf.Orderer.Organizations {
			mspConfig, err := msp.GetVerifyingMspConfig(org.MSPDir, org.ID)
			if err != nil {
				log.Logger.Panicf("1 - Error loading MSP configuration for org %s: %s", org.Name, err)
			}
			bs.ordererGroups = append(bs.ordererGroups,
				configvaluesmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.OrdererGroupKey, org.Name},
					mspConfig, org.AdminPrincipal == genesisconfig.AdminRoleAdminPrincipal,
				),
			)
		}

		switch conf.Orderer.OrdererType {
		case ConsensusTypeSolo:
		case ConsensusTypeKafka:
			bs.ordererGroups = append(bs.ordererGroups, config.TemplateKafkaBrokers(conf.Orderer.Kafka.Brokers))
		default:
			panic(fmt.Errorf("Wrong consenter type value given: %s", conf.Orderer.OrdererType))
		}
	}

	if conf.Application != nil {

		bs.applicationGroups = []*cb.ConfigGroup{
			// Initialize the default Reader/Writer/Admins application policies
			policies.TemplateImplicitMetaAnyPolicy([]string{config.ApplicationGroupKey}, configvaluesmsp.ReadersPolicyKey),
			policies.TemplateImplicitMetaAnyPolicy([]string{config.ApplicationGroupKey}, configvaluesmsp.WritersPolicyKey),
			policies.TemplateImplicitMetaMajorityPolicy([]string{config.ApplicationGroupKey}, configvaluesmsp.AdminsPolicyKey),
		}
		for _, org := range conf.Application.Organizations {
			mspConfig, err := msp.GetVerifyingMspConfig(org.MSPDir, org.ID)
			if err != nil {
				log.Logger.Panicf("2- Error loading MSP configuration for org %s: %s", org.Name, err)
			}

			bs.applicationGroups = append(bs.applicationGroups,
				configvaluesmsp.TemplateGroupMSPWithAdminRolePrincipal([]string{config.ApplicationGroupKey, org.Name},
					mspConfig, org.AdminPrincipal == genesisconfig.AdminRoleAdminPrincipal,
				),
			)
			var anchorProtos []*pb.AnchorPeer
			for _, anchorPeer := range org.AnchorPeers {
				anchorProtos = append(anchorProtos, &pb.AnchorPeer{
					Host: anchorPeer.Host,
					Port: int32(anchorPeer.Port),
				})
			}

			bs.applicationGroups = append(bs.applicationGroups, config.TemplateAnchorPeers(org.Name, anchorProtos))
		}

	}

	if conf.Consortiums != nil {
		tcg := config.TemplateConsortiumsGroup()
		tcg.Groups[config.ConsortiumsGroupKey].ModPolicy = OrdererAdminsPolicy

		// Fix for https://jira.rongzer.org/browse/FAB-4373
		// Note, AcceptAllPolicy in this context, does not grant any unrestricted
		// access, but allows the /Channel/Admins policy to evaluate to true
		// for the ordering system channel while set to MAJORITY with the addition
		// to the successful evaluation of the /Channel/Orderer/Admins policy (which
		// is not AcceptAll
		tcg.Groups[config.ConsortiumsGroupKey].Policies[configvaluesmsp.AdminsPolicyKey] = &cb.ConfigPolicy{
			Policy: &cb.Policy{
				Type:  int32(cb.Policy_SIGNATURE),
				Value: utils.MarshalOrPanic(cauthdsl.AcceptAllPolicy),
			},
		}

		bs.consortiumsGroups = append(bs.consortiumsGroups, tcg)

		for consortiumName, consortium := range conf.Consortiums {
			cg := config.TemplateConsortiumChannelCreationPolicy(consortiumName, policies.ImplicitMetaPolicyWithSubPolicy(
				configvaluesmsp.AdminsPolicyKey,
				cb.ImplicitMetaPolicy_ANY,
			).Policy)

			cg.Groups[config.ConsortiumsGroupKey].Groups[consortiumName].ModPolicy = OrdererAdminsPolicy
			cg.Groups[config.ConsortiumsGroupKey].Groups[consortiumName].Values[config.ChannelCreationPolicyKey].ModPolicy = OrdererAdminsPolicy
			bs.consortiumsGroups = append(bs.consortiumsGroups, cg)

			for _, org := range consortium.Organizations {
				mspConfig, err := msp.GetVerifyingMspConfig(org.MSPDir, org.ID)
				if err != nil {
					log.Logger.Panicf("3 - Error loading MSP configuration for org %s: %s", org.Name, err)
				}
				bs.consortiumsGroups = append(bs.consortiumsGroups,
					configvaluesmsp.TemplateGroupMSPWithAdminRolePrincipal(
						[]string{config.ConsortiumsGroupKey, consortiumName, org.Name},
						mspConfig, org.AdminPrincipal == genesisconfig.AdminRoleAdminPrincipal,
					),
				)
			}
		}
	}

	return bs
}

// ChannelTemplate TODO
func (bs *bootstrapper) ChannelTemplate() configtx.Template {
	return configtx.NewModPolicySettingTemplate(
		configvaluesmsp.AdminsPolicyKey,
		configtx.NewCompositeTemplate(
			configtx.NewSimpleTemplate(bs.channelGroups...),
			configtx.NewSimpleTemplate(bs.ordererGroups...),
			configtx.NewSimpleTemplate(bs.applicationGroups...),
		),
	)
}

// GenesisBlock TODO Deprecate and remove
func (bs *bootstrapper) GenesisBlock() *cb.Block {
	block, err := genesis.NewFactoryImpl(
		configtx.NewModPolicySettingTemplate(
			configvaluesmsp.AdminsPolicyKey,
			configtx.NewCompositeTemplate(
				configtx.NewSimpleTemplate(bs.consortiumsGroups...),
				bs.ChannelTemplate(),
			),
		),
	).Block(TestChainID)

	if err != nil {
		panic(err)
	}
	return block
}

// GenesisBlockForChannel TODO
func (bs *bootstrapper) GenesisBlockForChannel(channelID string) *cb.Block {
	block, err := genesis.NewFactoryImpl(
		configtx.NewModPolicySettingTemplate(
			configvaluesmsp.AdminsPolicyKey,
			configtx.NewCompositeTemplate(
				configtx.NewSimpleTemplate(bs.consortiumsGroups...),
				bs.ChannelTemplate(),
			),
		),
	).Block(channelID)

	if err != nil {
		panic(err)
	}
	return block
}
