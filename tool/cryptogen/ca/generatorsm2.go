package ca

import (
	"os"

	"github.com/rongzer/blockchain/common/gm"
	"github.com/rongzer/blockchain/tool/cryptogen/csp"
)

// NewCA creates an instance of CA and saves the signing key pair in
// baseDir/name
func NewCASm2(baseDir, org, name string) (*CA, error) {

	var response error
	var ca *CA

	err := os.MkdirAll(baseDir, 0755)
	if err == nil {
		priv, signer, err := csp.GeneratePrivateKeySm2(baseDir)
		if err != nil {
			panic(err)
		}
		response = err

		// get public signing certificate
		ecPubKey, err := csp.GetECPublicKey(priv)
		if err != nil {
			panic(err)
		}
		response = err

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
		if err != nil {
			panic(err)
		}
		response = err

		ca = &CA{
			Name:     name,
			Type:     "GM",
			Signer:   signer,
			SignCert: x509Cert,
		}
	}
	return ca, response
}
