package crypto

import cb "github.com/rongzer/blockchain/protos/common"

// LocalSigner is a temporary stub interface which will be implemented by the local MSP
type LocalSigner interface {
	// NewSignatureHeader creates a SignatureHeader with the correct signing identity and a valid nonce
	NewSignatureHeader() (*cb.SignatureHeader, error)

	// Sign a message which should embed a signature header created by NewSignatureHeader
	Sign(message []byte) ([]byte, error)
}
