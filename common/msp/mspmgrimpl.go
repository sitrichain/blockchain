/*
Copyright IBM Corp. 2016 All Rights Reserved.

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

package msp

import (
	"fmt"

	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/protos/msp"
)

type mspManagerImpl struct {
	// map that contains all MSPs that we have setup or otherwise added
	mspsMap map[string]MSP

	// error that might have occurred at startup
	up bool
}

// NewMSPManager returns a new MSP manager instance;
// note that this instance is not initialized until
// the Setup method is called
func NewMSPManager() MSPManager {
	return &mspManagerImpl{}
}

// Setup initializes the internal data structures of this manager and creates MSPs
func (mgr *mspManagerImpl) Setup(msps []MSP) error {
	if mgr.up {
		log.Logger.Infof("MSP manager already up")
		return nil
	}

	if msps == nil {
		return fmt.Errorf("Setup error: nil config object")
	}

	if len(msps) == 0 {
		return fmt.Errorf("Setup error: at least one MSP configuration item is required")
	}

	log.Logger.Debugf("Setting up the MSP manager (%d msps)", len(msps))

	// create the map that assigns MSP IDs to their manager instance - once
	mgr.mspsMap = make(map[string]MSP)

	for _, m := range msps {
		// add the MSP to the map of active MSPs
		mspID, err := m.GetIdentifier()
		if err != nil {
			return fmt.Errorf("Could not extract msp identifier, err %s", err)
		}
		mgr.mspsMap[mspID] = m
	}

	mgr.up = true

	log.Logger.Infof("MSP manager setup complete, setup %d msps", len(msps))

	return nil
}

// GetMSPs returns the MSPs that are managed by this manager
func (mgr *mspManagerImpl) GetMSPs() (map[string]MSP, error) {
	return mgr.mspsMap, nil
}

// DeserializeIdentity returns an identity given its serialized version supplied as argument
func (mgr *mspManagerImpl) DeserializeIdentity(serializedID []byte) (Identity, error) {
	// We first deserialize to a SerializedIdentity to get the MSP ID
	sId := &msp.SerializedIdentity{}
	if err := sId.Unmarshal(serializedID); err != nil {
		return nil, fmt.Errorf("Could not deserialize a SerializedIdentity, err %s", err)
	}

	// we can now attempt to obtain the MSP
	m := mgr.mspsMap[sId.Mspid]
	if m == nil {
		log.Logger.Errorf("MSP %s is unknown. Current: %v", sId.Mspid, m)
		return nil, fmt.Errorf("MSP %s is unknown", sId.Mspid)
	}

	switch t := m.(type) {
	case *bccspmsp:
		return t.deserializeIdentityInternal(sId.IdBytes)
	default:
		return t.DeserializeIdentity(serializedID)
	}
}
