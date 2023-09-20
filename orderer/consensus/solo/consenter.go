package solo

import (
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/consensus"
	"github.com/rongzer/blockchain/orderer/filters"
	cb "github.com/rongzer/blockchain/protos/common"
)

type commiter struct {
	envelope  *cb.Envelope
	committer filters.Committer
}

type consenter struct {
	chain        consensus.ChainResource
	batchTimeout time.Duration
	sendChan     chan *commiter
	exitChan     chan struct{}
}

func newConsenter(chain consensus.ChainResource) *consenter {
	return &consenter{
		batchTimeout: chain.SharedConfig().BatchTimeout(),
		chain:        chain,
		sendChan:     make(chan *commiter),
		exitChan:     make(chan struct{}),
	}
}

func (c *consenter) Start() {
	go c.main()
}

func (c *consenter) Halt() {
	select {
	case <-c.exitChan:
		// Allow multiple halts without panic
	default:
		close(c.exitChan)
	}
}

// Order 接收消息进行排序
func (c *consenter) Order(env *cb.Envelope, committer filters.Committer) bool {
	commiter := &commiter{env, committer}
	select {
	case c.sendChan <- commiter:
		return true
	case <-c.exitChan:
		return false
	}
}

// Errored only closes on exit
func (c *consenter) Errored() <-chan struct{} {
	return c.exitChan
}

func (c *consenter) main() {
	var timer <-chan time.Time
	for {
		select {
		case msg := <-c.sendChan:
			batches, committers, ok := c.chain.BlockCutter().Ordered(msg.envelope, msg.committer)
			if ok && len(batches) == 0 && timer == nil {
				timer = time.After(c.batchTimeout)
				continue
			}
			for i, batch := range batches {
				block, err := c.chain.CreateNextBlock(batch)
				if err != nil {
					log.Logger.Errorf("Create next block error: %s", err)
					continue
				}
				c.chain.WriteBlock(block, committers[i], nil)
			}

			pending := c.chain.BlockCutter().GetPendingBatchSize() < 1
			switch {
			case timer != nil && !pending:
				// Timer is already running but there are no messages pending, stop the timer
				timer = nil
			case timer == nil && pending:
				// Timer is not already running and there are messages pending, so start it
				timer = time.After(c.batchTimeout)
			default:
				// Do nothing when:
				// 1. Timer is already running and there are messages pending
				// 2. Timer is not set and there are no messages pending
			}
		case <-timer:
			//clear the timer
			timer = nil

			batch, committers := c.chain.BlockCutter().Cut()
			if len(batch) == 0 {
				log.Logger.Warnf("Batch timer expired with no pending requests, this might indicate a bug")
				continue
			}
			block, err := c.chain.CreateNextBlock(batch)
			if err != nil {
				log.Logger.Errorf("Create next block error: %s", err)
				continue
			}
			c.chain.WriteBlock(block, committers, nil)
		case <-c.exitChan:
			log.Logger.Debugf("Exiting")
			return
		}
	}
}
