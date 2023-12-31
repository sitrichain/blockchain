package config

import (
	"github.com/gogo/protobuf/proto"
	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/msp"
	"github.com/rongzer/blockchain/common/policies"
	cb "github.com/rongzer/blockchain/protos/common"
)

// Manager 配置管理器接口定义
type Manager interface {
	Resources

	// Apply 应用配置数据作为新配置
	Apply(configEnv *cb.ConfigEnvelope) error

	// Validate 模拟应用传入配置数据成为新配置, 最终不改数据
	Validate(configEnv *cb.ConfigEnvelope) error

	// Validate attempts to validate a new configtx against the current config state
	ProposeConfigUpdate(configtx *cb.Envelope) (*cb.ConfigEnvelope, error)

	// ChainID retrieves the chain ID associated with this manager
	ChainID() string

	// ConfigEnvelope returns the current config envelope
	ConfigEnvelope() *cb.ConfigEnvelope

	// Sequence returns the current sequence number of the config
	Sequence() uint64
}

// Resources is the common set of config resources for all channels
// Depending on whether chain is used at the orderer or at the peer, other
// config resources may be available
type Resources interface {
	// PolicyManager returns the policies.Manager for the channel
	PolicyManager() policies.Manager

	// ChannelConfig returns the config.Channel for the chain
	ChannelConfig() config.Channel

	// OrdererConfig returns the config.Orderer for the channel
	// and whether the Orderer config exists
	OrdererConfig() (config.Orderer, bool)

	// ConsortiumsConfig() returns the config.Consortiums for the channel
	// and whether the consortiums config exists
	ConsortiumsConfig() (config.Consortiums, bool)

	// ApplicationConfig returns the configtxapplication.SharedConfig for the channel
	// and whether the Application config exists
	ApplicationConfig() (config.Application, bool)

	// MSPManager returns the msp.MSPManager for the chain
	MSPManager() msp.MSPManager
}

// Transactional is an interface which allows for an update to be proposed and rolled back
type Transactional interface {
	// RollbackConfig called when a config proposal is abandoned
	RollbackProposals(tx interface{})

	// PreCommit verifies that the transaction can be committed successfully
	PreCommit(tx interface{}) error

	// CommitConfig called when a config proposal is committed
	CommitProposals(tx interface{})
}

// PolicyHandler is used for config updates to policy
type PolicyHandler interface {
	Transactional

	BeginConfig(tx interface{}, groups []string) ([]PolicyHandler, error)

	ProposePolicy(tx interface{}, key string, path []string, policy *cb.ConfigPolicy) (proto.Message, error)
}

// Proposer contains the references necesssary to appropriately unmarshal
// a cb.ConfigGroup
type Proposer interface {
	// ValueProposer return the root value proposer
	ValueProposer() config.ValueProposer

	// PolicyProposer return the root policy proposer
	PolicyProposer() policies.Proposer
}

// Initializer is used as indirection between Manager and Handler to allow
// for single Handlers to handle multiple paths
type Initializer interface {
	Proposer

	Resources
}
