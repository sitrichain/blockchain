package raft

import (
	"context"
	"encoding/pem"
	"fmt"
	"github.com/rongzer/blockchain/common/conf"
	"github.com/rongzer/blockchain/peer/consensus"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/clock"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/cluster"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/peer/filters"
	"github.com/rongzer/blockchain/protos/common"
	"github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/orderer/etcdraft"
	"github.com/rongzer/blockchain/protos/utils"
	"go.etcd.io/etcd/raft"
	"go.etcd.io/etcd/raft/raftpb"
	"go.etcd.io/etcd/wal"
)

const (
	BYTE = 1 << (10 * iota)
	KILOBYTE
	MEGABYTE
	GIGABYTE
	TERABYTE
)

const (
	// DefaultSnapshotCatchUpEntries is the default number of entries
	// to preserve in memory when a snapshot is taken. This is for
	// slow followers to catch up.
	DefaultSnapshotCatchUpEntries = uint64(4)

	// DefaultSnapshotIntervalSize is the default snapshot interval. It is
	// used if SnapshotIntervalSize is not provided in channel config options.
	// It is needed to enforce snapshot being set.
	DefaultSnapshotIntervalSize = 16 * MEGABYTE

	// DefaultEvictionSuspicion is the threshold that a node will start
	// suspecting its own eviction if it has been leaderless for this
	// period of time.
	DefaultEvictionSuspicion = time.Minute * 10

	// DefaultLeaderlessCheckInterval is the interval that a chain checks
	// its own leadership status.
	DefaultLeaderlessCheckInterval = time.Second * 10
)

//go:generate counterfeiter -o mocks/configurator.go . Configurator

// Configurator is used to configure the communication layer
// when the chain starts.
type Configurator interface {
	Configure(channel string, newNodes []*cluster.RemoteNode)
}

//go:generate counterfeiter -o mocks/mock_rpc.go . RPC

// RPC is used to mock the transport layer in tests.
type RPC interface {
	SendConsensus(dest uint64, msg *orderer.ConsensusRequest) error
	SendSubmit(dest uint64, request *orderer.SubmitRequest) error
}

//go:generate counterfeiter -o mocks/mock_blockpuller.go . BlockPuller

// BlockPuller is used to pull blocks from other OSN
type BlockPuller interface {
	PullBlock(seq uint64) *common.Block
	HeightsByEndpoints() (map[string]uint64, error)
	Close()
}

// CreateBlockPuller is a function to create BlockPuller on demand.
// It is passed into chain initializer so that tests could mock this.
type CreateBlockPuller func() (BlockPuller, error)

// Options contains all the configurations relevant to the chain.
type Options struct {
	RaftID uint64

	Clock clock.Clock

	WALDir               string
	SnapDir              string
	SnapshotIntervalSize uint32

	// This is configurable mainly for testing purpose. Users are not
	// expected to alter this. Instead, DefaultSnapshotCatchUpEntries is used.
	SnapshotCatchUpEntries uint64

	MemoryStorage MemoryStorage
	Logger        *log.RaftLogger

	TickInterval      time.Duration
	ElectionTick      int
	HeartbeatTick     int
	MaxSizePerMsg     uint64
	MaxInflightBlocks int

	// BlockMetdata and Consenters should only be modified while under lock
	// of raftMetadataLock
	BlockMetadata *etcdraft.BlockMetadata
	Consenters    map[uint64]*etcdraft.Consenter

	LeaderCheckInterval time.Duration
}

type submit struct {
	req    *orderer.SubmitRequest
	leader chan uint64
}

type gc struct {
	index uint64
	state raftpb.ConfState
	data  []byte
}

// Consenter implements consensus.Consenter interface.
type Consenter struct {
	configurator Configurator

	rpc RPC

	raftID    uint64
	channelID string

	lastKnownLeader uint64
	ActiveNodes     atomic.Value

	submitC  chan *submit
	applyC   chan apply
	observeC chan<- raft.SoftState // Notifies external observer on leader change (passed in optionally as an argument for tests)
	haltC    chan struct{}         // Signals to goroutines that the chain is halting
	doneC    chan struct{}         // Closes when the chain halts
	startC   chan struct{}         // Closes when the node is started
	snapC    chan *raftpb.Snapshot // Signal to catch up with snapshot
	gcC      chan *gc              // Signal to take snapshot

	errorCLock sync.RWMutex
	errorC     chan struct{} // returned by Errored()

	raftMetadataLock     sync.RWMutex
	confChangeInProgress *raftpb.ConfChange
	justElected          bool // this is true when node has just been elected
	configInflight       bool // this is true when there is config block or ConfChange in flight
	blockInflight        int  // number of in flight blocks

	clock clock.Clock // Tests can inject a fake clock

	support consensus.ChainResource

	lastBlock    *common.Block
	appliedIndex uint64

	// needed by snapshotting
	sizeLimit        uint32 // SnapshotIntervalSize in bytes
	accDataSize      uint32 // accumulative data size since last snapshot
	lastSnapBlockNum uint64
	confState        raftpb.ConfState // Etcdraft requires ConfState to be persisted within snapshot

	createPuller CreateBlockPuller // func used to create BlockPuller on demand

	fresh bool // indicate if this is a fresh raft node

	// this is exported so that test can use `Node.Status()` to get raft node status.
	Node *node
	Opts Options

	logger *log.RaftLogger

	haltCallback func()
	// BCCSP instane
	CryptoProvider bccsp.BCCSP
}

type SendSubmitParams struct {
	destId  uint64
	request *orderer.SubmitRequest
}

var SendSubmitMap sync.Map //map[string]*SendSubmitParams{}

// NewChain constructs a chain object.
func NewChain(
	support consensus.ChainResource,
	envelope *common.Envelope,
	opts Options,
	configure Configurator,
	rpc RPC,
	cryptoProvider bccsp.BCCSP,
	f CreateBlockPuller,
	observeC chan<- raft.SoftState,
) (*Consenter, error) {
	// 若有新节点加入，即envelope不为空，则需向cluster中的任意节点发送rpc请求；
	// 发送的rpc请求为SubmitRequest，即将更新后的*etcdraft.ConfigMetadata以envelope的形式发给raft集群
	if envelope != nil {
		// 找到BootStrapEndPoint对应的raftId
		destId, err := determineIdFromEndpoint(opts.Consenters, conf.V.Sealer.Raft.BootStrapEndPoint)
		if err != nil {
			return nil, err
		}
		// 构造*orderer.SubmitRequest
		nodeInfo := &orderer.NodeInfo{Id: opts.RaftID, EndPoint: conf.V.Sealer.Raft.EndPoint}
		request := &orderer.SubmitRequest{Payload: envelope, Channel: support.ChainID(), SourceNodeInfo: nodeInfo}
		// 组装为SendSubmitParams后，塞入SendSubmitMap
		SendSubmitMap.Store(support.ChainID(), &SendSubmitParams{destId: destId, request: request})
	}

	lg := log.NewRaftLogger(log.Logger).With("channel", support.ChainID(), "node", opts.RaftID)

	fresh := !wal.Exist(opts.WALDir)
	storage, err := CreateStorage(lg, opts.WALDir, opts.SnapDir, opts.MemoryStorage)
	if err != nil {
		return nil, errors.Errorf("failed to restore persisted raft data: %s", err)
	}

	if opts.SnapshotCatchUpEntries == 0 {
		storage.SnapshotCatchUpEntries = DefaultSnapshotCatchUpEntries
	} else {
		storage.SnapshotCatchUpEntries = opts.SnapshotCatchUpEntries
	}

	sizeLimit := opts.SnapshotIntervalSize
	if sizeLimit == 0 {
		sizeLimit = DefaultSnapshotIntervalSize
	}

	// get block number in last snapshot, if exists
	var snapBlkNum uint64
	var cc raftpb.ConfState
	if s := storage.Snapshot(); !raft.IsEmptySnap(s) {
		b, err := utils.GetBlockFromBlockBytes(s.Data)
		if err != nil {
			return nil, errors.Errorf("failed to unmarshal block from bytes")
		}
		snapBlkNum = b.Header.Number
		cc = s.Metadata.ConfState
	}

	b, err := support.GetBlockByNumber(support.Height() - 1)
	if b == nil || err != nil {
		return nil, errors.Errorf("failed to get last block")
	}

	c := &Consenter{
		configurator:     configure,
		rpc:              rpc,
		channelID:        support.ChainID(),
		raftID:           opts.RaftID,
		submitC:          make(chan *submit),
		applyC:           make(chan apply),
		haltC:            make(chan struct{}),
		doneC:            make(chan struct{}),
		startC:           make(chan struct{}),
		snapC:            make(chan *raftpb.Snapshot),
		errorC:           make(chan struct{}),
		gcC:              make(chan *gc),
		observeC:         observeC,
		support:          support,
		fresh:            fresh,
		appliedIndex:     opts.BlockMetadata.RaftIndex,
		lastBlock:        b,
		sizeLimit:        sizeLimit,
		lastSnapBlockNum: snapBlkNum,
		confState:        cc,
		createPuller:     f,
		clock:            opts.Clock,
		logger:           lg,
		Opts:             opts,
		CryptoProvider:   cryptoProvider,
	}

	// DO NOT use Applied option in config, see https://github.com/etcd-io/etcd/issues/10217
	// We guard against replay of written blocks with `appliedIndex` instead.
	config := &raft.Config{
		ID:              c.raftID,
		ElectionTick:    c.Opts.ElectionTick,
		HeartbeatTick:   c.Opts.HeartbeatTick,
		MaxSizePerMsg:   c.Opts.MaxSizePerMsg,
		MaxInflightMsgs: c.Opts.MaxInflightBlocks,
		Logger:          c.logger,
		Storage:         c.Opts.MemoryStorage,
		// PreVote prevents reconnected node from disturbing network.
		// See etcd/raft doc for more details.
		PreVote:                   true,
		CheckQuorum:               true,
		DisableProposalForwarding: true, // This prevents blocks from being accidentally proposed by followers
	}

	disseminator := &Disseminator{RPC: c.rpc}
	disseminator.UpdateMetadata(nil) // initialize
	c.ActiveNodes.Store([]uint64{})

	c.Node = &node{
		chainID:      c.channelID,
		chain:        c,
		logger:       c.logger,
		storage:      storage,
		rpc:          disseminator,
		config:       config,
		tickInterval: c.Opts.TickInterval,
		clock:        c.clock,
		metadata:     c.Opts.BlockMetadata,
		tracker: &Tracker{
			id:     c.raftID,
			sender: disseminator,
			active: &c.ActiveNodes,
			logger: c.logger,
		},
	}

	return c, nil
}

// Start instructs the orderer to begin serving the chain and keep it current.
func (c *Consenter) Start() {
	c.logger.Infof("Starting Raft node")

	if err := c.configureComm(); err != nil {
		c.logger.Errorf("Failed to start chain, aborting: +%v", err)
		close(c.doneC)
		return
	}

	isJoin := c.support.Height() > 1
	if isJoin {
		isJoin = false
		c.logger.Infof("Consensus-type migration detected, starting new raft node on an existing channel; height=%d", c.support.Height())
	}
	c.Node.start(c.fresh, isJoin)

	close(c.startC)
	close(c.errorC)

	go c.gc()
	go c.run()

	// 发送rpc请求
	if v, ok := SendSubmitMap.Load(c.channelID); ok {
		params := v.(*SendSubmitParams)
		if err := c.rpc.SendSubmit(params.destId, params.request); err != nil {
			log.Logger.Errorf("cannot send configUpdate rpc request to %v, err is: %v", params.destId, err)
		}
	}
}

// Order submits normal type transactions for ordering.
func (c *Consenter) Order(env *common.Envelope, committer filters.Committer) bool {
	if err := c.Submit(&orderer.SubmitRequest{Payload: env, Channel: c.channelID}, 0); err != nil {
		return false
	}
	return true
}

// Errored returns a channel that closes when the chain stops.
func (c *Consenter) Errored() <-chan struct{} {
	c.errorCLock.RLock()
	defer c.errorCLock.RUnlock()
	return c.errorC
}

// Halt stops the chain.
func (c *Consenter) Halt() {
	select {
	case <-c.startC:
	default:
		c.logger.Warnf("Attempted to halt a chain that has not started")
		return
	}

	select {
	case c.haltC <- struct{}{}:
	case <-c.doneC:
		return
	}
	<-c.doneC

	if c.haltCallback != nil {
		c.haltCallback()
	}
}

func (c *Consenter) isRunning() error {
	select {
	case <-c.startC:
	default:
		return errors.Errorf("chain is not started")
	}

	select {
	case <-c.doneC:
		return errors.Errorf("chain is stopped")
	default:
	}

	return nil
}

// Consensus passes the given ConsensusRequest message to the raft.Node instance
func (c *Consenter) Consensus(req *orderer.ConsensusRequest, sender uint64) error {
	if err := c.isRunning(); err != nil {
		return err
	}

	stepMsg := &raftpb.Message{}
	if err := proto.Unmarshal(req.Payload, stepMsg); err != nil {
		return fmt.Errorf("failed to unmarshal StepRequest payload to Raft Message: %s", err)
	}
	if err := c.Node.Step(context.TODO(), *stepMsg); err != nil {
		return fmt.Errorf("failed to process Raft Step message: %s", err)
	}
	if len(req.Metadata) == 0 || atomic.LoadUint64(&c.lastKnownLeader) != sender { // ignore metadata from non-leader
		return nil
	}

	clusterMetadata := &etcdraft.ClusterMetadata{}
	if err := proto.Unmarshal(req.Metadata, clusterMetadata); err != nil {
		return errors.Errorf("failed to unmarshal ClusterMetadata: %s", err)
	}

	c.ActiveNodes.Store(clusterMetadata.ActiveNodes)

	return nil
}

// Submit forwards the incoming request to:
// - the local run goroutine if this is leader
// - the actual leader via the transport mechanism
// The call fails if there's no leader elected yet.
func (c *Consenter) Submit(req *orderer.SubmitRequest, sender uint64) error {
	if err := c.isRunning(); err != nil {
		return err
	}

	leadC := make(chan uint64, 1)
	select {
	case c.submitC <- &submit{req, leadC}:
		lead := <-leadC
		if lead == raft.None {
			return fmt.Errorf("no Raft leader")
		}

		if lead != c.raftID {
			req.SourceNodeInfo = &orderer.NodeInfo{Id: c.raftID, EndPoint: conf.V.Sealer.Raft.EndPoint}
			if err := c.rpc.SendSubmit(lead, req); err != nil {
				return err
			}
		}

	case <-c.doneC:
		return fmt.Errorf("chain is stopped")
	}

	return nil
}

type apply struct {
	entries []raftpb.Entry
	soft    *raft.SoftState
}

func isCandidate(state raft.StateType) bool {
	return state == raft.StatePreCandidate || state == raft.StateCandidate
}

func (c *Consenter) run() {
	ticking := false
	timer := c.clock.NewTimer(time.Second)
	// we need a stopped timer rather than nil,
	// because we will be select waiting on timer.C()
	if !timer.Stop() {
		<-timer.C()
	}

	// if timer is already started, this is a no-op
	startTimer := func() {
		if !ticking {
			ticking = true
			timer.Reset(c.support.SharedConfig().BatchTimeout())
		}
	}

	stopTimer := func() {
		if !timer.Stop() && ticking {
			// we only need to drain the channel if the timer expired (not explicitly stopped)
			<-timer.C()
		}
		ticking = false
	}

	var soft raft.SoftState
	submitC := c.submitC
	var bc *blockCreator

	var propC chan<- *common.Block
	var cancelProp context.CancelFunc
	cancelProp = func() {} // no-op as initial value

	becomeLeader := func() (chan<- *common.Block, context.CancelFunc) {

		c.blockInflight = 0
		c.justElected = true
		submitC = nil
		ch := make(chan *common.Block, c.Opts.MaxInflightBlocks)

		// if there is unfinished ConfChange, we should resume the effort to propose it as
		// new leader, and wait for it to be committed before start serving new requests.
		if cc := c.getInFlightConfChange(); cc != nil {
			// The reason `ProposeConfChange` should be called in go routine is documented in `writeConfigBlock` method.
			go func() {
				if err := c.Node.ProposeConfChange(context.TODO(), *cc); err != nil {
					c.logger.Warnf("Failed to propose configuration update to Raft node: %s", err)
				}
			}()

			c.confChangeInProgress = cc
			c.configInflight = true
		}

		// Leader should call Propose in go routine, because this method may be blocked
		// if node is leaderless (this can happen when leader steps down in a heavily
		// loaded network). We need to make sure applyC can still be consumed properly.
		ctx, cancel := context.WithCancel(context.Background())
		go func(ctx context.Context, ch <-chan *common.Block) {
			for {
				select {
				case b := <-ch:
					data := utils.MarshalOrPanic(b)
					if err := c.Node.Propose(ctx, data); err != nil {
						c.logger.Errorf("Failed to propose block [%d] to raft and discard %d blocks in queue: %s", b.Header.Number, len(ch), err)
						return
					}
					c.logger.Debugf("Proposed block [%d] to raft consensus", b.Header.Number)

				case <-ctx.Done():
					c.logger.Debugf("Quit proposing blocks, discarded %d blocks in the queue", len(ch))
					return
				}
			}
		}(ctx, ch)

		return ch, cancel
	}

	becomeFollower := func() {
		cancelProp()
		c.blockInflight = 0
		c.support.BlockCutter().Cut()
		stopTimer()
		submitC = c.submitC
		bc = nil
	}

	for {
		select {
		case s := <-submitC:
			if s == nil {
				// polled by `WaitReady`
				continue
			}

			if soft.RaftState == raft.StatePreCandidate || soft.RaftState == raft.StateCandidate {
				s.leader <- raft.None
				continue
			}

			s.leader <- soft.Lead
			if soft.Lead != c.raftID {
				continue
			}

			batches, pending, err := c.ordered(s.req)
			if err != nil {
				c.logger.Errorf("Failed to order message: %s", err)
				continue
			}
			if pending {
				startTimer() // no-op if timer is already started
			} else {
				stopTimer()
			}

			c.propose(propC, bc, batches...)

			if c.configInflight {
				c.logger.Info("Received config transaction, pause accepting transaction till it is committed")
				submitC = nil
			} else if c.blockInflight >= c.Opts.MaxInflightBlocks {
				c.logger.Debugf("Number of in-flight blocks (%d) reaches limit (%d), pause accepting transaction",
					c.blockInflight, c.Opts.MaxInflightBlocks)
				submitC = nil
			}

		case app := <-c.applyC:
			if app.soft != nil {
				newLeader := atomic.LoadUint64(&app.soft.Lead) // etcdraft requires atomic access
				if newLeader != soft.Lead {
					c.logger.Infof("Raft leader changed: %d -> %d", soft.Lead, newLeader)

					atomic.StoreUint64(&c.lastKnownLeader, newLeader)

					if newLeader == c.raftID {
						propC, cancelProp = becomeLeader()
					}

					if soft.Lead == c.raftID {
						becomeFollower()
					}
				}

				foundLeader := soft.Lead == raft.None && newLeader != raft.None
				quitCandidate := isCandidate(soft.RaftState) && !isCandidate(app.soft.RaftState)

				if foundLeader || quitCandidate {
					c.errorCLock.Lock()
					c.errorC = make(chan struct{})
					c.errorCLock.Unlock()
				}

				if isCandidate(app.soft.RaftState) || newLeader == raft.None {
					atomic.StoreUint64(&c.lastKnownLeader, raft.None)
					select {
					case <-c.errorC:
					default:
						nodeCount := len(c.Opts.BlockMetadata.ConsenterEndpoints)
						// Only close the error channel (to signal the broadcast/deliver front-end a consensus backend error)
						// If we are a cluster of size 3 or more, otherwise we can't expand a cluster of size 1 to 2 nodes.
						if nodeCount > 2 {
							close(c.errorC)
						} else {
							c.logger.Errorf("No leader is present, cluster size is %d", nodeCount)
						}
					}
				}

				soft = raft.SoftState{Lead: newLeader, RaftState: app.soft.RaftState}

				// notify external observer
				select {
				case c.observeC <- soft:
				default:
				}
			}

			c.apply(app.entries)

			if c.justElected {
				msgInflight := c.Node.lastIndex() > c.appliedIndex
				if msgInflight {
					c.logger.Debugf("There are in flight blocks, new leader should not serve requests")
					continue
				}

				if c.configInflight {
					c.logger.Debugf("There is config block in flight, new leader should not serve requests")
					continue
				}

				c.logger.Infof("Start accepting requests as Raft leader at block [%d]", c.lastBlock.Header.Number)
				bc = &blockCreator{
					hash:   utils.BlockHeaderHash(c.lastBlock.Header),
					number: c.lastBlock.Header.Number,
					logger: c.logger,
				}
				submitC = c.submitC
				c.justElected = false
			} else if c.configInflight {
				c.logger.Info("Config block or ConfChange in flight, pause accepting transaction")
				submitC = nil
			} else if c.blockInflight < c.Opts.MaxInflightBlocks {
				submitC = c.submitC
			}

		case <-timer.C():
			ticking = false

			batch, _ := c.support.BlockCutter().Cut()
			if len(batch) == 0 {
				c.logger.Warningf("Batch timer expired with no pending requests, this might indicate a bug")
				continue
			}

			c.logger.Debugf("Batch timer expired, creating block")
			c.propose(propC, bc, batch) // we are certain this is normal block, no need to block

		case sn := <-c.snapC:
			if sn.Metadata.Index != 0 {
				if sn.Metadata.Index <= c.appliedIndex {
					c.logger.Debugf("Skip snapshot taken at index %d, because it is behind current applied index %d", sn.Metadata.Index, c.appliedIndex)
					break
				}

				c.confState = sn.Metadata.ConfState
				c.appliedIndex = sn.Metadata.Index
			} else {
				c.logger.Infof("Received artificial snapshot to trigger catchup")
			}

			if err := c.catchUp(sn); err != nil {
				c.logger.Panicf("Failed to recover from snapshot taken at Term %d and Index %d: %s",
					sn.Metadata.Term, sn.Metadata.Index, err)
			}

		case <-c.doneC:
			stopTimer()
			cancelProp()

			select {
			case <-c.errorC: // avoid closing closed channel
			default:
				close(c.errorC)
			}

			c.logger.Infof("Stop serving requests")
			return
		}
	}
}

func (c *Consenter) writeBlock(block *common.Block, index uint64) {
	if block.Header.Number > c.lastBlock.Header.Number+1 {
		c.logger.Panicf("Got block [%d], expect block [%d]", block.Header.Number, c.lastBlock.Header.Number+1)
	} else if block.Header.Number < c.lastBlock.Header.Number+1 {
		c.logger.Infof("Got block [%d], expect block [%d], this node was forced to catch up", block.Header.Number, c.lastBlock.Header.Number+1)
		return
	}

	if c.blockInflight > 0 {
		c.blockInflight-- // only reduce on leader
	}
	c.lastBlock = block

	c.logger.Infof("Writing block [%d] (Raft index: %d) to ledger", block.Header.Number, index)

	if utils.IsConfigBlock(block) {
		c.writeConfigBlock(block, index)
		return
	}

	c.raftMetadataLock.Lock()
	c.Opts.BlockMetadata.RaftIndex = index
	m := utils.MarshalOrPanic(c.Opts.BlockMetadata)
	c.raftMetadataLock.Unlock()

	c.support.WriteBlock(block, m)
}

// Orders the envelope in the `msg` content. SubmitRequest.
// Returns
//   -- batches [][]*common.Envelope; the batches cut,
//   -- pending bool; if there are envelopes pending to be ordered,
//   -- err error; the error encountered, if any.
// It takes care of config messages as well as the revalidation of messages if the config sequence has advanced.
func (c *Consenter) ordered(msg *orderer.SubmitRequest) (batches [][]*common.Envelope, pending bool, err error) {
	if c.isConfig(msg.Payload) {
		// 配置相关交易独立成块
		//batch, _ := c.support.BlockCutter().Cut()
		batches = [][]*common.Envelope{}
		//if len(batch) != 0 {
		//	batches = append(batches, batch)
		//}
		batches = append(batches, []*common.Envelope{msg.Payload})
		return batches, false, nil
	}

	batches, _, pending = c.support.BlockCutter().Ordered(msg.Payload, nil)
	return batches, pending, nil

}

func (c *Consenter) propose(ch chan<- *common.Block, bc *blockCreator, batches ...[]*common.Envelope) {
	for _, batch := range batches {
		b := bc.createNextBlock(batch)
		c.logger.Infof("Created block [%d], there are %d blocks in flight", b.Header.Number, c.blockInflight)

		select {
		case ch <- b:
		default:
			c.logger.Panic("Programming error: limit of in-flight blocks does not properly take effect or block is proposed by follower")
		}

		// if it is config block, then we should wait for the commit of the block
		if utils.IsConfigBlock(b) {
			c.configInflight = true
		}

		c.blockInflight++
	}

	return
}

func (c *Consenter) catchUp(snap *raftpb.Snapshot) error {
	b, err := utils.GetBlockFromBlockBytes(snap.Data)
	if err != nil {
		return errors.Errorf("failed to unmarshal snapshot data to block: %s", err)
	}

	if c.lastBlock.Header.Number >= b.Header.Number {
		c.logger.Warnf("Snapshot is at block [%d], local block number is %d, no sync needed", b.Header.Number, c.lastBlock.Header.Number)
		return nil
	}

	puller, err := c.createPuller()
	if err != nil {
		return errors.Errorf("failed to create block puller: %s", err)
	}
	defer puller.Close()

	next := c.lastBlock.Header.Number + 1

	c.logger.Infof("Catching up with snapshot taken at block [%d], starting from block [%d]", b.Header.Number, next)

	for next <= b.Header.Number {
		block := puller.PullBlock(next)
		if block == nil {
			return errors.Errorf("failed to fetch block [%d] from cluster", next)
		}
		if utils.IsConfigBlock(block) {
			c.support.WriteBlock(block, nil)

			configMembership := c.detectConfChange(block)

			if configMembership != nil && configMembership.Changed() {
				c.logger.Infof("Config block [%d] changes Consenter set, communication should be reconfigured", block.Header.Number)

				c.raftMetadataLock.Lock()
				c.Opts.BlockMetadata = configMembership.NewBlockMetadata
				c.Opts.Consenters = configMembership.NewConsenters
				c.raftMetadataLock.Unlock()

				if err := c.configureComm(); err != nil {
					c.logger.Panicf("Failed to configure communication: %s", err)
				}
			}
		} else {
			c.support.WriteBlock(block, nil)
		}

		c.lastBlock = block
		next++
	}

	c.logger.Infof("Finished syncing with cluster up to and including block [%d]", b.Header.Number)
	return nil
}

func (c *Consenter) detectConfChange(block *common.Block) *MembershipChanges {
	// If config is targeting THIS channel, inspect Consenter set and
	// propose raft ConfChange if it adds/removes node.
	configMetadata := c.newConfigMetadata(block)

	if configMetadata == nil {
		return nil
	}

	if configMetadata.Options != nil &&
		configMetadata.Options.SnapshotIntervalSize != 0 &&
		configMetadata.Options.SnapshotIntervalSize != c.sizeLimit {
		c.logger.Infof("Update snapshot interval size to %d bytes (was %d)",
			configMetadata.Options.SnapshotIntervalSize, c.sizeLimit)
		c.sizeLimit = configMetadata.Options.SnapshotIntervalSize
	}

	changes, err := ComputeMembershipChanges(c.Opts.BlockMetadata, c.Opts.Consenters, configMetadata.Consenters)
	if err != nil {
		c.logger.Panicf("illegal configuration change detected: %s", err)
	}

	if changes.Rotated() {
		c.logger.Infof("Config block [%d] rotates TLS certificate of node %d", block.Header.Number, changes.RotatedNode)
	}

	return changes
}

func (c *Consenter) apply(ents []raftpb.Entry) {
	if len(ents) == 0 {
		return
	}

	if ents[0].Index > c.appliedIndex+1 {
		c.logger.Panicf("first index of committed entry[%d] should <= appliedIndex[%d]+1", ents[0].Index, c.appliedIndex)
	}

	var position int
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal:
			if len(ents[i].Data) == 0 {
				break
			}

			position = i
			c.accDataSize += uint32(len(ents[i].Data))

			// We need to strictly avoid re-applying normal entries,
			// otherwise we are writing the same block twice.
			if ents[i].Index <= c.appliedIndex {
				c.logger.Infof("Received block with raft index (%d) <= applied index (%d), skip", ents[i].Index, c.appliedIndex)
				break
			}

			block, err := utils.GetBlockFromBlockBytes(ents[i].Data)
			if err != nil {
				c.logger.Error("unmarshal block err from ents[i].Data")
				continue
			}
			c.writeBlock(block, ents[i].Index)

		case raftpb.EntryConfChange:
			var cc raftpb.ConfChange
			if err := cc.Unmarshal(ents[i].Data); err != nil {
				c.logger.Warnf("Failed to unmarshal ConfChange data: %s", err)
				continue
			}

			c.confState = *c.Node.ApplyConfChange(cc)

			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				c.logger.Infof("Applied config change to add node %d, current nodes in channel: %+v", cc.NodeID, c.confState.Voters)
			case raftpb.ConfChangeRemoveNode:
				c.logger.Infof("Applied config change to remove node %d, current nodes in channel: %+v", cc.NodeID, c.confState.Voters)
			default:
				c.logger.Panic("Programming error, encountered unsupported raft config change")
			}

			// This ConfChange was introduced by a previously committed config block,
			// we can now unblock submitC to accept envelopes.
			var configureComm bool
			if c.confChangeInProgress != nil &&
				c.confChangeInProgress.NodeID == cc.NodeID &&
				c.confChangeInProgress.Type == cc.Type {

				configureComm = true
				c.confChangeInProgress = nil
				c.configInflight = false
			}

			lead := atomic.LoadUint64(&c.lastKnownLeader)
			removeLeader := cc.Type == raftpb.ConfChangeRemoveNode && cc.NodeID == lead
			shouldHalt := cc.Type == raftpb.ConfChangeRemoveNode && cc.NodeID == c.raftID

			// unblock `run` go routine so it can still consume Raft messages
			go func() {
				if removeLeader {
					c.logger.Infof("Current leader is being removed from channel, attempt leadership transfer")
					c.Node.abdicateLeader(lead)
				}

				if configureComm && !shouldHalt { // no need to configure comm if this node is going to halt
					if err := c.configureComm(); err != nil {
						c.logger.Panicf("Failed to configure communication: %s", err)
					}
				}

				if shouldHalt {
					c.logger.Infof("This node is being removed from replica set")
					c.Halt()
					return
				}
			}()
		}

		if ents[i].Index > c.appliedIndex {
			c.appliedIndex = ents[i].Index
		}
	}

	if c.accDataSize >= c.sizeLimit {
		b, err := utils.GetBlockFromBlockBytes(ents[position].Data)
		if err != nil {
			c.logger.Error("unmarshal block err from ents[position].Data")
			return
		}
		select {
		case c.gcC <- &gc{index: c.appliedIndex, state: c.confState, data: ents[position].Data}:
			c.logger.Infof("Accumulated %d bytes since last snapshot, exceeding size limit (%d bytes), "+
				"taking snapshot at block [%d] (index: %d), last snapshotted block number is %d, current nodes: %+v",
				c.accDataSize, c.sizeLimit, b.Header.Number, c.appliedIndex, c.lastSnapBlockNum, c.confState.Voters)
			c.accDataSize = 0
			c.lastSnapBlockNum = b.Header.Number
		default:
			c.logger.Warnf("Snapshotting is in progress, it is very likely that SnapshotIntervalSize is too small")
		}
	}

	return
}

func (c *Consenter) gc() {
	for {
		select {
		case g := <-c.gcC:
			c.Node.takeSnapshot(g.index, g.state, g.data)
		case <-c.doneC:
			c.logger.Infof("Stop garbage collecting")
			return
		}
	}
}

func (c *Consenter) isConfig(env *common.Envelope) bool {
	h, err := utils.ChannelHeader(env)
	if err != nil {
		c.logger.Panicf("failed to extract channel header from envelope")
	}

	return h.Type == int32(common.HeaderType_CONFIG) || h.Type == int32(common.HeaderType_ORDERER_TRANSACTION)
}

func (c *Consenter) configureComm() error {
	// Reset unreachable map when communication is reconfigured
	c.Node.unreachableLock.Lock()
	c.Node.unreachable = make(map[uint64]struct{})
	c.Node.unreachableLock.Unlock()

	nodes, err := c.remotePeers()
	if err != nil {
		return err
	}
	c.configurator.Configure(c.channelID, nodes)
	return nil
}

func (c *Consenter) remotePeers() ([]*cluster.RemoteNode, error) {
	c.raftMetadataLock.RLock()
	defer c.raftMetadataLock.RUnlock()

	var nodes []*cluster.RemoteNode
	for raftID, consenter := range c.Opts.Consenters {
		// No need to know yourself
		if raftID == c.raftID {
			continue
		}
		nodes = append(nodes, &cluster.RemoteNode{
			ID:       raftID,
			Endpoint: fmt.Sprintf("%s:%d", consenter.Host, consenter.Port),
		})
	}
	return nodes, nil
}

func pemToDER(pemBytes []byte, id uint64, certType string, logger *log.RaftLogger) ([]byte, error) {
	bl, _ := pem.Decode(pemBytes)
	if bl == nil {
		logger.Errorf("Rejecting PEM block of %s TLS cert for node %d, offending PEM is: %s", certType, id, string(pemBytes))
		return nil, errors.Errorf("invalid PEM block")
	}
	return bl.Bytes, nil
}

// writeConfigBlock writes configuration blocks into the ledger in
// addition extracts updates about raft replica set and if there
// are changes updates cluster membership as well
func (c *Consenter) writeConfigBlock(block *common.Block, index uint64) {
	hdr, err := ConfigChannelHeader(block)
	if err != nil {
		c.logger.Panicf("Failed to get config header type from config block: %s", err)
	}

	c.configInflight = false

	switch common.HeaderType(hdr.Type) {
	case common.HeaderType_CONFIG:
		configMembership := c.detectConfChange(block)

		c.raftMetadataLock.Lock()
		c.Opts.BlockMetadata.RaftIndex = index
		if configMembership != nil {
			c.Opts.BlockMetadata = configMembership.NewBlockMetadata
			c.Opts.Consenters = configMembership.NewConsenters
		}
		c.raftMetadataLock.Unlock()

		blockMetadataBytes := utils.MarshalOrPanic(c.Opts.BlockMetadata)

		// write block with metadata
		c.support.WriteBlock(block, blockMetadataBytes)

		if configMembership == nil {
			return
		}

		// update membership
		if configMembership.ConfChange != nil {
			// We need to propose conf change in a go routine, because it may be blocked if raft node
			// becomes leaderless, and we should not block `run` so it can keep consuming applyC,
			// otherwise we have a deadlock.
			go func() {
				// ProposeConfChange returns error only if node being stopped.
				// This proposal is dropped by followers because DisableProposalForwarding is enabled.
				if err := c.Node.ProposeConfChange(context.TODO(), *configMembership.ConfChange); err != nil {
					c.logger.Warnf("Failed to propose configuration update to Raft node: %s", err)
				}
			}()

			c.confChangeInProgress = configMembership.ConfChange

			switch configMembership.ConfChange.Type {
			case raftpb.ConfChangeAddNode:
				c.logger.Infof("Config block just committed adds node %d, pause accepting transactions till config change is applied", configMembership.ConfChange.NodeID)
			case raftpb.ConfChangeRemoveNode:
				c.logger.Infof("Config block just committed removes node %d, pause accepting transactions till config change is applied", configMembership.ConfChange.NodeID)
			default:
				c.logger.Panic("Programming error, encountered unsupported raft config change")
			}

			c.configInflight = true
		} else if configMembership.Rotated() {
			lead := atomic.LoadUint64(&c.lastKnownLeader)
			if configMembership.RotatedNode == lead {
				c.logger.Infof("Certificate of Raft leader is being rotated, attempt leader transfer before reconfiguring communication")
				go func() {
					c.Node.abdicateLeader(lead)
					if err := c.configureComm(); err != nil {
						c.logger.Panicf("Failed to configure communication: %s", err)
					}
				}()
			} else {
				if err := c.configureComm(); err != nil {
					c.logger.Panicf("Failed to configure communication: %s", err)
				}
			}
		}

	default:
		c.logger.Panicf("Programming error: unexpected config type: %s", common.HeaderType(hdr.Type))
	}
}

// getInFlightConfChange returns ConfChange in-flight if any.
// It returns confChangeInProgress if it is not nil. Otherwise
// it returns ConfChange from the last committed block (might be nil).
func (c *Consenter) getInFlightConfChange() *raftpb.ConfChange {
	if c.confChangeInProgress != nil {
		return c.confChangeInProgress
	}

	if c.lastBlock.Header.Number == 0 {
		return nil // nothing to failover just started the chain
	}

	if !utils.IsConfigBlock(c.lastBlock) {
		return nil
	}

	// extracting current Raft configuration state
	confState := c.Node.ApplyConfChange(raftpb.ConfChange{})

	if len(confState.Voters) == len(c.Opts.BlockMetadata.ConsenterEndpoints) {
		// Raft configuration change could only add one node or
		// remove one node at a time, if raft conf state size is
		// equal to membership stored in block metadata field,
		// that means everything is in sync and no need to propose
		// config update.
		return nil
	}

	return ConfChange(c.Opts.BlockMetadata, confState)
}

// newMetadata extract config metadata from the configuration block
func (c *Consenter) newConfigMetadata(block *common.Block) *etcdraft.ConfigMetadata {
	envelope, err := utils.ExtractEnvelope(block, 0)
	if err != nil {
		log.Logger.Error("failed to extract envelope from the block")
		return nil
	}
	payload, err := utils.UnmarshalPayload(envelope.Payload)
	if err != nil {
		log.Logger.Error("failed to extract payload from config envelope")
		return nil
	}
	nodes := &etcdraft.ConfigMetadata{}
	err = proto.Unmarshal(payload.Data, nodes)
	if err != nil {
		log.Logger.Error("cannot unmarshal payload.Data into ConfigMetadata")
		return nil
	}
	return nodes
}

func (c *Consenter) suspectEviction() bool {
	if c.isRunning() != nil {
		return false
	}

	return atomic.LoadUint64(&c.lastKnownLeader) == uint64(0)
}

func (c *Consenter) triggerCatchup(sn *raftpb.Snapshot) {
	select {
	case c.snapC <- sn:
	case <-c.doneC:
	}
}
