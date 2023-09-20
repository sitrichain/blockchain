package sw

import (
	"errors"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/util"
)

type aesPrivateKey struct {
	privKey    []byte
	exportable bool
}

// Bytes converts this key to its byte representation,
// if this operation is allowed.
func (k *aesPrivateKey) Bytes() (raw []byte, err error) {
	if k.exportable {
		return k.privKey, nil
	}

	return nil, errors.New("Not supported.")
}

// SKI returns the subject key identifier of this key.
func (k *aesPrivateKey) SKI() (ski []byte) {
	return util.WriteAndSumSha256(util.ConcatenateBytes([]byte{0x01}, k.privKey))
}

// Symmetric returns true if this key is a symmetric key,
// false if this key is asymmetric
func (k *aesPrivateKey) Symmetric() bool {
	return true
}

// Private returns true if this key is a private key,
// false otherwise.
func (k *aesPrivateKey) Private() bool {
	return true
}

// PublicKey returns the corresponding public key part of an asymmetric public/private key pair.
// This method returns an error in symmetric key schemes.
func (k *aesPrivateKey) PublicKey() (bccsp.Key, error) {
	return nil, errors.New("Cannot call this method on a symmetric key.")
}
