package ca

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rongzer/blockchain/common/bccsp/utils"
	"github.com/rongzer/blockchain/common/gm"
	"github.com/rongzer/blockchain/common/gm/pkix"
	"github.com/rongzer/blockchain/tool/cryptogen/csp"
)

type CA struct {
	Name string
	Type string
	//SignKey  *ecdsa.PrivateKey
	Signer   crypto.Signer
	SignCert *gm.Certificate
}

// NewCA creates an instance of CA and saves the signing key pair in
// baseDir/name
func NewCA(baseDir, org, name string) (*CA, error) {

	var response error
	var ca *CA

	err := os.MkdirAll(baseDir, 0755)
	if err == nil {
		priv, signer, err := csp.GeneratePrivateKey(baseDir)
		response = err
		if err == nil {
			// get public signing certificate
			ecPubKey, err := csp.GetECPublicKey(priv)
			response = err
			if err == nil {
				template := x509Template()
				//this is a CA
				template.IsCA = true
				template.KeyUsage |= gm.KeyUsageDigitalSignature |
					gm.KeyUsageKeyEncipherment | gm.KeyUsageCertSign |
					gm.KeyUsageCRLSign
				template.ExtKeyUsage = []gm.ExtKeyUsage{gm.ExtKeyUsageAny}

				//set the organization for the subject
				subject := subjectTemplate()
				subject.Organization = []string{org}
				subject.CommonName = name

				template.Subject = subject
				template.SubjectKeyId = priv.SKI()

				x509Cert, err := genCertificateECDSA(baseDir, name, &template, &template,
					ecPubKey, signer)
				response = err
				if err == nil {
					ca = &CA{
						Name:     name,
						Type:     "ECDSA",
						Signer:   signer,
						SignCert: x509Cert,
					}
				}
			}
		}
	}
	return ca, response
}

// SignCertificate creates a signed certificate based on a built-in template
// and saves it in baseDir/name
func (ca *CA) SignCertificate(baseDir, name string, sans []string, pub *ecdsa.PublicKey,
	ku gm.KeyUsage, eku []gm.ExtKeyUsage) (*gm.Certificate, error) {

	template := x509Template()
	template.KeyUsage = ku
	template.ExtKeyUsage = eku

	//set the organization for the subject
	subject := subjectTemplate()
	subject.CommonName = name

	template.Subject = subject
	template.DNSNames = sans

	cert, err := genCertificateECDSA(baseDir, name, &template, ca.SignCert,
		pub, ca.Signer)

	if err != nil {
		return nil, err
	}

	// write artifacts to MSP folders
	//签名验证
	err = cert.CheckSignatureFrom(ca.SignCert)
	if err != nil {
		fmt.Printf("\ncert %s CheckSignatureFrom %s is error\n", cert.Subject.CommonName, ca.SignCert.Subject.CommonName)
		return nil, err
	}

	return cert, nil
}

// default template for X509 subject
func subjectTemplate() pkix.Name {
	return pkix.Name{
		Country:  []string{"US"},
		Locality: []string{"San Francisco"},
		Province: []string{"California"},
	}
}

// default template for X509 certificates
func x509Template() gm.Certificate {

	//generate a serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	now := time.Now()
	//basic template to use
	g := gm.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             now,
		NotAfter:              now.Add(3650 * 24 * time.Hour), //~ten years
		BasicConstraintsValid: true,
	}
	return g

}

// generate a signed X509 certficate using ECDSA
func genCertificateECDSA(baseDir, name string, template, parent *gm.Certificate, pub *ecdsa.PublicKey,
	priv interface{}) (*gm.Certificate, error) {

	//create the gm public cert
	certBytes, err := gm.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		panic(err)
		return nil, err
	}

	//write cert out to file
	fileName := filepath.Join(baseDir, name+"-cert.pem")
	certFile, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	//pem encode the cert
	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certFile.Close()
	if err != nil {
		return nil, err
	}

	x509Cert, err := gm.ParseCertificate(certBytes)

	if err != nil {
		return nil, err
	}
	return x509Cert, nil
}

// LoadCertificateECDSA load a ecdsa cert from a file in cert path
func LoadCertificateECDSA(certPath string) (*gm.Certificate, error) {
	var cert *gm.Certificate
	var err error

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".pem") {
			rawCert, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			block, _ := pem.Decode(rawCert)
			cert, err = utils.DERToX509Certificate(block.Bytes)
		}
		return nil
	}

	err = filepath.Walk(certPath, walkFunc)
	if err != nil {
		return nil, err
	}

	return cert, err
}
