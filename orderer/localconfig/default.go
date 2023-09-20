package localconfig

import (
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/common/log"
)

// 配置默认值
var defaults = TopLevel{
	General: General{
		ListenAddress: "127.0.0.1",
		ListenPort:    7050,
		GenesisFile:   "genesisblock",
		Profile: Profile{
			Enabled: false,
			Address: "0.0.0.0:6060",
		},
		LogLevel:    "INFO",
		LocalMSPDir: "msp",
		LocalMSPID:  "DEFAULT",
		BCCSP:       factory.GetDefaultOpts(),
	},
	RAMLedger: RAMLedger{
		HistorySize: 10000,
	},
	FileLedger: FileLedger{
		Location: "/var/rongzer/blockchain/orderer",
		Prefix:   "rongzer-blockchain-ordererledger",
	},
	Kafka: Kafka{
		Retry: Retry{
			ShortInterval: 1 * time.Minute,
			ShortTotal:    10 * time.Minute,
			LongInterval:  10 * time.Minute,
			LongTotal:     12 * time.Hour,
			NetworkTimeouts: NetworkTimeouts{
				DialTimeout:  30 * time.Second,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			},
			Metadata: Metadata{
				RetryBackoff: 250 * time.Millisecond,
				RetryMax:     3,
			},
			Producer: Producer{
				RetryBackoff: 100 * time.Millisecond,
				RetryMax:     3,
			},
			Consumer: Consumer{
				RetryBackoff: 2 * time.Second,
			},
		},
		Verbose: false,
		Version: sarama.V0_10_2_0,
		TLS: TLS{
			Enabled: false,
		},
	},
}

const defaultLogMsg = "Set default value to the unset config"

// 将缺少的值设置默认, 如果有相对路径的地方也改为绝对路径
func (c *TopLevel) setMissingValueToDefault(configDir string) error {
	defer func() {
		// 转换相对路径为绝对路径
		c.General.TLS.RootCAs = translateCAs(configDir, c.General.TLS.RootCAs)
		c.General.TLS.ClientRootCAs = translateCAs(configDir, c.General.TLS.ClientRootCAs)
		translatePathInPlace(configDir, &c.General.TLS.PrivateKey)
		translatePathInPlace(configDir, &c.General.TLS.Certificate)
		translatePathInPlace(configDir, &c.General.GenesisFile)
		translatePathInPlace(configDir, &c.General.LocalMSPDir)
	}()
	// 补缺少值
	for {
		switch {
		case c.General.ListenAddress == "":
			log.Logger.Infow(defaultLogMsg, "General.ListenAddress", defaults.General.ListenAddress)
			c.General.ListenAddress = defaults.General.ListenAddress
		case c.General.ListenPort == 0:
			log.Logger.Infow(defaultLogMsg, "General.ListenPort", defaults.General.ListenPort)
			c.General.ListenPort = defaults.General.ListenPort

		case c.General.LogLevel == "":
			log.Logger.Infow(defaultLogMsg, "General.LogLevel", defaults.General.LogLevel)
			c.General.LogLevel = defaults.General.LogLevel

		case c.General.GenesisFile == "":
			log.Logger.Infow(defaultLogMsg, "General.GenesisFile", defaults.General.GenesisFile)
			c.General.GenesisFile = defaults.General.GenesisFile

		// KAFKA TLS开启情况下, 相关证书不得为空
		case c.Kafka.TLS.Enabled && c.Kafka.TLS.Certificate == "":
			return errors.New("General.Kafka.TLS.Certificate must be set if General.Kafka.TLS.Enabled is set to true.")
		case c.Kafka.TLS.Enabled && c.Kafka.TLS.PrivateKey == "":
			return errors.New("General.Kafka.TLS.PrivateKey must be set if General.Kafka.TLS.Enabled is set to true.")
		case c.Kafka.TLS.Enabled && c.Kafka.TLS.RootCAs == nil:
			return errors.New("General.Kafka.TLS.CertificatePool must be set if General.Kafka.TLS.Enabled is set to true.")

		case c.General.Profile.Enabled && c.General.Profile.Address == "":
			log.Logger.Infow(defaultLogMsg, "General.Profile.Address", defaults.General.Profile.Address)
			c.General.Profile.Address = defaults.General.Profile.Address

		case c.General.LocalMSPDir == "":
			log.Logger.Infow(defaultLogMsg, "General.LocalMSPDir", defaults.General.LocalMSPDir)
			c.General.LocalMSPDir = defaults.General.LocalMSPDir
		case c.General.LocalMSPID == "":
			log.Logger.Infow(defaultLogMsg, "General.LocalMSPID", defaults.General.LocalMSPID)
			c.General.LocalMSPID = defaults.General.LocalMSPID

		case c.FileLedger.Prefix == "":
			log.Logger.Infow(defaultLogMsg, "FileLedger.Prefix", defaults.FileLedger.Prefix)
			c.FileLedger.Prefix = defaults.FileLedger.Prefix

		case c.Kafka.Retry.ShortInterval == 0*time.Minute:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.ShortInterval", defaults.Kafka.Retry.ShortInterval)
			c.Kafka.Retry.ShortInterval = defaults.Kafka.Retry.ShortInterval
		case c.Kafka.Retry.ShortTotal == 0*time.Minute:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.ShortTotal", defaults.Kafka.Retry.ShortTotal)
			c.Kafka.Retry.ShortTotal = defaults.Kafka.Retry.ShortTotal
		case c.Kafka.Retry.LongInterval == 0*time.Minute:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.LongInterval", defaults.Kafka.Retry.LongInterval)
			c.Kafka.Retry.LongInterval = defaults.Kafka.Retry.LongInterval
		case c.Kafka.Retry.LongTotal == 0*time.Minute:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.LongTotal", defaults.Kafka.Retry.LongTotal)
			c.Kafka.Retry.LongTotal = defaults.Kafka.Retry.LongTotal

		case c.Kafka.Retry.NetworkTimeouts.DialTimeout == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.NetworkTimeouts.DialTimeout", defaults.Kafka.Retry.NetworkTimeouts.DialTimeout)
			c.Kafka.Retry.NetworkTimeouts.DialTimeout = defaults.Kafka.Retry.NetworkTimeouts.DialTimeout
		case c.Kafka.Retry.NetworkTimeouts.ReadTimeout == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.NetworkTimeouts.ReadTimeout", defaults.Kafka.Retry.NetworkTimeouts.ReadTimeout)
			c.Kafka.Retry.NetworkTimeouts.ReadTimeout = defaults.Kafka.Retry.NetworkTimeouts.ReadTimeout
		case c.Kafka.Retry.NetworkTimeouts.WriteTimeout == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.NetworkTimeouts.WriteTimeout", defaults.Kafka.Retry.NetworkTimeouts.WriteTimeout)
			c.Kafka.Retry.NetworkTimeouts.WriteTimeout = defaults.Kafka.Retry.NetworkTimeouts.WriteTimeout

		case c.Kafka.Retry.Metadata.RetryBackoff == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.Metadata.RetryBackoff", defaults.Kafka.Retry.Metadata.RetryBackoff)
			c.Kafka.Retry.Metadata.RetryBackoff = defaults.Kafka.Retry.Metadata.RetryBackoff
		case c.Kafka.Retry.Metadata.RetryMax == 0:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.Metadata.RetryMax", defaults.Kafka.Retry.Metadata.RetryMax)
			c.Kafka.Retry.Metadata.RetryMax = defaults.Kafka.Retry.Metadata.RetryMax

		case c.Kafka.Retry.Producer.RetryBackoff == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.Producer.RetryBackoff", defaults.Kafka.Retry.Producer.RetryBackoff)
			c.Kafka.Retry.Producer.RetryBackoff = defaults.Kafka.Retry.Producer.RetryBackoff
		case c.Kafka.Retry.Producer.RetryMax == 0:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.Producer.RetryMax", defaults.Kafka.Retry.Producer.RetryMax)
			c.Kafka.Retry.Producer.RetryMax = defaults.Kafka.Retry.Producer.RetryMax

		case c.Kafka.Retry.Consumer.RetryBackoff == 0*time.Second:
			log.Logger.Infow(defaultLogMsg, "Kafka.Retry.Consumer.RetryBackoff", defaults.Kafka.Retry.Consumer.RetryBackoff)
			c.Kafka.Retry.Consumer.RetryBackoff = defaults.Kafka.Retry.Consumer.RetryBackoff

		case c.Kafka.Version == sarama.KafkaVersion{}:
			log.Logger.Infow(defaultLogMsg, "Kafka.Version", defaults.Kafka.Version.String())
			c.Kafka.Version = defaults.Kafka.Version

		default:
			return nil
		}
	}
}
