package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/consensus"
	"github.com/rongzer/blockchain/orderer/localconfig"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
)

// mode holds the implementation of type that satisfies the
// multichain.Mode interface --as the NewConsenter contract requires-- and
// the consenterConfig one.
type mode struct {
	brokerConfigVal *sarama.Config
	tlsConfigVal    localconfig.TLS
	retryOptionsVal localconfig.Retry
	kafkaVersionVal sarama.KafkaVersion
}

// Init creates a Kafka-based consenter. Called by orderer's main.go.
func Init(tlsConfig localconfig.TLS, retryOptions localconfig.Retry, kafkaVersion sarama.KafkaVersion) consensus.Mode {
	brokerConfig := newBrokerConfig(tlsConfig, retryOptions, kafkaVersion)
	return &mode{
		brokerConfigVal: brokerConfig,
		tlsConfigVal:    tlsConfig,
		retryOptionsVal: retryOptions,
		kafkaVersionVal: kafkaVersion}
}

// NewConsenter creates/returns a reference to a multichain.Consenter object for the
// given set of support resources. Implements the multichain.Mode
// interface. Called by multichain.newChainSupport(), which is itself called by
// multichain.NewManagerImpl() when ranging over the ledgerFactory's
// existingChains.
func (m *mode) NewConsenter(chain consensus.ChainResource, metadata *cb.Metadata) (consensus.Consenter, error) {
	var lastOffsetPersisted int64
	if metadata.Value != nil {
		// Extract orderer-related metadata from the tip of the ledger first
		kafkaMetadata := &ab.KafkaMetadata{}
		if err := kafkaMetadata.Unmarshal(metadata.Value); err != nil {
			log.Logger.Errorf("[chain: %s] Ledger may be corrupted:" +
				"cannot unmarshal orderer metadata in most recent block")
			return nil, err
		}
		lastOffsetPersisted = kafkaMetadata.LastOffsetPersisted
	} else {
		lastOffsetPersisted = sarama.OffsetOldest - 1 // default
	}

	return newConsenter(m, chain, lastOffsetPersisted)
}

func (m *mode) brokerConfig() *sarama.Config {
	return m.brokerConfigVal
}

func (m *mode) retryOptions() localconfig.Retry {
	return m.retryOptionsVal
}
