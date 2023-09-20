/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package comm

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	"github.com/rongzer/blockchain/common/log"
	"google.golang.org/grpc/credentials"
)

var caSupport *CASupport
var once sync.Once

// CASupport type manages certificate authorities scoped by channel
type CASupport struct {
	sync.RWMutex
	AppRootCAsByChain     map[string][][]byte
	OrdererRootCAsByChain map[string][][]byte
	ClientRootCAs         [][]byte
	ServerRootCAs         [][]byte
}

// GetCASupport returns the signleton CASupport instance
func GetCASupport() *CASupport {

	once.Do(func() {
		caSupport = &CASupport{
			AppRootCAsByChain:     make(map[string][][]byte),
			OrdererRootCAsByChain: make(map[string][][]byte),
		}
	})
	return caSupport
}

// GetServerRootCAs returns the PEM-encoded root certificates for all of the
// application and orderer organizations defined for all chains.  The root
// certificates returned should be used to set the trusted server roots for
// TLS clients.
func (cas *CASupport) GetServerRootCAs() (appRootCAs, ordererRootCAs [][]byte) {
	cas.RLock()
	defer cas.RUnlock()

	appRootCAs = [][]byte{}
	ordererRootCAs = [][]byte{}

	for _, appRootCA := range cas.AppRootCAsByChain {
		appRootCAs = append(appRootCAs, appRootCA...)
	}

	for _, ordererRootCA := range cas.OrdererRootCAsByChain {
		ordererRootCAs = append(ordererRootCAs, ordererRootCA...)
	}

	// also need to append statically configured root certs
	appRootCAs = append(appRootCAs, cas.ServerRootCAs...)
	return appRootCAs, ordererRootCAs
}

// GetDeliverServiceCredentials returns GRPC transport credentials for given channel to be used by GRPC
// clients which communicate with ordering service endpoint.
// If the channel isn't found, error is returned.
func (cas *CASupport) GetDeliverServiceCredentials(channelID string) (credentials.TransportCredentials, error) {
	cas.RLock()
	defer cas.RUnlock()

	var creds credentials.TransportCredentials
	var tlsConfig = &tls.Config{}
	var certPool = x509.NewCertPool()

	rootCACerts, exists := cas.OrdererRootCAsByChain[channelID]
	if !exists {
		log.Logger.Errorf("Attempted to obtain root CA certs of a non existent channel: %s", channelID)
		return nil, fmt.Errorf("didn't find any root CA certs for channel %s", channelID)
	}

	for _, cert := range rootCACerts {
		block, _ := pem.Decode(cert)
		if block != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				certPool.AddCert(cert)
			} else {
				log.Logger.Warnf("Failed to add root cert to credentials (%s)", err)
			}
		} else {
			log.Logger.Warn("Failed to add root cert to credentials")
		}
	}
	tlsConfig.RootCAs = certPool
	creds = credentials.NewTLS(tlsConfig)
	return creds, nil
}

// GetPeerCredentials returns GRPC transport credentials for use by GRPC
// clients which communicate with remote peer endpoint.
func (cas *CASupport) GetPeerCredentials(tlsCert tls.Certificate) credentials.TransportCredentials {
	var creds credentials.TransportCredentials
	var tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}
	var certPool = x509.NewCertPool()
	// loop through the orderer CAs
	roots, _ := cas.GetServerRootCAs()
	for _, root := range roots {
		block, _ := pem.Decode(root)
		if block != nil {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				certPool.AddCert(cert)
			} else {
				log.Logger.Warnf("Failed to add root cert to credentials (%s)", err)
			}
		} else {
			log.Logger.Warn("Failed to add root cert to credentials")
		}
	}
	tlsConfig.RootCAs = certPool
	creds = credentials.NewTLS(tlsConfig)
	return creds
}

// GetClientRootCAs returns the PEM-encoded root certificates for all of the
// application and orderer organizations defined for all chains.  The root
// certificates returned should be used to set the trusted client roots for
// TLS servers.
func (cas *CASupport) GetClientRootCAs() (appRootCAs, ordererRootCAs [][]byte) {
	cas.RLock()
	defer cas.RUnlock()

	appRootCAs = [][]byte{}
	ordererRootCAs = [][]byte{}

	for _, appRootCA := range cas.AppRootCAsByChain {
		appRootCAs = append(appRootCAs, appRootCA...)
	}

	for _, ordererRootCA := range cas.OrdererRootCAsByChain {
		ordererRootCAs = append(ordererRootCAs, ordererRootCA...)
	}

	// also need to append statically configured root certs
	appRootCAs = append(appRootCAs, cas.ClientRootCAs...)
	return appRootCAs, ordererRootCAs
}
