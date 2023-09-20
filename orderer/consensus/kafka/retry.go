package kafka

import (
	"fmt"
	"time"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/orderer/localconfig"
)

type retryProcess struct {
	shortPollingInterval, shortTimeout time.Duration
	longPollingInterval, longTimeout   time.Duration
	exit                               chan struct{}
	topic                              string
	msg                                string
	fn                                 func() error
}

func newRetryProcess(retryOptions localconfig.Retry, exit chan struct{}, topic string, msg string, fn func() error) *retryProcess {
	return &retryProcess{
		shortPollingInterval: retryOptions.ShortInterval,
		shortTimeout:         retryOptions.ShortTotal,
		longPollingInterval:  retryOptions.LongInterval,
		longTimeout:          retryOptions.LongTotal,
		exit:                 exit,
		topic:                topic,
		msg:                  msg,
		fn:                   fn,
	}
}

func (rp *retryProcess) retry() error {
	if err := rp.try(rp.shortPollingInterval, rp.shortTimeout); err != nil {
		log.Logger.Debugf("[chain: %s] Switching to the long retry interval", rp.topic)
		return rp.try(rp.longPollingInterval, rp.longTimeout)
	}
	return nil
}

func (rp *retryProcess) try(interval, total time.Duration) error {
	// Configuration validation will not allow non-positive ticker values
	// (which would result in panic). The path below is for those test cases
	// when we cannot avoid the creation of a retriable process but we wish
	// to terminate it right away.
	if rp.shortPollingInterval == 0*time.Second {
		return fmt.Errorf("illegal value")
	}

	tickInterval := time.NewTicker(time.Second * 2)
	tickTotal := time.NewTicker(total)
	defer tickTotal.Stop()
	defer tickInterval.Stop()
	log.Logger.Debugf("[chain: %s] Retrying every %s for a total of %s", rp.topic, interval.String(), total.String())

	for {
		select {
		case <-rp.exit:
			exitErr := fmt.Errorf("[chain: %s] process asked to exit", rp.topic)
			log.Logger.Warn(exitErr.Error())
			return exitErr
		case <-tickTotal.C:
			return fmt.Errorf("process has not been executed yet")
		case <-tickInterval.C:
			log.Logger.Debugf("[chain: %s] "+rp.msg, rp.topic)
			if err := rp.fn(); err == nil {
				log.Logger.Debugf("[chain: %s] Error is nil, breaking the retry loop", rp.topic)
				return err
			}
		}
	}
}
