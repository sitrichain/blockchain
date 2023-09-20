package msp

import (
	"os"
	"path/filepath"

	"github.com/rongzer/blockchain/common/gm"
	"github.com/rongzer/blockchain/tool/cryptogen/ca"
	"github.com/rongzer/blockchain/tool/cryptogen/csp"
)

func GenerateLocalMSPSm2(baseDir, name string, sans []string, signCA *ca.CA,
	tlsCA *ca.CA) error {

	// create folder structure
	mspDir := filepath.Join(baseDir, "msp")
	tlsDir := filepath.Join(baseDir, "tls")

	err := createFolderStructure(mspDir, true)
	if err != nil {
		return err
	}

	err = os.MkdirAll(tlsDir, 0755)
	if err != nil {
		return err
	}

	/*
		Create the MSP identity artifacts
	*/
	// get keystore path
	keystore := filepath.Join(mspDir, "keystore")

	// generate private key
	priv, _, err := csp.GeneratePrivateKeySm2(keystore)
	if err != nil {
		return err
	}

	// get public key
	ecPubKey, err := csp.GetECPublicKey(priv)
	if err != nil {
		return err
	}
	// generate X509 certificate using signing CA
	cert, err := signCA.SignCertificate(filepath.Join(mspDir, "signcerts"),
		name, []string{}, ecPubKey, gm.KeyUsageDigitalSignature, []gm.ExtKeyUsage{})
	if err != nil {
		return err
	}

	// the signing CA certificate goes into cacerts
	err = x509Export(filepath.Join(mspDir, "cacerts", x509Filename(signCA.Name)), signCA.SignCert)
	if err != nil {
		return err
	}
	// the TLS CA certificate goes into tlscacerts
	err = x509Export(filepath.Join(mspDir, "tlscacerts", x509Filename(tlsCA.Name)), tlsCA.SignCert)
	if err != nil {
		return err
	}

	// the signing identity goes into admincerts.
	// This means that the signing identity
	// of this MSP is also an admin of this MSP
	// NOTE: the admincerts folder is going to be
	// cleared up anyway by copyAdminCert, but
	// we leave a valid admin for now for the sake
	// of unit tests
	err = x509Export(filepath.Join(mspDir, "admincerts", x509Filename(name)), cert)
	if err != nil {
		return err
	}

	/*
		Generate the TLS artifacts in the TLS folder
	*/

	// generate private key
	tlsPrivKey, _, err := csp.GeneratePrivateKeySm2(tlsDir)
	if err != nil {
		return err
	}
	// get public key
	tlsPubKey, err := csp.GetECPublicKey(tlsPrivKey)
	if err != nil {
		return err
	}
	// generate X509 certificate using TLS CA
	_, err = tlsCA.SignCertificate(filepath.Join(tlsDir),
		name, sans, tlsPubKey, gm.KeyUsageDigitalSignature|gm.KeyUsageKeyEncipherment,
		[]gm.ExtKeyUsage{gm.ExtKeyUsageServerAuth, gm.ExtKeyUsageClientAuth})
	if err != nil {
		return err
	}
	err = x509Export(filepath.Join(tlsDir, "ca.crt"), tlsCA.SignCert)
	if err != nil {
		return err
	}

	// rename the generated TLS X509 cert
	err = os.Rename(filepath.Join(tlsDir, x509Filename(name)),
		filepath.Join(tlsDir, "server.crt"))
	if err != nil {
		return err
	}

	err = keyExport(tlsDir, filepath.Join(tlsDir, "server.key"), tlsPrivKey)
	if err != nil {
		return err
	}

	return nil
}
