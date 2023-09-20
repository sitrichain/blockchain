package kafka

import (
	"fmt"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/consensus"
	"github.com/rongzer/blockchain/orderer/filters"
	"github.com/rongzer/blockchain/orderer/localconfig"
	cb "github.com/rongzer/blockchain/protos/common"
	ab "github.com/rongzer/blockchain/protos/orderer"
	"github.com/rongzer/blockchain/protos/utils"
	"github.com/spf13/viper"
)

// Used for capturing metrics -- see processMessagesToBlocks
const (
	indexRecvError = iota
	indexUnmarshalError
	indexRecvPass
	indexProcessConnectPass
	indexProcessTimeToCutError
	indexProcessTimeToCutPass
	indexProcessRegularError
	indexProcessRegularPass
	indexSendTimeToCutError
	indexSendTimeToCutPass
	indexExitChanPass
)

// consenterConfig allows us to retrieve the configuration options set on the
// consenterConfig object. These will be common across all chain objects derived by
// this consenterConfig. They are set using using local configuration settings. This
// interface is satisfied by mode.
type consenterConfig interface {
	brokerConfig() *sarama.Config
	retryOptions() localconfig.Retry
}

type consenter struct {
	consenterConfig consenterConfig
	chain           consensus.ChainResource

	topic               string
	lastOffsetPersisted int64
	lastCutBlockNumber  uint64

	producer        sarama.SyncProducer
	parentConsumer  sarama.Consumer
	channelConsumer sarama.PartitionConsumer

	// When the partition consumer errors, close the chain. Otherwise, make
	// this an open, unbuffered chain.
	errorChan chan struct{}
	// When a Halt() request comes, close the chain. Unlike errorChan, this
	// chain never re-opens when closed. Its closing triggers the exit of the
	// processMessagesToBlock loop.
	haltChan chan struct{}
	// Close when the retriable steps in Start have completed.
	startChan chan struct{}

	rOffset int64
	pOffset int64
}

func newConsenter(config consenterConfig, chain consensus.ChainResource, lastOffsetPersisted int64) (*consenter, error) {
	chainId := chain.ChainID()
	lastCutBlockNumber := chain.Height() - 1

	log.Logger.Infof("[chain: %s] Starting chain consenter with last persisted offset %d and last recorded block %d",
		chainId, lastOffsetPersisted, lastCutBlockNumber)

	kafkaInit := strings.TrimSpace(viper.GetString("kafka.init"))
	if kafkaInit == "true" {
		log.Logger.Infof("[chain: %s] kafka init to 0")
		lastOffsetPersisted = 0
	}
	errorChan := make(chan struct{})
	close(errorChan) // We need this closed when starting up

	return &consenter{
		consenterConfig:     config,
		chain:               chain,
		topic:               chainId,
		lastOffsetPersisted: lastOffsetPersisted,
		lastCutBlockNumber:  lastCutBlockNumber,
		errorChan:           errorChan,
		haltChan:            make(chan struct{}),
		startChan:           make(chan struct{}),
		rOffset:             0,
		pOffset:             0,
	}, nil
}

// Errored returns a chain which will close when a partition consumer error
// has occurred. Checked by Deliver().
func (c *consenter) Errored() <-chan struct{} {
	select {
	case <-c.startChan:
		return c.errorChan
	default:
		dummyError := make(chan struct{})
		close(dummyError)
		return dummyError
	}
}

// Start allocates the necessary resources for staying up to date with this
// Consenter. Implements the multichain.Consenter interface. Called by
// multichain.NewManagerImpl() which is invoked when the ordering process is
// launched, before the call to NewServer(). Launches a goroutine so as not to
// block the multichain.Manager.
func (c *consenter) Start() {
	go c.main()
}

// Halt frees the resources which were allocated for this Consenter. Implements the
// multichain.Consenter interface.
func (c *consenter) Halt() {
	select {
	case <-c.haltChan:
		// This construct is useful because it allows Halt() to be called
		// multiple times (by a single thread) w/o panicking. Recal that a
		// receive from a closed chain returns (the zero value) immediately.
		log.Logger.Warnf("[chain: %s] Halting of chain requested again", c.topic)
	default:
		log.Logger.Errorf("[chain: %s] Halting of chain requested", c.topic)
		close(c.haltChan)
		c.closeKafkaObjects() // Also close the producer and the consumer
		log.Logger.Debugf("[chain: %s] Closed the haltChan", c.topic)
	}
}

// Order 接收消息进行排序
func (c *consenter) Order(env *cb.Envelope, _ filters.Committer) bool {
	select {
	case <-c.startChan: // The Start phase has completed
		select {
		case <-c.haltChan: // The chain has been halted, stop here
			log.Logger.Warnf("[chain: %s] Will not enqueue, consenterConfig for this chain has been halted", c.topic)
			return false
		default: // The post path
			marshaledEnv, err := utils.Marshal(env)
			if err != nil {
				log.Logger.Errorf("[chain: %s] cannot enqueue, unable to marshal envelope = %s", c.topic, err)
				return false
			}
			payload := utils.MarshalOrPanic(newRegularMessage(marshaledEnv))
			message := newProducerMessage(c.topic, payload)
			_, pOffset, err := c.producer.SendMessage(message)
			if err != nil {
				log.Logger.Errorf("[chain: %s] cannot enqueue envelope = %s", c.topic, err)
				return false
			}
			c.pOffset = pOffset
			return true
		}
	default: // Not ready yet
		log.Logger.Warnf("[chain: %s] Will not enqueue, consenterConfig for this chain hasn't started yet", c.topic)
		return false
	}
}

// Called by Start().
func (c *consenter) main() {
	var err error

	brokers := c.chain.SharedConfig().KafkaBrokers()
	// Set up the producer
	c.producer, err = c.setupProducerForChannel(brokers)
	if err != nil {
		log.Logger.Panicf("[chain: %s] Cannot set up producer = %s", c.topic, err)
	}
	log.Logger.Infof("[chain: %s] Producer set up successfully", c.topic)

	// Have the producer post the CONNECT message
	if err = c.sendConnectMessage(); err != nil {
		log.Logger.Panicf("[chain: %s] Cannot post CONNECT message = %s", c.topic, err)
	}
	log.Logger.Infof("[chain: %s] CONNECT message posted successfully", c.topic)

	// Set up the parent consumer
	c.parentConsumer, err = c.setupParentConsumerForChannel(brokers)
	if err != nil {
		log.Logger.Panicf("[chain: %s] Cannot set up parent consumer = %s", c.topic, err)
	}
	log.Logger.Infof("[chain: %s] Parent consumer set up successfully", c.topic)

	// Set up the chain consumer
	c.channelConsumer, err = c.setupChannelConsumerForChannel()
	if err != nil {
		log.Logger.Panicf("[chain: %s] Cannot set up chain consumer = %s", c.topic, err)
	}
	log.Logger.Infof("[chain: %s] Channel consumer set up successfully", c.topic)

	close(c.startChan)                // Broadcast requests will now go through
	c.errorChan = make(chan struct{}) // Deliver requests will also go through

	log.Logger.Infof("[chain: %s] Start phase completed successfully", c.topic)

	c.processMessagesToBlocks() // Keep up to date with the chain
}

// processMessagesToBlocks drains the Kafka consumer for the given chain, and
// takes care of converting the stream of ordered messages into blocks for the
// chain's ledger.
func (c *consenter) processMessagesToBlocks() ([]uint64, error) {
	counts := make([]uint64, 11) // For metrics and tests
	msg := new(ab.KafkaMessage)
	var timer <-chan time.Time

	defer func() { // When Halt() is called
		select {
		case <-c.errorChan: // If already closed, don't do anything
		default:
			close(c.errorChan)
		}
	}()

	for {
		select {
		case <-c.haltChan:
			log.Logger.Warnf("[chain: %s] consenter for chain exiting", c.topic)
			counts[indexExitChanPass]++
			panic("consenter for chain exiting")
			return counts, nil
		case kafkaErr := <-c.channelConsumer.Errors():
			log.Logger.Errorf("[chain: %s] Error during consumption: %s", c.topic, kafkaErr)
			counts[indexRecvError]++
			select {
			case <-c.errorChan: // If already closed, don't do anything
			default:
				close(c.errorChan)
			}
			log.Logger.Warnf("[chain: %s] Closed the errorChan", c.topic)
			// This covers the edge case where (1) a consumption error has
			// closed the errorChan and thus rendered the chain unavailable to
			// deliver clients, (2) we're already at the newest offset, and (3)
			// there are no new Broadcast requests coming in. In this case,
			// there is no trigger that can recreate the errorChan again and
			// mark the chain as available, so we have to force that trigger via
			// the emission of a CONNECT message. TODO Consider rate limiting
			go c.sendConnectMessage()
		case in, ok := <-c.channelConsumer.Messages():
			if !ok {
				log.Logger.Errorf("[chain: %s] Kafka consumer closed.", c.topic)
				panic("Kafka consumer closed")
				return counts, nil
			}
			select {
			case <-c.errorChan: // If this chain was closed...
				c.errorChan = make(chan struct{}) // ...make a new one.
				log.Logger.Infof("[chain: %s] Marked consenterConfig as available again", c.topic)
			default:
			}

			c.rOffset = in.Offset
			if err := msg.Unmarshal(in.Value); err != nil {
				// This shouldn't happen, it should be filtered at ingress
				log.Logger.Errorf("[chain: %s] Unable to unmarshal consumed message = %s", c.topic, err)
				counts[indexUnmarshalError]++
				continue
			} else {
				log.Logger.Debugf("[chain: %s] Successfully unmarshalled consumed message, offset is %d. Inspecting type...", c.topic, in.Offset)
				counts[indexRecvPass]++
			}

			switch msg.Type.(type) {
			case *ab.KafkaMessage_Connect:
				log.Logger.Debugf("[chain: %s] It's a connect message - ignoring", c.topic)
				counts[indexProcessConnectPass]++
			case *ab.KafkaMessage_TimeToCut:
				if err := c.processTimeToCut(msg.GetTimeToCut(), &timer, in.Offset); err != nil {
					log.Logger.Errorf("[chain: %s] Error when processing TimeToCut message. %s", c.topic, err)
					counts[indexProcessTimeToCutError]++
				} else {
					counts[indexProcessTimeToCutPass]++
				}
			case *ab.KafkaMessage_Regular:
				if err := c.processRegular(msg.GetRegular(), &timer, in.Offset); err != nil {
					log.Logger.Errorf("[chain: %s] Error when processing Regular message. %s", c.topic, err)
					counts[indexProcessRegularError]++
				} else {
					counts[indexProcessRegularPass]++
				}
			}
		case <-timer:
			if err := c.sendTimeToCut(&timer); err != nil {
				log.Logger.Errorf("[chain: %s] cannot post time-to-cut message. %s", c.topic, err)
				// Do not return though
				counts[indexSendTimeToCutError]++
			} else {
				counts[indexSendTimeToCutPass]++
			}
		}
	}
}

func (c *consenter) closeKafkaObjects() []error {
	var errs []error

	err := c.channelConsumer.Close()
	if err != nil {
		log.Logger.Errorf("[chain: %s] could not close channelConsumer cleanly = %s", c.topic, err)
		errs = append(errs, err)
	} else {
		log.Logger.Debugf("[chain: %s] Closed the chain consumer", c.topic)
	}

	err = c.parentConsumer.Close()
	if err != nil {
		log.Logger.Errorf("[chain: %s] could not close parentConsumer cleanly = %s", c.topic, err)
		errs = append(errs, err)
	} else {
		log.Logger.Debugf("[chain: %s] Closed the parent consumer", c.topic)
	}

	err = c.producer.Close()
	if err != nil {
		log.Logger.Errorf("[chain: %s] could not close producer cleanly = %s", c.topic, err)
		errs = append(errs, err)
	} else {
		log.Logger.Debugf("[chain: %s] Closed the producer", c.topic)
	}

	return errs
}

func (c *consenter) processRegular(regularMessage *ab.KafkaMessageRegular, timer *<-chan time.Time, receivedOffset int64) error {
	env := &cb.Envelope{}
	if err := env.Unmarshal(regularMessage.Payload); err != nil {
		// This shouldn't happen, it should be filtered at ingress
		return fmt.Errorf("unmarshal/%s", err)
	}

	env.Offset = receivedOffset
	batches, committers, ok := c.chain.BlockCutter().Ordered(env, nil)
	log.Logger.Debugf("[chain: %s] Ordering results: items in batch = %d, ok = %v", c.topic, len(batches), ok)
	if ok && len(batches) == 0 && *timer == nil {
		*timer = time.After(c.chain.SharedConfig().BatchTimeout())
		log.Logger.Debugf("[chain: %s] Just began %s batch timer", c.topic, c.chain.SharedConfig().BatchTimeout().String())
		return nil
	}

	// If !ok, batches == nil, so this will be skipped
	for i, batch := range batches {
		// If more than one batch is produced, exactly 2 batches are produced.
		// The receivedOffset for the first batch is one less than the supplied
		// offset to this function.
		c.lastCutBlockNumber++
		offset := batch[len(batch)-1].Offset
		block, err := c.chain.CreateNextBlock(batch)
		if err != nil {
			log.Logger.Errorf("Create next block error: %s", err)
			return err
		}
		encodedLastOffsetPersisted := utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: offset})
		c.chain.WriteBlock(block, committers[i], encodedLastOffsetPersisted)
		log.Logger.Infof("[chain: %s] Batch cut block[%d] txnum[%d] pending[%d], offset %d pOffset %d",
			c.topic, c.lastCutBlockNumber, len(block.Data.Data), c.chain.BlockCutter().GetPendingBatchSize(), offset, c.pOffset)
	}
	if len(batches) > 0 {
		*timer = nil
	}
	return nil
}

func (c *consenter) processTimeToCut(ttcMessage *ab.KafkaMessageTimeToCut, timer *<-chan time.Time, receivedOffset int64) error {
	ttcNumber := ttcMessage.GetBlockNumber()
	log.Logger.Debugf("[chain: %s] It's a time-to-cut message for block %d", c.topic, ttcNumber)
	if ttcNumber == c.lastCutBlockNumber+1 {
		*timer = nil
		log.Logger.Debugf("[chain: %s] Nil'd the timer", c.topic)
		batch, committers := c.chain.BlockCutter().Cut()
		if len(batch) == 0 {
			return fmt.Errorf("got right time-to-cut message (for block %d),"+
				" no pending requests though; this might indicate a bug", c.lastCutBlockNumber+1)
		}
		c.lastCutBlockNumber++

		block, err := c.chain.CreateNextBlock(batch)
		if err != nil {
			return fmt.Errorf("Create next block error: %s", err)
		}
		encodedLastOffsetPersisted := utils.MarshalOrPanic(&ab.KafkaMetadata{LastOffsetPersisted: receivedOffset})

		c.chain.WriteBlock(block, committers, encodedLastOffsetPersisted)
		log.Logger.Infof("[chain: %s] Timer cut block[%d] txnum[%d] pending[%d] , offset %d pOffset %d",
			c.topic, c.lastCutBlockNumber, len(block.Data.Data), c.chain.BlockCutter().GetPendingBatchSize(), receivedOffset, c.pOffset)
		return nil
	} else if ttcNumber > c.lastCutBlockNumber+1 {
		return fmt.Errorf("got larger time-to-cut message (%d) than allowed/expected (%d)"+
			" - this might indicate a bug", ttcNumber, c.lastCutBlockNumber+1)
	}
	log.Logger.Debugf("[chain: %s] Ignoring stale time-to-cut-message for block %d", c.topic, ttcNumber)
	return nil
}

// Post a CONNECT message to the chain using the given retry options. This
// prevents the panicking that would occur if we were to set up a consumer and
// seek on a partition that hadn't been written to yet.
func (c *consenter) sendConnectMessage() error {
	log.Logger.Infof("[chain: %s] About to post the CONNECT kafka message...", c.topic)

	payload := utils.MarshalOrPanic(newConnectMessage())
	message := newProducerMessage(c.topic, payload)

	retryMsg := "Attempting to post the CONNECT kafka message..."
	postConnect := newRetryProcess(c.consenterConfig.retryOptions(), c.haltChan, c.topic, retryMsg, func() error {
		_, _, err := c.producer.SendMessage(message)
		return err
	})

	return postConnect.retry()
}

func (c *consenter) sendTimeToCut(timer *<-chan time.Time) error {
	timeToCutBlockNumber := c.lastCutBlockNumber + 1
	log.Logger.Debugf("[chain: %s] Time-to-cut block %d timer expired", c.topic, timeToCutBlockNumber)
	*timer = nil
	payload := utils.MarshalOrPanic(newTimeToCutMessage(timeToCutBlockNumber))
	message := newProducerMessage(c.topic, payload)
	_, pOffset, err := c.producer.SendMessage(message)
	c.pOffset = pOffset
	return err
}

// Sets up the partition consumer for a chain using the given retry options.
func (c *consenter) setupChannelConsumerForChannel() (sarama.PartitionConsumer, error) {
	var err error
	var channelConsumer sarama.PartitionConsumer
	startFrom := c.lastOffsetPersisted + 1
	log.Logger.Infof("[chain: %s] Setting up the chain consumer of kafka(start offset: %d)...", c.topic, startFrom)

	retryMsg := "Connecting to the Kafka cluster"
	setupChannelConsumer := newRetryProcess(c.consenterConfig.retryOptions(), c.haltChan, c.topic, retryMsg, func() error {
		channelConsumer, err = c.parentConsumer.ConsumePartition(c.topic, 0, startFrom)
		return err
	})

	return channelConsumer, setupChannelConsumer.retry()
}

// Sets up the parent consumer for a chain using the given retry options.
func (c *consenter) setupParentConsumerForChannel(brokers []string) (sarama.Consumer, error) {
	var err error
	var parentConsumer sarama.Consumer

	log.Logger.Infof("[chain: %s] Setting up the parent consumer of kafka...", c.topic)

	retryMsg := "Connecting to the Kafka cluster"
	setupParentConsumer := newRetryProcess(c.consenterConfig.retryOptions(), c.haltChan, c.topic, retryMsg, func() error {
		parentConsumer, err = sarama.NewConsumer(brokers, c.consenterConfig.brokerConfig())
		return err
	})

	return parentConsumer, setupParentConsumer.retry()
}

// Sets up the writer/producer for a chain using the given retry options.
func (c *consenter) setupProducerForChannel(brokers []string) (sarama.SyncProducer, error) {
	var err error
	var producer sarama.SyncProducer

	log.Logger.Infof("[chain: %s] Setting up the producer of kafka...", c.topic)

	retryMsg := "Connecting to the Kafka cluster"
	setupProducer := newRetryProcess(c.consenterConfig.retryOptions(), c.haltChan, c.topic, retryMsg, func() error {
		producer, err = sarama.NewSyncProducer(brokers, c.consenterConfig.brokerConfig())
		return err
	})

	return producer, setupProducer.retry()
}
