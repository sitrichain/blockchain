package conf

import (
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/Shopify/sarama"

	"github.com/rongzer/blockchain/common/bccsp/factory"
)

// 所有配置
type v struct {
	TLS                 TLS                  // TLS连接配置
	Profile             Profile              // 调试配置
	LogPath             string               // 日志存储路径
	LogLevel            string               // 日志级别
	FileSystemPath      string               // 存储数据的文件系统路径
	BCCSP               *factory.FactoryOpts // 加密服务配置
	Ledger              Ledger               // 账本配置
	Sealer              Sealer               // Sealer模块相关配置
	Peer                Peer                 // Peer模块相关配置
	AttachServerAddress string               // attach外部服务地址
}

// Peer Peer角色相关配置
type Peer struct {
	ListenAddress      string    // 服务地址:端口
	MSPDir             string    // MSP文件路径
	MSPID              string    // MSP ID
	ID                 string    // Peer实例身份标识
	NetworkId          string    // 网络ID逻辑分割网络
	AddressAutoDetect  bool      // 自动检测当前服务使用的地址
	Endpoint           string    // 自身节点的交互模块服务地址
	Events             Events    // 事件推送配置
	EndorserBufferSize int       // 可堆积的最大待背书提案数量
	ProcessEndorserNum int       // 可并发执行的背书提案最大数量
	VM                 VM        // 容器配置
	Chaincode          Chaincode // 链码配置
	IsLight            bool      //是否为轻量节点
}

// Gossip Gossip相关配置
type Gossip struct {
	Bootstrap                  []string      // 初始连接地址
	UseLeaderElection          bool          // 是否选举leader
	OrgLeader                  bool          // 是否为组织的leader, 可连接orderer并分发块
	Endpoint                   string        // 覆盖组织中的其他节点列表
	MaxBlockCountToStore       int           // 内存中存储块的最大数量
	MaxPropagationBurstLatency time.Duration // 连续消息推送的最大时间间隔
	MaxPropagationBurstSize    int           // 触发推送的最大消息数
	PropagateIterations        int           // 推送次数
	PropagatePeerNum           int           // 单次推送给节点数量
	PullInterval               time.Duration // 拉取间隔
	PullPeerNum                int           // 拉取来源节点数量
	RequestStateInfoInterval   time.Duration // 拉取状态信息消息间隔
	PublishStateInfoInterval   time.Duration // 推送状态信息消息间隔
	StateInfoRetentionInterval time.Duration // 状态消息最大过期时间
	PublishCertPeriod          time.Duration // 心跳消息中证书过期时间
	DialTimeout                time.Duration // 拨号超时
	ConnTimeout                time.Duration // 连接超时
	RecvBuffSize               int           // 接收消息Buffer的大小
	SendBuffSize               int           // 发送消息Buffer的大小
	DigestWaitTime             time.Duration // 处理摘要超时时间
	RequestWaitTime            time.Duration // 请求超时时间
	ResponseWaitTime           time.Duration // 拉取超时时间
	AliveTimeInterval          time.Duration // 心跳检查周期
	AliveExpirationTimeout     time.Duration // 心跳超时时间
	ReconnectInterval          time.Duration // 重连间隔时间
	ExternalEndpoint           string        // 可连接的组织外节点列表
	Election                   Election      // 选举配置
	SkipBlockVerification      bool          // 是否跳过块效验
	FromMembers                []string      // 轻量账本需要存储的交易的来源会员的证书
}

// Election 选举配置
type Election struct {
	StartupGracePeriod       time.Duration // 选举前等待其他成员的最长时间
	MembershipSampleInterval time.Duration // 为检查成员稳定性进行抽样的间隔
	LeaderAliveThreshold     time.Duration // 最近一次申明消息到节点决定选举前的时间
	LeaderElectionDuration   time.Duration // 节点发送选举消息的倒计时时间
}

// Events 事件推送配置
type Events struct {
	Address    string        // 服务地址:端口
	BufferSize int           // 可堆积的最大通知数量
	Timeout    time.Duration // 发送超时
}

// VM 容器配置
type VM struct {
	Endpoint         string             // 容器管理系统访问地址
	TLS              TLS                // TLS连接配置
	AttachStdout     bool               // 是否开启链码容器标准输出/错误, 用于调试
	ImageTag         string             // 镜像标签
	Hub              string             // 镜像仓库地址
	MemorySwappiness int64              //
	OOMKillDisable   bool               //
	HostConfig       *docker.HostConfig //
}

// Chaincode 链码配置
type Chaincode struct {
	ListenAddress  string        // 链码地址:端口
	PeerAddress    string        // 当链码目标解析因chaincodeListenAddress属性而失败时使用此配置
	Name           string        // 链码名称
	Path           string        // 链码安装路径
	JavaDockerfile string        // JAVA镜像名称
	CarRuntime     string        //
	GolangRuntime  string        //
	Startuptimeout time.Duration // 启动超时时间
	Executetimeout time.Duration // 执行超时时间
	Mode           string        // 运行模式: "dev", "net"
	Keepalive      time.Duration // 保活间隔时间, <=0可关闭
	EnableCSCC     bool          // 是否使用CSCC系统链码
	EnableLSCC     bool          // 是否使用LSCC系统链码
	EnableESCC     bool          // 是否使用ESCC系统链码
	EnableVSCC     bool          // 是否使用VSCC系统链码
	EnableQSCC     bool          // 是否使用QSCC系统链码
	WhiteList      []string      // 白名单
}

// Sealer Orderer角色相关配置
type Sealer struct {
	ListenAddress       string // 服务地址:端口
	MSPDir              string // MSP文件路径
	MSPID               string // MSP ID
	CaAddress           string // ca服务——http://地址:端口
	GenesisFile         string // 创世块文件名
	WhiteList           string // 背书白名单
	BlackList           string // 背书黑名单
	EndorseLimitPerPeer int    // 单节点每秒可被分配的最大背书数量
	Raft                Raft   // raft配置
	SystemChainId       string //系统链名称
	ParticipateConsensusOfSysChain bool // 该节点是否参与系统链的共识
}

// TLS TLS连接配置
type TLS struct {
	Enabled            bool     // 是否开启TLS
	PrivateKey         string   // 私钥
	Certificate        string   // 证书
	RootCAs            []string // 根CA证书
	ClientAuthEnabled  bool     // 客户端是否开启鉴权
	ClientRootCAs      []string // 客户端根CA证书
}

// Profile 调试配置 Go web pprof配置
type Profile struct {
	Enabled bool   // 是否开启pprof
	Address string // 远程访问地址:端口
}

// Ledger 账本配置
type Ledger struct {
	Location              string  // 存储路径
	Temp                  string  // 无存储路径配置时, 账本存在临时目录中的目录名
	StateDatabase         string  // 状态数据库类型: leveldb\rocksdb
	RocksDB               RocksDB // 如选用rocksdb时, rocksdb的配置
	EnableHistoryDatabase bool    // 是否开启历史数据库, 记录key的变更
}

// RocksDB RocksDB配置
type RocksDB struct {
	MaxBackgroundCompactions int // Compaction线程数
	MaxBackgroundFlushes     int // Flush线程数
	// 其他项
	BytesPerSync                    int
	BytesPerSecond                  int64
	StatsDumpPeriodSec              int
	MaxOpenFiles                    int
	CreateIfMissing                 bool
	Level0StopWritesTrigger         int
	HardPendingCompactionBytesLimit int
	Level0SlowdownWritesTrigger     int
	// Compaction相关
	NumLevels                        int     // 默认使用7层压缩
	Level0FileNumCompactionTrigger   int     // 当 level 0 的文件数据达到这个值的时候，就开始进行 level 0 到 level 1 的 compaction。 所以通常 level 0 的大小就是 write_buffer_size * min_write_buffer_number_to_merge * level0_file_num_compaction_trigger。
	MaxBytesForLevelBase             int     // max_bytes_for_level_base 就是 level1 的总大小，通常建议 level 1 跟 level 0 的 size 相当，所以直接用上面公式计算出的值即可，这里是1G。
	MaxBytesForLevelMultiplier       float64 // 上层的 level 的 size 每层都会比当前层大 max_bytes_for_level_multiplier 倍，这个值默认是 10，通常也不建议修改。
	TargetFileSizeBase               int     // level 1 SST 文件的 size，取 max_bytes_for_level_base / 10，也就是 level 1 会有 10 个 SST 文件。
	TargetFileSizeMultiplier         int     // 上层的文件 size 都会比当前层大 target_file_size_multiplier 倍，默认取10，所以level2-6每层也都有10个SST文件
	LevelCompactionDynamicLevelBytes bool
	// flush相关
	WriteBufferSize             int // memtable的最大size，如果超过了这个值，RocksDB就会将其变成immutable memtable，并在使用另一个新的memtable，默认128M
	MaxWriteBufferNumber        int // 总的memtable个数（active+immutable）达到这个阈值后，就会停止后续写入（即flush速度跟不上写入速度），因此调大此值，默认为20
	MinWriteBufferNumberToMerge int // immutable memtable被flush到level 0之前，最少需要被merge的个数，默认为2，最好不改，太大会影响读性能
}

// Kafka KAFKA配置
type Kafka struct {
	Brokers    []string            // Broker列表
	InitOffset bool                // 是否初始化偏移
	Retry      Retry               // 连接失败重试配置
	Verbose    bool                // 是否打印KAFKA日志
	Version    sarama.KafkaVersion // KAFKA版本
	TLS        TLS                 // TLS连接配置
}

type Raft struct {
	TickInterval         time.Duration // time interval between two Node.Tick invocations
	ElectionTick         int           // the number of Node.Tick invocations that must pass between elections. That is, if a follower does not receive any message from the leader of current term before ElectionTick has elapsed, it will become candidate and start an election. ElectionTick must be greater than HeartbeatTick.
	HeartbeatTick        int           // the number of Node.Tick invocations that must pass between heartbeats. That is, a leader sends heartbeat messages to maintain its leadership every HeartbeatTick ticks.
	MaxInflightBlocks    int           // limits the max number of in-flight append messages during optimistic replication phase.
	SnapshotIntervalSize uint32        // defines number of bytes per which a snapshot is taken
	WALDir               string        // WAL data of <my-channel> is stored in WALDir/<my-channel>
	SnapDir              string        // Snapshots of <my-channel> are stored in SnapDir/<my-channel>
	EvictionSuspicion    string        // Duration threshold that the node samples in order to suspect its eviction from the channel.
	// cluster相关配置
	EndPoint                string
	BootStrapEndPoint       string
	DialTimeout             time.Duration
	RPCTimeout              time.Duration
	ReplicationBufferSize   int
	ReplicationPullTimeout  time.Duration
	ReplicationRetryTimeout time.Duration
	ReplicationMaxRetries   uint64
}

// Retry KAFKA连接失败重试配置
type Retry struct {
	ShortInterval   time.Duration
	ShortTotal      time.Duration
	LongInterval    time.Duration
	LongTotal       time.Duration
	NetworkTimeouts NetworkTimeouts // 网络请求超时时间
	Metadata        Metadata        // 元数据请求配置
	Producer        Producer        // 生产者请求配置
	Consumer        Consumer        // 消费者请求配置
}

// NetworkTimeouts 网络请求超时时间
type NetworkTimeouts struct {
	DialTimeout  time.Duration // 连接超时
	ReadTimeout  time.Duration // 读取超时
	WriteTimeout time.Duration // 写入超时
}

// Metadata 元数据请求配置
type Metadata struct {
	RetryMax     int           // 最大重试次数
	RetryBackoff time.Duration // 重试间隔
}

// Producer 生产者请求配置
type Producer struct {
	RetryMax     int           // 最大重试次数
	RetryBackoff time.Duration // 重试间隔
}

// Consumer 消费者请求配置
type Consumer struct {
	RetryBackoff time.Duration // 重试间隔
}
