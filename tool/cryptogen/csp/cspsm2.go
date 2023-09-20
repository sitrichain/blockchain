package csp

import (
	"crypto"

	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/bccsp/factory"
	"github.com/rongzer/blockchain/common/bccsp/signer"
)

// GeneratePrivateKey creates a private key and stores it in keystorePath
func GeneratePrivateKeySm2(keystorePath string) (bccsp.Key,
	crypto.Signer, error) {

	var err error
	var priv bccsp.Key
	var s crypto.Signer

	opts := &factory.FactoryOpts{
		//ProviderName: "SW",
		ProviderName: "GM",
		SwOpts: &factory.SwOpts{
			HashFamily: "SHA2",
			SecLevel:   256,

			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: keystorePath,
			},
		},
	}
	csp, err := factory.GetBCCSPFromOpts(opts)
	if err == nil {
		// generate a key
		priv, err = csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: false})
		if err == nil && len(keystorePath) > 0 {
			// create a crypto.Signer
			s, err = signer.New(csp, priv)
			if err != nil {
				panic(err)
			}
		}
	}
	return priv, s, err
}
