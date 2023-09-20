package sw

import (
	"crypto/elliptic"
	"crypto/sha512"
	"reflect"

	"github.com/minio/sha256-simd"
	"github.com/rongzer/blockchain/common/bccsp"
	"github.com/rongzer/blockchain/common/errors"
	"github.com/rongzer/blockchain/common/gm"
	"golang.org/x/crypto/sha3"
)

// New returns a new instance of the software-based BCCSP
// set at the passed security level, hash family and KeyStore.
func NewGM(securityLevel int, hashFamily string, keyStore bccsp.KeyStore) (bccsp.BCCSP, error) {
	// Init config
	conf := &config{}
	err := conf.setSecurityLevel(securityLevel, hashFamily)
	if err != nil {
		return nil, errors.ErrorWithCallstack(errors.BCCSP, errors.Internal, "Failed initializing configuration at [%v,%v]", securityLevel, hashFamily).WrapError(err)
	}

	// Check KeyStore
	if keyStore == nil {
		return nil, errors.ErrorWithCallstack(errors.BCCSP, errors.BadRequest, "Invalid bccsp.KeyStore instance. It must be different from nil.")
	}

	// Set the encryptors
	encryptors := make(map[reflect.Type]Encryptor)
	encryptors[reflect.TypeOf(&aesPrivateKey{})] = &aescbcpkcs7Encryptor{}

	// Set the decryptors
	decryptors := make(map[reflect.Type]Decryptor)
	decryptors[reflect.TypeOf(&aesPrivateKey{})] = &aescbcpkcs7Decryptor{}

	// Set the signers
	signers := make(map[reflect.Type]Signer)
	signers[reflect.TypeOf(&ecdsaPrivateKey{})] = &ecdsaSigner{}
	signers[reflect.TypeOf(&rsaPrivateKey{})] = &rsaSigner{}

	// Set the verifiers
	verifiers := make(map[reflect.Type]Verifier)
	verifiers[reflect.TypeOf(&ecdsaPrivateKey{})] = &ecdsaPrivateKeyVerifier{}
	verifiers[reflect.TypeOf(&ecdsaPublicKey{})] = &ecdsaPublicKeyKeyVerifier{}
	verifiers[reflect.TypeOf(&rsaPrivateKey{})] = &rsaPrivateKeyVerifier{}
	verifiers[reflect.TypeOf(&rsaPublicKey{})] = &rsaPublicKeyKeyVerifier{}

	// Set the hashers
	hashers := make(map[reflect.Type]Hasher)
	hashers[reflect.TypeOf(&bccsp.SHAOpts{})] = &hasher{hash: conf.hashFunction}
	hashers[reflect.TypeOf(&bccsp.SHA256Opts{})] = &hasher{hash: sha256.New}
	hashers[reflect.TypeOf(&bccsp.SHA384Opts{})] = &hasher{hash: sha512.New384}
	hashers[reflect.TypeOf(&bccsp.SHA3_256Opts{})] = &hasher{hash: sha3.New256}
	hashers[reflect.TypeOf(&bccsp.SHA3_384Opts{})] = &hasher{hash: sha3.New384}

	impl := &impl{
		conf:       conf,
		ks:         keyStore,
		encryptors: encryptors,
		decryptors: decryptors,
		signers:    signers,
		verifiers:  verifiers,
		hashers:    hashers}

	// Set the key generators
	keyGenerators := make(map[reflect.Type]KeyGenerator)
	keyGenerators[reflect.TypeOf(&bccsp.ECDSAKeyGenOpts{})] = &ecdsaKeyGenerator{curve: conf.ellipticCurve}
	keyGenerators[reflect.TypeOf(&bccsp.ECDSAP256KeyGenOpts{})] = &ecdsaKeyGenerator{curve: gm.P256Sm2()}
	keyGenerators[reflect.TypeOf(&bccsp.ECDSAP384KeyGenOpts{})] = &ecdsaKeyGenerator{curve: elliptic.P384()}
	keyGenerators[reflect.TypeOf(&bccsp.AESKeyGenOpts{})] = &aesKeyGenerator{length: conf.aesBitLength}
	keyGenerators[reflect.TypeOf(&bccsp.AES256KeyGenOpts{})] = &aesKeyGenerator{length: 32}
	keyGenerators[reflect.TypeOf(&bccsp.AES192KeyGenOpts{})] = &aesKeyGenerator{length: 24}
	keyGenerators[reflect.TypeOf(&bccsp.AES128KeyGenOpts{})] = &aesKeyGenerator{length: 16}
	keyGenerators[reflect.TypeOf(&bccsp.RSAKeyGenOpts{})] = &rsaKeyGenerator{length: conf.rsaBitLength}
	keyGenerators[reflect.TypeOf(&bccsp.RSA1024KeyGenOpts{})] = &rsaKeyGenerator{length: 1024}
	keyGenerators[reflect.TypeOf(&bccsp.RSA2048KeyGenOpts{})] = &rsaKeyGenerator{length: 2048}
	keyGenerators[reflect.TypeOf(&bccsp.RSA3072KeyGenOpts{})] = &rsaKeyGenerator{length: 3072}
	keyGenerators[reflect.TypeOf(&bccsp.RSA4096KeyGenOpts{})] = &rsaKeyGenerator{length: 4096}
	impl.keyGenerators = keyGenerators

	// Set the key generators
	keyDerivers := make(map[reflect.Type]KeyDeriver)
	keyDerivers[reflect.TypeOf(&ecdsaPrivateKey{})] = &ecdsaPrivateKeyKeyDeriver{}
	keyDerivers[reflect.TypeOf(&ecdsaPublicKey{})] = &ecdsaPublicKeyKeyDeriver{}
	keyDerivers[reflect.TypeOf(&aesPrivateKey{})] = &aesPrivateKeyKeyDeriver{bccsp: impl}
	impl.keyDerivers = keyDerivers

	// Set the key importers
	keyImporters := make(map[reflect.Type]KeyImporter)
	keyImporters[reflect.TypeOf(&bccsp.AES256ImportKeyOpts{})] = &aes256ImportKeyOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.HMACImportKeyOpts{})] = &hmacImportKeyOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.ECDSAPKIXPublicKeyImportOpts{})] = &ecdsaPKIXPublicKeyImportOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.ECDSAPrivateKeyImportOpts{})] = &ecdsaPrivateKeyImportOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.ECDSAGoPublicKeyImportOpts{})] = &ecdsaGoPublicKeyImportOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.RSAGoPublicKeyImportOpts{})] = &rsaGoPublicKeyImportOptsKeyImporter{}
	keyImporters[reflect.TypeOf(&bccsp.X509PublicKeyImportOpts{})] = &x509PublicKeyImportOptsKeyImporter{bccsp: impl}

	impl.keyImporters = keyImporters

	return impl, nil
}
