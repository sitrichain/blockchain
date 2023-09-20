package utils

import (
	"github.com/rongzer/blockchain/common/gm"
)

// DERToX509Certificate converts der to x509
func DERToX509Certificate(asn1Data []byte) (*gm.Certificate, error) {
	return gm.ParseCertificate(asn1Data)
}
