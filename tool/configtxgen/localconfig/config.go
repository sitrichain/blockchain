package localconfig

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/viperutil"
	cf "github.com/rongzer/blockchain/peer/config"
	"github.com/spf13/viper"
)

const (
	// Prefix identifies the prefix for the configtxgen-related ENV vars.
	Prefix string = "CONFIGTX"
)

var (
	configName = strings.ToLower(Prefix)
)

const (
	// SampleInsecureProfile references the sample profile which does not include any MSPs and uses solo for ordering.
	SampleInsecureProfile = "SampleInsecureSolo"

	// AdminRoleAdminPrincipal is set as AdminRole to cause the MSP role of type Admin to be used as the admin principal default
	AdminRoleAdminPrincipal = "Role.ADMIN"
)

// TopLevel consists of the structs used by the configtxgen tool.
type TopLevel struct {
	Profiles      map[string]*Profile `yaml:"Profiles"`
	Organizations []*Organization     `yaml:"Organizations"`
	Application   *Application        `yaml:"Application"`
	Orderer       *Orderer            `yaml:"Orderer"`
}

// Profile encodes orderer/application configuration combinations for the configtxgen tool.
type Profile struct {
	Consortium  string                 `yaml:"Consortium"`
	Application *Application           `yaml:"Application"`
	Orderer     *Orderer               `yaml:"Orderer"`
	Consortiums map[string]*Consortium `yaml:"Consortiums"`
}

// Consortium represents a group of organizations which may create channels with eachother
type Consortium struct {
	Organizations []*Organization `yaml:"Organizations"`
}

// Application encodes the application-level configuration needed in config transactions.
type Application struct {
	Organizations []*Organization `yaml:"Organizations"`
}

// Organization encodes the organization-level configuration needed in config transactions.
type Organization struct {
	Name           string `yaml:"Name"`
	ID             string `yaml:"ID"`
	MSPDir         string `yaml:"MSPDir"`
	AdminPrincipal string `yaml:"AdminPrincipal"`

	// Note: Viper deserialization does not seem to care for
	// embedding of types, so we use one organization struct
	// for both orderers and applications.
	AnchorPeers []*AnchorPeer `yaml:"AnchorPeers"`
}

// AnchorPeer encodes the necessary fields to identify an anchor peer.
type AnchorPeer struct {
	Host string `yaml:"Host"`
	Port int    `yaml:"Port"`
}

// ApplicationOrganization ...
// TODO This should probably be removed
type ApplicationOrganization struct {
	Organization `yaml:"Organization"`
}

// Orderer contains configuration which is used for the
// bootstrapping of an orderer by the provisional bootstrapper.
type Orderer struct {
	OrdererType   string          `yaml:"OrdererType"`
	Addresses     []string        `yaml:"Addresses"`
	BatchTimeout  time.Duration   `yaml:"BatchTimeout"`
	BatchSize     BatchSize       `yaml:"BatchSize"`
	Kafka         Kafka           `yaml:"Kafka"`
	Organizations []*Organization `yaml:"Organizations"`
	MaxChannels   uint64          `yaml:"MaxChannels"`
}

// BatchSize contains configuration affecting the size of batches.
type BatchSize struct {
	MaxMessageCount   uint32 `yaml:"MaxMessageSize"`
	AbsoluteMaxBytes  uint32 `yaml:"AbsoluteMaxBytes"`
	PreferredMaxBytes uint32 `yaml:"PreferredMaxBytes"`
}

// Kafka contains configuration for the Kafka-based orderer.
type Kafka struct {
	Brokers []string `yaml:"Brokers"`
}

var genesisDefaults = TopLevel{
	Orderer: &Orderer{
		OrdererType:  "solo",
		Addresses:    []string{"127.0.0.1:7050"},
		BatchTimeout: 2 * time.Second,
		BatchSize: BatchSize{
			MaxMessageCount:   10,
			AbsoluteMaxBytes:  10 * 1024 * 1024,
			PreferredMaxBytes: 512 * 1024,
		},
		Kafka: Kafka{
			Brokers: []string{"127.0.0.1:9092"},
		},
	},
}

// Load returns the orderer/application config combination that corresponds to a given profile.
func Load(profile string) *Profile {
	config := viper.New()
	cf.InitViper(config, configName)
	config.SetConfigFile("./configtx.yaml")
	// For environment variables
	config.SetEnvPrefix(Prefix)
	config.AutomaticEnv()
	// This replacer allows substitution within the particular profile without having to fully qualify the name
	replacer := strings.NewReplacer(strings.ToUpper(fmt.Sprintf("profiles.%s.", profile)), "", ".", "_")
	config.SetEnvKeyReplacer(replacer)

	err := config.ReadInConfig()
	if err != nil {
		log.Logger.Panic("Error reading configuration: ", err)
	}
	log.Logger.Debugf("Using config file: %s", config.ConfigFileUsed())

	var uconf TopLevel
	err = viperutil.EnhancedExactUnmarshal(config, &uconf)
	if err != nil {
		log.Logger.Panic("Error unmarshaling config into struct: ", err)
	}

	result, ok := uconf.Profiles[strings.ToLower(profile)]
	log.Logger.Infof("%+v", uconf.Profiles)
	if !ok {
		log.Logger.Panic("Could not find profile: ", profile)
	}

	result.completeInitialization(filepath.Dir(config.ConfigFileUsed()))

	log.Logger.Infof("Loaded configuration: %s", config.ConfigFileUsed())

	return result
}

// Load returns the orderer/application config combination that corresponds to a given profile.
func OriginLoad(profile string) *Profile {
	config := viper.New()
	cf.InitViper(config, "sample-configtx")
	err := config.ReadInConfig()
	if err != nil {
		log.Logger.Panic("Error reading configuration: ", err)
	}
	log.Logger.Debugf("Using config file: %s", config.ConfigFileUsed())

	var uconf TopLevel
	err = viperutil.EnhancedExactUnmarshal(config, &uconf)
	if err != nil {
		log.Logger.Panic("Error unmarshaling config into struct: ", err)
	}

	//result, ok := uconf.Profiles[profile]
	result, ok := uconf.Profiles[strings.ToLower(profile)]

	if !ok {
		log.Logger.Panic("Could not find profile: ", profile)
	}

	result.completeInitialization(filepath.Dir(config.ConfigFileUsed()))

	log.Logger.Infof("Loaded configuration: %s", config.ConfigFileUsed())

	return result
}

func (p *Profile) completeInitialization(configDir string) {
	if p.Orderer != nil {
		for _, org := range p.Orderer.Organizations {
			if org.AdminPrincipal == "" {
				org.AdminPrincipal = AdminRoleAdminPrincipal
			}
			translatePaths(configDir, org)
		}
	}

	if p.Application != nil {
		for _, org := range p.Application.Organizations {
			if org.AdminPrincipal == "" {
				org.AdminPrincipal = AdminRoleAdminPrincipal
			}
			translatePaths(configDir, org)
		}
	}

	if p.Consortiums != nil {
		for _, consortium := range p.Consortiums {
			for _, org := range consortium.Organizations {
				if org.AdminPrincipal == "" {
					org.AdminPrincipal = AdminRoleAdminPrincipal
				}
				translatePaths(configDir, org)
			}
		}
	}

	// Some profiles will not define orderer parameters
	if p.Orderer == nil {
		return
	}

	for {
		switch {
		case p.Orderer.OrdererType == "":
			log.Logger.Infof("Orderer.OrdererType unset, setting to %s", genesisDefaults.Orderer.OrdererType)
			p.Orderer.OrdererType = genesisDefaults.Orderer.OrdererType
		case p.Orderer.Addresses == nil:
			log.Logger.Infof("Orderer.Addresses unset, setting to %s", genesisDefaults.Orderer.Addresses)
			p.Orderer.Addresses = genesisDefaults.Orderer.Addresses
		case p.Orderer.BatchTimeout == 0:
			log.Logger.Infof("Orderer.BatchTimeout unset, setting to %s", genesisDefaults.Orderer.BatchTimeout)
			p.Orderer.BatchTimeout = genesisDefaults.Orderer.BatchTimeout
		case p.Orderer.BatchSize.MaxMessageCount == 0:
			log.Logger.Infof("Orderer.BatchSize.MaxMessageCount unset, setting to %s", genesisDefaults.Orderer.BatchSize.MaxMessageCount)
			p.Orderer.BatchSize.MaxMessageCount = genesisDefaults.Orderer.BatchSize.MaxMessageCount
		case p.Orderer.BatchSize.AbsoluteMaxBytes == 0:
			log.Logger.Infof("Orderer.BatchSize.AbsoluteMaxBytes unset, setting to %s", genesisDefaults.Orderer.BatchSize.AbsoluteMaxBytes)
			p.Orderer.BatchSize.AbsoluteMaxBytes = genesisDefaults.Orderer.BatchSize.AbsoluteMaxBytes
		case p.Orderer.BatchSize.PreferredMaxBytes == 0:
			log.Logger.Infof("Orderer.BatchSize.PreferredMaxBytes unset, setting to %s", genesisDefaults.Orderer.BatchSize.PreferredMaxBytes)
			p.Orderer.BatchSize.PreferredMaxBytes = genesisDefaults.Orderer.BatchSize.PreferredMaxBytes
		case p.Orderer.Kafka.Brokers == nil:
			log.Logger.Infof("Orderer.Kafka.Brokers unset, setting to %v", genesisDefaults.Orderer.Kafka.Brokers)
			p.Orderer.Kafka.Brokers = genesisDefaults.Orderer.Kafka.Brokers
		default:
			return
		}
	}
}

func translatePaths(configDir string, org *Organization) {
	cf.TranslatePathInPlace(configDir, &org.MSPDir)
}
