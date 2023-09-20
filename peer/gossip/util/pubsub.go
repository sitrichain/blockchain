package util

import (
	"errors"
	"sync"
	"time"
)

const (
	subscriptionBuffSize = 50
)

// PubSub defines a struct that one can use to:
// - publish items to a topic to multiple subscribers
// - and subscribe to items from a topic
// The subscriptions have a TTL and are cleaned when it passes.
type PubSub struct {
	// a map from topic to Set of subscriptions
	subscriptions sync.Map // map[string]*Set
}

// Subscription defines a subscription to a topic
// that can be used to receive publishes on
type Subscription interface {
	// Listen blocks until a publish was made
	// to the subscription, or an error if the
	// subscription's TTL passed
	Listen() (interface{}, error)
}

type subscription struct {
	top string
	ttl time.Duration
	c   chan interface{}
}

// Listen blocks until a publish was made
// to the subscription, or an error if the
// subscription's TTL passed
func (s *subscription) Listen() (interface{}, error) {
	select {
	case <-time.After(s.ttl):
		return nil, errors.New("timed out")
	case item := <-s.c:
		return item, nil
	}
}

// NewPubSub creates a new PubSub with an empty
// set of subscriptions
func NewPubSub() *PubSub {
	return &PubSub{}
}

// Publish publishes an item to all subscribers on the topic
func (ps *PubSub) Publish(topic string, item interface{}) error {
	s, ok := ps.subscriptions.Load(topic)
	if !ok {
		return errors.New("no subscribers")
	}
	for _, sub := range s.(*Set).ToArray() {
		c := sub.(*subscription).c
		// Not enough room in buffer, continue in order to not block publisher
		if len(c) == subscriptionBuffSize {
			continue
		}
		c <- item
	}
	return nil
}

// Subscribe returns a subscription to a topic that expires when given TTL passes
func (ps *PubSub) Subscribe(topic string, ttl time.Duration) Subscription {
	sub := &subscription{
		top: topic,
		ttl: ttl,
		c:   make(chan interface{}, subscriptionBuffSize),
	}

	// Add subscription to subscriptions map
	s, _ := ps.subscriptions.LoadOrStore(topic, NewSet())

	// Add the subscription
	s.(*Set).Add(sub)

	// When the timeout expires, remove the subscription
	time.AfterFunc(ttl, func() {
		ps.unSubscribe(sub)
	})
	return sub
}

func (ps *PubSub) unSubscribe(sub *subscription) {
	s, ok := ps.subscriptions.Load(sub.top)
	if !ok {
		return
	}

	s.(*Set).Remove(sub)
	if s.(*Set).Size() != 0 {
		return
	}
	// Else, this is the last subscription for the topic.
	// Remove the set from the subscriptions map
	ps.subscriptions.Delete(sub.top)
}
