package gossip

import (
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/common/msp/mgmt"
	"github.com/rongzer/blockchain/peer/gossip/api"
)

// mspSecurityAdvisor implements the SecurityAdvisor interface
// using peer's MSPs.
//
// In order for the system to be secure it is vital to have the
// MSPs to be up-to-date. Channels' MSPs are updated via
// configuration transactions distributed by the ordering service.
//
// This implementation assumes that these mechanisms are all in place and working.
type mspSecurityAdvisor struct {
	deserializer mgmt.DeserializersManager
}

// NewSecurityAdvisor creates a new instance of mspSecurityAdvisor
// that implements MessageCryptoService
func NewSecurityAdvisor(deserializer mgmt.DeserializersManager) api.SecurityAdvisor {
	return &mspSecurityAdvisor{deserializer: deserializer}
}

// OrgByPeerIdentity returns the OrgIdentityType
// of a given peer identity.
// If any error occurs, nil is returned.
// This method does not validate peerIdentity.
// This validation is supposed to be done appropriately during the execution flow.
func (advisor *mspSecurityAdvisor) OrgByPeerIdentity(peerIdentity api.PeerIdentityType) api.OrgIdentityType {
	// Validate arguments
	if len(peerIdentity) == 0 {
		log.Logger.Error("Invalid Peer Identity. It must be different from nil.")

		return nil
	}

	// Notice that peerIdentity is assumed to be the serialization of an identity.
	// So, first step is the identity deserialization

	// TODO: This method should return a structure consisting of two fields:
	// one of the MSPidentifier of the MSP the identity belongs to,
	// and then a list of organization units this identity is in possession of.
	// For gossip use, it is the first part that we would need for now,
	// namely the identity's MSP identifier be returned (Identity.GetMSPIdentifier())

	// First check against the local MSP.
	identity, err := advisor.deserializer.GetLocalDeserializer().DeserializeIdentity(peerIdentity)
	if err == nil {
		return []byte(identity.GetMSPIdentifier())
	}

	// Check against managers
	for chainID, mspManager := range advisor.deserializer.GetChannelDeserializers() {
		// Deserialize identity
		identity, err := mspManager.DeserializeIdentity(peerIdentity)
		if err != nil {
			log.Logger.Debugf("Failed deserialization identity [% x] on [%s]: [%s]", peerIdentity, chainID, err)
			continue
		}

		return []byte(identity.GetMSPIdentifier())
	}

	log.Logger.Warnf("Peer Identity [% x] cannot be desirialized. No MSP found able to do that.", peerIdentity)

	return nil
}
