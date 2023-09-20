package chain

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/ledger"
	"github.com/rongzer/blockchain/peer/ledger/ledgermgmt"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"strings"
	"sync"

	"github.com/rongzer/blockchain/common/config"
	"github.com/rongzer/blockchain/common/configtx"
	configtxapi "github.com/rongzer/blockchain/common/configtx/config"
	"github.com/rongzer/blockchain/common/crypto"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/policies"
	"github.com/rongzer/blockchain/peer/consensus"
	"github.com/rongzer/blockchain/peer/filters"
	"github.com/rongzer/blockchain/peer/filters/filter"
	"github.com/rongzer/blockchain/peer/ledger/kvledger"
	cb "github.com/rongzer/blockchain/protos/common"
	pb "github.com/rongzer/blockchain/protos/peer"
	"github.com/rongzer/blockchain/protos/utils"
)

var manager *Manager
var chains sync.Map // 所有链map[string]*Chain

func GetManager() *Manager {
	return manager
}

// peer/scc包和peer/endorser/validator包需要调用的获取账本的工具函数
func GetLedger(cid string) ledger.PeerLedger {
	if c, ok := chains.Load(cid); ok {
		chain, ok := c.(*Chain)
		if !ok {
			return nil
		}
		ledger := chain.Ledger().KvLedger
		return ledger
	}
	return nil
}

// GetChannelsInfo returns an array with information about all channels for
// this peer
func GetChannelsInfo() []*pb.ChannelInfo {
	// array to store metadata for all channels
	var channelInfoArray []*pb.ChannelInfo

	chains.Range(func(key, value interface{}) bool {
		channelInfo := &pb.ChannelInfo{ChannelId: key.(string)}
		// add this specific chaincode's metadata to the array of all chaincodes
		channelInfoArray = append(channelInfoArray, channelInfo)
		return true
	})

	return channelInfoArray
}

// GetCurrConfigBlock returns the cached config block of the specified chain.
// Note that this call returns nil if chain cid has not been created.
func GetCurrConfigBlock(cid string) *cb.Block {
	if c, ok := chains.Load(cid); ok {
		chain, ok := c.(*Chain)
		if !ok {
			return nil
		}
		ledger := chain.Ledger().KvLedger
		// get most recent config block
		configBlock, err := ledger.GetBlockByNumber(chain.lastConfigIndex)
		if err != nil {
			return nil
		}
		return configBlock
	}
	return nil
}

// 配置资源
type ConfigResources struct {
	configtxapi.Manager
	ordererConfig config.Orderer
}

// SharedConfig 获取orderer配置
func (cr *ConfigResources) SharedConfig() config.Orderer {
	return cr.ordererConfig
}

// 账本资源 = 配置资源 + 文件账本
type LedgerResources struct {
	*ConfigResources
	ledger *kvledger.SignaledLedger
}

// 创建账本资源对象
func newLedgerResources(configTx *cb.Envelope) (*LedgerResources, error) {
	configManager, err := configtx.NewConfigManager(configTx, configtx.NewInitializer(), nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating configtx manager and handlers. %w", err)
	}
	ordererConfig, ok := configManager.OrdererConfig()
	if !ok {
		return nil, fmt.Errorf("[chain: %s] No orderer configuration", configManager.ChainID())
	}

	chainID := configManager.ChainID()
	l, err := ledgermgmt.GetSignaledLedger(chainID)
	if err != nil {
		return nil, fmt.Errorf("Error getting ledger for %s", chainID)
	}

	return &LedgerResources{
		ConfigResources: &ConfigResources{Manager: configManager, ordererConfig: ordererConfig},
		ledger:          l,
	}, nil
}

// Manager 提供对账本资源的访问及管理.
type Manager struct {
	Chains        sync.Map                  // 所有链 map[string]*Chain
	modes         map[string]consensus.Mode // 所有注册的共识模式
	signer        crypto.LocalSigner        // 签名对象
	systemChain   *Chain                    // 系统链
	SystemChainID string                    // 系统链ID
}

// NewManager 创建链管理器
func NewManager(modes map[string]consensus.Mode, signer crypto.LocalSigner, m *Manager) (*Manager, error) {
	m.signer = signer
	m.modes = modes

	if err := m.initialize(); err != nil {
		return nil, err
	}
	manager = m
	return m, nil
}

func (m *Manager) InitCreateChainAccordingly(chainID string, participateConsensus bool) error {
	// 获取链的账本
	chainLedger, err := ledgermgmt.GetSignaledLedger(chainID)
	if err != nil {
		return fmt.Errorf("Found exist chainID %s but could not retrieve its ledger. %w", chainID, err)
	}
	// 读取配置块. 最新的块中应当保存着最新配置块的索引, 配置块中第一笔交易应当为配置信息.
	lastBlock, err := chainLedger.GetBlockByNumber(chainLedger.Height() - 1)
	if err != nil {
		return fmt.Errorf("[chain: %s] Can not get lastest block. %w", chainID, err)
	}
	configBlockindex, err := utils.GetLastConfigIndexFromBlock(lastBlock)
	if err != nil {
		return fmt.Errorf("[chain: %s] Not found config block index in latest block. %w", chainID, err)
	}
	configBlock, err := chainLedger.GetBlockByNumber(configBlockindex)
	if configBlock == nil {
		return fmt.Errorf("Config block index %d does not exist of chainID %s", configBlockindex, chainID)
	}
	configEnvelope, err := utils.ExtractEnvelope(configBlock, 0)
	if err != nil {
		return fmt.Errorf("Extract config envelope of chainID %s from config block error. %w", chainID, err)
	}
	// 根据配置交易初始化账本资源对象
	ledgerResources, err := newLedgerResources(configEnvelope)
	if err != nil {
		return fmt.Errorf("Extract config envelope of chainID %s from config block error. %w", chainID, err)
	}
	var mode consensus.Mode
	var ok bool
	if participateConsensus {
		// 选择共识模式
		mode, ok = m.modes[ledgerResources.SharedConfig().ConsensusType()]
		if !ok {
			return fmt.Errorf("Unregistered consensus type: %s", ledgerResources.SharedConfig().ConsensusType())
		}
	}
	set := filters.Set{
		filter.NewEmptyRejectFilter(),
		filter.NewMaxBytesFilter(ledgerResources.SharedConfig().BatchSize().AbsoluteMaxBytes),
		filter.NewPolicyFilter(policies.ChannelWriters, ledgerResources.PolicyManager()),
		filter.NewConfigFilter(ledgerResources),
		filter.NewAcceptFilter(),
	}
	chain, err := newChain(set, ledgerResources, mode, m.signer)
	if err != nil {
		return err
	}
	m.Chains.Store(chainID, chain)
	chains.Store(chainID, chain)

	return nil
}

func (m *Manager) RestartCreateChainAccordingly(chainID string, isSystem bool) error {
	// 获取链的账本
	chainLedger, err := ledgermgmt.GetSignaledLedger(chainID)
	if err != nil {
		return fmt.Errorf("Found exist chainID %s but could not retrieve its ledger. %w", chainID, err)
	}
	// 读取配置块. 最新的块中应当保存着最新配置块的索引, 配置块中第一笔交易应当为配置信息.
	lastBlock, err := chainLedger.GetBlockByNumber(chainLedger.Height() - 1)
	if err != nil {
		return fmt.Errorf("[chain: %s] Can not get lastest block. %w", chainID, err)
	}
	configBlockindex, err := utils.GetLastConfigIndexFromBlock(lastBlock)
	if err != nil {
		return fmt.Errorf("[chain: %s] Not found config block index in latest block. %w", chainID, err)
	}
	configBlock, err := chainLedger.GetBlockByNumber(configBlockindex)
	if configBlock == nil {
		return fmt.Errorf("config block index %d does not exist of chainID %s", configBlockindex, chainID)
	}
	configEnvelope, err := utils.ExtractEnvelope(configBlock, 0)
	if err != nil {
		return fmt.Errorf("extract config envelope of chainID %s from config block error. %w", chainID, err)
	}
	// 根据配置交易初始化账本资源对象
	ledgerResources, err := newLedgerResources(configEnvelope)
	if err != nil {
		return fmt.Errorf("create ledger resources of chainID %s error. %w", chainID, err)
	}
	var mode consensus.Mode
	var ok bool
	// 节点创建或重启时是否参与系统链的共识，由配置conf.V.Sealer.ParticipateConsensusOfSysChain决定
	if isSystem {
		if conf.V.Sealer.ParticipateConsensusOfSysChain {
			// 选择共识模式
			mode, ok = m.modes[ledgerResources.SharedConfig().ConsensusType()]
			if !ok {
				return fmt.Errorf("Unregistered consensus type: %s", ledgerResources.SharedConfig().ConsensusType())
			}
		}
	} else {
		// 节点重启时是否参与应用链的共识，由该链最新块中的raft共识元数据决定
		metadata, err := utils.GetMetadataFromBlock(lastBlock, cb.BlockMetadataIndex_ORDERER)
		if err != nil {
			return fmt.Errorf("extract metadata of chainID %s from last block error. %w", chainID, err)
		}
		// 从metadata中读取*etcdraft.BlockMetadata
		if metadata == nil || len(metadata.Value) == 0 {
			return fmt.Errorf("there is no block metadata in last block of chainId %s", chainID)
		}
		raftMeta := &etcdraft.BlockMetadata{}
		if err := proto.Unmarshal(metadata.Value, raftMeta); err != nil {
			return fmt.Errorf("failed to unmarshal block's metadata")
		}
		var participateConsensus bool
		for _, endpoint := range raftMeta.ConsenterEndpoints {
			if endpoint == conf.V.Sealer.Raft.EndPoint {
				participateConsensus = true
			}
		}
		if participateConsensus {
			// 选择共识模式
			mode, ok = m.modes[ledgerResources.SharedConfig().ConsensusType()]
			if !ok {
				return fmt.Errorf("Unregistered consensus type: %s", ledgerResources.SharedConfig().ConsensusType())
			}
		}
	}

	set := filters.Set{
		filter.NewEmptyRejectFilter(),
		filter.NewMaxBytesFilter(ledgerResources.SharedConfig().BatchSize().AbsoluteMaxBytes),
		filter.NewPolicyFilter(policies.ChannelWriters, ledgerResources.PolicyManager()),
		filter.NewConfigFilter(ledgerResources),
		filter.NewAcceptFilter(),
	}
	chain, err := newChain(set, ledgerResources, mode, m.signer)
	if err != nil {
		return err
	}
	m.Chains.Store(chainID, chain)
	chains.Store(chainID, chain)
	if isSystem {
		m.SystemChainID = chainID
		m.systemChain = chain
	}
	return nil
}

// initialize 初始化
func (m *Manager) initialize() error {
	// 初始化复制器
	replicator, err := cluster.NewReplicator(conf.V.Sealer.SystemChainId)
	if err != nil {
		return err
	}
	// 若是后加入raft集群的节点，需要把系统链的所有块都拉下来；first节点不需要拉系统链，若first节点挂掉重启，也不需要用复制器去拉系统链，会走catup的逻辑
	if conf.V.Sealer.Raft.BootStrapEndPoint != "" {
		err := replicator.ReplicateSystemChain()
		if err != nil {
			return err
		}
	}
	// 初始化系统链
	err = m.RestartCreateChainAccordingly(conf.V.Sealer.SystemChainId, true)
	if err != nil {
		log.Logger.Errorf("cannot initialize system channel")
		return err
	}
	// 复制器工作完成后，开启系统链
	log.Logger.Infof("[chain: %s] Starting as system chain and orderer type %s", m.SystemChainID, m.systemChain.SharedConfig().ConsensusType())
	m.systemChain.Start()

	// 获取所有已存在的链
	existingChains, err := ledgermgmt.GetLedgerIDs()
	if err != nil {
		return err
	}
	// 初始化并开启其他应用链
	for _, chainID := range existingChains {
		if chainID == m.SystemChainID {
			continue
		}
		err := m.RestartCreateChainAccordingly(chainID, false)
		if err != nil {
			log.Logger.Errorf("cannot initialize channel: %v", chainID)
			return err
		}
		log.Logger.Infof("Starting Consenter %s", chainID)
		chain, ok := m.GetChain(chainID)
		if ok {
			chain.Start()
		}
	}

	if m.SystemChainID == "" {
		return fmt.Errorf("No system chain found. If bootstrapping, does your system chain contain a consortiums group definition?")
	}

	return nil
}

// GetChain 获取链
func (m *Manager) GetChain(chainID string) (*Chain, bool) {
	cs, ok := m.Chains.Load(chainID)
	if !ok {
		return nil, false
	}
	return cs.(*Chain), true
}

// NewChain 新建链
func (m *Manager) NewChain(configtx *cb.Envelope) {
	// 创建账本
	ledgerResources, err := newLedgerResources(configtx)
	if err != nil {
		log.Logger.Errorf("New ledger resources error. %s", err)
		return
	}
	// 构造元数据
	metadata := &cb.Metadata{}
	// 构造*etcdraft.BlockMetadata
	raftMeta := &etcdraft.BlockMetadata{
		ConsenterEndpoints: []string{conf.V.Sealer.Raft.EndPoint},
		NextConsenterId:    uint64(2),
		RaftIndex:          uint64(0),
		PeerEndpoints:      []string{conf.V.Peer.Endpoint},
	}
	// 重新序列化*etcdraft.BlockMetadata为metadata.Value
	metadata.Value = utils.MarshalOrPanic(raftMeta)
	// 写入创世块
	block, err := ledgerResources.ledger.CreateGenesisBlock([]*cb.Envelope{configtx}, metadata)
	if err != nil {
		log.Logger.Errorf("Create next block error. %s", err)
		return
	}
	if err := ledgerResources.ledger.Append(block); err != nil {
		log.Logger.Errorf("Write first block to ledger error. %s", err)
		return
	}
	// 选择共识模式
	mode, ok := m.modes[ledgerResources.SharedConfig().ConsensusType()]
	if !ok {
		log.Logger.Errorf("Unregistered consensus type: %s", ledgerResources.SharedConfig().ConsensusType())
		return
	}
	// 创建并启动链
	set := filters.Set{
		filter.NewEmptyRejectFilter(),
		filter.NewMaxBytesFilter(ledgerResources.SharedConfig().BatchSize().AbsoluteMaxBytes),
		filter.NewPolicyFilter(policies.ChannelWriters, ledgerResources.PolicyManager()),
		filter.NewConfigFilter(ledgerResources),
		filter.NewAcceptFilter(),
	}
	chain, err := newChain(set, ledgerResources, mode, m.signer)
	if err != nil {
		log.Logger.Errorf("newChain error. %s", err)
		return
	}
	chainID := ledgerResources.ChainID()
	m.Chains.Store(chainID, chain)
	chains.Store(chainID, chain)
	chain.Start()
	log.Logger.Warnf("Created and starting new chain %s", chainID)
}

// ChainsCount 获取链数量
func (m *Manager) ChainsCount() int {
	count := 0
	m.Chains.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// NewChainConfigManager 根据配置更新消息创建配置管理器
func (m *Manager) NewChainConfigManager(configEnv *cb.Envelope) (configtxapi.Manager, error) {
	// 反序列出通用payload
	configUpdatePayload := &cb.Payload{}
	if err := configUpdatePayload.Unmarshal(configEnv.Payload); err != nil {
		return nil, fmt.Errorf("Failing chain config creation because of payload unmarshaling error: %w", err)
	}

	// 反序列化通道头
	if configUpdatePayload.Header == nil {
		return nil, fmt.Errorf("Failed initial chain config creation because config update header was missing")
	}
	channelHeader := &cb.ChannelHeader{}
	if err := channelHeader.Unmarshal(configUpdatePayload.Header.ChannelHeader); err != nil {
		return nil, fmt.Errorf("Failing initial chain config creation because of chain header unmarshaling error: %s", err)
	}

	// 反序列化配置更新结构
	configUpdateEnv := &cb.ConfigUpdateEnvelope{}
	if err := configUpdateEnv.Unmarshal(configUpdatePayload.Data); err != nil {
		return nil, fmt.Errorf("Failing initial chain config creation because of config update envelope unmarshaling error: %w", err)
	}
	configUpdate := &cb.ConfigUpdate{}
	if err := configUpdate.Unmarshal(configUpdateEnv.ConfigUpdate); err != nil {
		return nil, fmt.Errorf("Failing initial chain config creation because of config update unmarshaling error: %s", err)
	}

	// 配置更新中的链需要与消息中的链一致
	if configUpdate.ChannelId != channelHeader.ChannelId {
		return nil, fmt.Errorf("Failing initial chain config creation: mismatched chain IDs: '%s' != '%s'", configUpdate.ChannelId, channelHeader.ChannelId)
	}
	// 配置更新的写集不得为空
	if configUpdate.WriteSet == nil {
		return nil, fmt.Errorf("Config update has an empty writeset")
	}
	// 写集中必须要有应用配置组
	if configUpdate.WriteSet.Groups == nil || configUpdate.WriteSet.Groups[config.ApplicationGroupKey] == nil {
		return nil, fmt.Errorf("Config update has missing application group")
	}
	// 配置组的版本必须为1
	if uv := configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Version; uv != 1 {
		return nil, fmt.Errorf("Config update for chain creation does not set application group version to 1, was %d", uv)
	}

	// 反序列化共识模式名
	consortiumConfigValue, ok := configUpdate.WriteSet.Values[config.ConsortiumKey]
	if !ok {
		return nil, fmt.Errorf("Consortium config value missing")
	}
	consortium := &cb.Consortium{}
	if err := consortium.Unmarshal(consortiumConfigValue.Value); err != nil {
		return nil, fmt.Errorf("Error reading unmarshaling consortium name: %s", err)
	}

	// 获取当前系统链的共识配置
	consortiumsConfig, ok := m.systemChain.ConsortiumsConfig()
	if !ok {
		return nil, fmt.Errorf("The ordering system chain does not appear to support creating Chains")
	}

	// 获取该共识模式的配置
	name := consortium.Name
	consortiumConf, ok := consortiumsConfig.Consortiums()[name]
	if !ok {
		name = strings.ToLower(consortium.Name)
		consortiumConf, ok = consortiumsConfig.Consortiums()[name]
		if !ok {
			return nil, fmt.Errorf("Unknown consortium type: %s", name)
		}
	}

	// 创建新的应用配置组
	applicationGroup := cb.NewConfigGroup()
	applicationGroup.Policies[config.ChannelCreationPolicyKey] = &cb.ConfigPolicy{
		Policy: consortiumConf.ChannelCreationPolicy(),
	}
	applicationGroup.ModPolicy = config.ChannelCreationPolicyKey

	// 获取当前系统链的组配置
	systemChainGroup := m.systemChain.ConfigEnvelope().Config.ChannelGroup

	// 当系统链的共识组没有成员, 则允许应用组内也没有成员. 当共识组内有成员, 则应用组内至少有一个成员
	if len(systemChainGroup.Groups[config.ConsortiumsGroupKey].Groups[name].Groups) > 0 &&
		len(configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups) == 0 {
		return nil, fmt.Errorf("Proposed configuration has no application group members, but consortium contains members")
	}

	// 当系统链的共识组有成员, 应用组中的成员必须是共识组的子集
	if len(systemChainGroup.Groups[config.ConsortiumsGroupKey].Groups[name].Groups) > 0 {
		for orgName := range configUpdate.WriteSet.Groups[config.ApplicationGroupKey].Groups {
			consortiumGroup, ok := systemChainGroup.Groups[config.ConsortiumsGroupKey].Groups[name].Groups[orgName]
			if !ok {
				return nil, fmt.Errorf("Attempted to include a member which is not in the consortium")
			}
			// 加入共识组中的成员到应用组
			applicationGroup.Groups[orgName] = consortiumGroup
		}
	}

	// 创建新的链配置组, 复制系统链的配置及策略
	chainGroup := cb.NewConfigGroup()
	for key, value := range systemChainGroup.Values {
		chainGroup.Values[key] = value
	}
	for key, policy := range systemChainGroup.Policies {
		chainGroup.Policies[key] = policy
	}
	// 复制系统链的Orderer配置
	chainGroup.Groups[config.OrdererGroupKey] = systemChainGroup.Groups[config.OrdererGroupKey]
	// 加入上面新建的应用配置
	chainGroup.Groups[config.ApplicationGroupKey] = applicationGroup
	// 加入配置更新中的共识模式
	chainGroup.Values[config.ConsortiumKey] = config.TemplateConsortium(name).Values[config.ConsortiumKey]

	// 签名该链配置数据, 作为新链的创世块
	templateConfig, _ := utils.CreateSignedEnvelope(cb.HeaderType_CONFIG, configUpdate.ChannelId, m.signer, &cb.ConfigEnvelope{
		Config: &cb.Config{
			ChannelGroup: chainGroup,
		},
	})

	// 关闭一次策略管理器中提交提案的健全性检查
	initializer := configtx.NewInitializer()
	pm, ok := initializer.PolicyManager().(*policies.ManagerImpl)
	if ok {
		pm.SuppressSanityLogMessages = true
		defer func() {
			pm.SuppressSanityLogMessages = false
		}()
	}

	// 创建新的配置管理器
	return configtx.NewConfigManager(templateConfig, initializer, nil)
}

// NewChannelPolicyManagerGetter returns a new instance of ChannelPolicyManagerGetter
func NewChannelPolicyManagerGetter() policies.ChannelPolicyManagerGetter {
	return &channelPolicyManagerGetter{}
}

type channelPolicyManagerGetter struct{}

func (c *channelPolicyManagerGetter) Manager(channelID string) (policies.Manager, bool) {
	policyManager := GetPolicyManager(channelID)
	return policyManager, policyManager != nil
}

// GetPolicyManager returns the policy manager of the chain with chain ID. Note that this
// call returns nil if chain cid has not been created.
func GetPolicyManager(cid string) policies.Manager {
	cs, ok := chains.Load(cid)
	if !ok {
		return nil
	}
	return cs.(*Chain).PolicyManager()
}
