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
	"crypto"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/gm"
	"github.com/rongzer/blockchain/common/log"
	"github.com/rongzer/blockchain/protos/msp"
	"go.uber.org/zap"
)

type identity struct {
	// id contains the identifier (MSPID and identity identifier) for this instance
	id *IdentityIdentifier

	// cert contains the x.509 certificate that signs the public key of this instance
	cert *gm.Certificate

	// this is the public key of this instance
	pk bccsp.Key

	// reference to the MSP that "owns" this identity
	msp *bccspmsp
}

func newIdentity(id *IdentityIdentifier, cert *gm.Certificate, pk bccsp.Key, msp *bccspmsp) (Identity, error) {
	log.Logger.Debugf("Creating identity instance for ID %s", id)

	// Sanitize first the certificate
	cert, err := msp.sanitizeCert(cert)
	if err != nil {
		return nil, err
	}
	return &identity{id: id, cert: cert, pk: pk, msp: msp}, nil
}

// SatisfiesPrincipal returns null if this instance matches the supplied principal or an error otherwise
func (id *identity) SatisfiesPrincipal(principal *msp.MSPPrincipal) error {
	return id.msp.SatisfiesPrincipal(id, principal)
}

// GetIdentifier returns the identifier (MSPID/IDID) for this instance
func (id *identity) GetIdentifier() *IdentityIdentifier {
	return id.id
}

// GetMSPIdentifier returns the MSP identifier for this instance
func (id *identity) GetMSPIdentifier() string {
	return id.id.Mspid
}

// IsValid returns nil if this instance is a valid identity or an error otherwise
func (id *identity) Validate() error {
	return id.msp.Validate(id)
}

// GetOrganizationalUnits returns the OU for this instance
func (id *identity) GetOrganizationalUnits() []*OUIdentifier {
	if id.cert == nil {
		return nil
	}

	cid, err := id.msp.getCertificationChainIdentifier(id)
	if err != nil {
		log.Logger.Errorf("Failed getting certification chain identifier for [%v]: [%s]", id, err)

		return nil
	}

	var res []*OUIdentifier
	for _, unit := range id.cert.Subject.OrganizationalUnit {
		res = append(res, &OUIdentifier{
			OrganizationalUnitIdentifier: unit,
			CertifiersIdentifier:         cid,
		})
	}

	return res
}

// Verify checks against a signature and a message
// to determine whether this identity produced the
// signature; it returns nil if so or an error otherwise
func (id *identity) Verify(msg []byte, sig []byte) error {
	// log.Logger.Infof("Verifying signature")

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return fmt.Errorf("Failed getting hash function options [%s]", err)
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return fmt.Errorf("Failed computing digest [%s]", err)
	}

	if log.IsEnabledFor(zap.DebugLevel) {
		log.Logger.Debugf("Verify: digest = %s", hex.Dump(digest))
		log.Logger.Debugf("Verify: sig = %s", hex.Dump(sig))
	}

	valid, err := id.msp.bccsp.Verify(id.pk, sig, digest, nil)
	if err != nil {
		return fmt.Errorf("Could not determine the validity of the signature, err %s", err)
	} else if !valid {
		return errors.New("The signature is invalid")
	}

	return nil
}

// Serialize returns a byte array representation of this identity
func (id *identity) Serialize() ([]byte, error) {
	// log.Logger.Infof("Serializing identity %s", id.id)

	pb := &pem.Block{Bytes: id.cert.Raw}
	pemBytes := pem.EncodeToMemory(pb)
	if pemBytes == nil {
		return nil, fmt.Errorf("Encoding of identitiy failed")
	}

	// We serialize identities by prepending the MSPID and appending the ASN.1 DER content of the cert
	sId := &msp.SerializedIdentity{Mspid: id.id.Mspid, IdBytes: pemBytes}
	idBytes, err := sId.Marshal()
	if err != nil {
		return nil, fmt.Errorf("Could not marshal a SerializedIdentity structure for identity %s, err %s", id.id, err)
	}

	return idBytes, nil
}

func (id *identity) getHashOpt(hashFamily string) (bccsp.HashOpts, error) {
	switch hashFamily {
	case bccsp.SHA2:
		return bccsp.GetHashOpt(bccsp.SHA256)
	case bccsp.SHA3:
		return bccsp.GetHashOpt(bccsp.SHA3_256)
	}
	return nil, fmt.Errorf("hash famility not recognized [%s]", hashFamily)
}

type signingidentity struct {
	// we embed everything from a base identity
	identity

	// signer corresponds to the object that can produce signatures from this identity
	signer crypto.Signer
}

func newSigningIdentity(id *IdentityIdentifier, cert *gm.Certificate, pk bccsp.Key, signer crypto.Signer, msp *bccspmsp) (SigningIdentity, error) {
	//log.Logger.Infof("Creating signing identity instance for ID %s", id)
	mspId, err := newIdentity(id, cert, pk, msp)
	if err != nil {
		return nil, err
	}
	return &signingidentity{identity: *mspId.(*identity), signer: signer}, nil
}

// Sign produces a signature over msg, signed by this instance
func (id *signingidentity) Sign(msg []byte) ([]byte, error) {
	//log.Logger.Infof("Signing message")

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return nil, fmt.Errorf("Failed getting hash function options [%s]", err)
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return nil, fmt.Errorf("Failed computing digest [%s]", err)
	}

	if len(msg) < 32 {
		log.Logger.Debugf("Sign: plaintext: %X \n", msg)
	} else {
		log.Logger.Debugf("Sign: plaintext: %X...%X \n", msg[0:16], msg[len(msg)-16:])
	}
	log.Logger.Debugf("Sign: digest: %X \n", digest)

	// Sign
	return id.signer.Sign(rand.Reader, digest, nil)
}

func (id *signingidentity) GetPublicVersion() Identity {
	return &id.identity
}
