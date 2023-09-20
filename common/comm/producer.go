/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package comm

import (
	"fmt"
	"github.com/rongzer/blockchain/common/log"
	"google.golang.org/grpc"
	"sync"
)

// ConnectionFactory creates a connection to a certain endpoint
type ConnectionFactory func(endpoint string) (*grpc.ClientConn, error)

// ConnectionProducer produces connections out of a set of predefined
// endpoint
type ConnectionProducer interface {
	// NewConnection creates a new connection.
	// Returns the connection, the endpoint selected, nil on success.
	// Returns nil, "", error on failure
	NewConnection() (*grpc.ClientConn, string, error)
}

type connProducer struct {
	sync.RWMutex
	endpoint string
	connect  ConnectionFactory
}

// NewConnectionProducer creates a new ConnectionProducer with given endpoint and connection factory.
// It returns nil, if the given endpoint slice is empty.
func NewConnectionProducer(factory ConnectionFactory, endpoint string) ConnectionProducer {
	if len(endpoint) == 0 {
		return nil
	}
	return &connProducer{endpoint: endpoint, connect: factory}
}

// NewConnection creates a new connection.
// Returns the connection, the endpoint selected, nil on success.
// Returns nil, "", error on failure
func (cp *connProducer) NewConnection() (*grpc.ClientConn, string, error) {
	cp.RLock()
	defer cp.RUnlock()

	conn, err := cp.connect(cp.endpoint)
	if err != nil {
		log.Logger.Errorf("Failed connecting to %v, err is: %v", cp.endpoint, err)
		return nil, "", fmt.Errorf("Could not connect to the endpoint: %v", cp.endpoint)
	}
	return conn, cp.endpoint, nil
}

