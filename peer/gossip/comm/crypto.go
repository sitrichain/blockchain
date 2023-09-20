package comm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/rongzer/blockchain/common/gm"
	"github.com/rongzer/blockchain/common/util"
	util2 "github.com/rongzer/blockchain/peer/gossip/util"
	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func writeFile(filename string, keyType string, data []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: keyType, Bytes: data})
}

func GenerateCertificatesOrPanic() tls.Certificate {
	privKeyFile := fmt.Sprintf("key.%d.priv", util2.RandomUInt64())
	certKeyFile := fmt.Sprintf("cert.%d.pub", util2.RandomUInt64())

	defer os.Remove(privKeyFile)
	defer os.Remove(certKeyFile)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	sn, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		panic(err)
	}
	template := gm.Certificate{
		KeyUsage:     gm.KeyUsageKeyEncipherment | gm.KeyUsageDigitalSignature,
		SerialNumber: sn,
		ExtKeyUsage:  []gm.ExtKeyUsage{gm.ExtKeyUsageServerAuth},
	}
	rawBytes, err := gm.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		panic(err)
	}
	err = writeFile(certKeyFile, "CERTIFICATE", rawBytes)
	if err != nil {
		panic(err)
	}
	privBytes, err := gm.MarshalECPrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	err = writeFile(privKeyFile, "EC PRIVATE KEY", privBytes)
	if err != nil {
		panic(err)
	}
	cert, err := tls.LoadX509KeyPair(certKeyFile, privKeyFile)
	if err != nil {
		panic(err)
	}
	if len(cert.Certificate) == 0 {
		panic(errors.New("Certificate chain is empty"))
	}
	return cert
}

func certHashFromRawCert(rawCert []byte) []byte {
	if len(rawCert) == 0 {
		return nil
	}
	return util.WriteAndSumSha256(rawCert)
}

// ExtractCertificateHash extracts the hash of the certificate from the stream
func extractCertificateHashFromContext(ctx context.Context) []byte {
	pr, extracted := peer.FromContext(ctx)
	if !extracted {
		return nil
	}

	authInfo := pr.AuthInfo
	if authInfo == nil {
		return nil
	}

	tlsInfo, isTLSConn := authInfo.(credentials.TLSInfo)
	if !isTLSConn {
		return nil
	}
	certs := tlsInfo.State.PeerCertificates
	if len(certs) == 0 {
		return nil
	}
	raw := certs[0].Raw
	return certHashFromRawCert(raw)
}
