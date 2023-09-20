package rsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"path/filepath"
)

type Rsa struct {
	path string
	priv *rsa.PrivateKey
}

func NewRsa(path string) *Rsa {
	return &Rsa{path: path}
}

func (r *Rsa) GetPrivateKey() (string, error) {

	if r.priv != nil {
		return r.encodeToPemString(r.priv), nil
	}

	exist := r.isExist(r.path)
	if !exist {
		priv, err :=  r.generateKey()
		if err != nil {
			return "", err
		}
		return r.encodeToPemString(priv), nil
	}

	priv, err := r.loadKey()
	if err != nil {
		return "", err
	}
	return r.encodeToPemString(priv), nil
}

func (r *Rsa) GetPublicKey() (string, error) {
	if r.priv == nil {
		_, err := r.GetPrivateKey()
		if err != nil {
			return "", err
		}
	}

	pubKey := &r.priv.PublicKey
	derPubStream := x509.MarshalPKCS1PublicKey(pubKey)
	block := &pem.Block{
		Type:    "RSA PUBLIC KEY",
		Bytes:   derPubStream,
	}

	mem := pem.EncodeToMemory(block)
	return string(mem), nil
}

func (r *Rsa) encodeToPemString(priv *rsa.PrivateKey) string{
	derStream := x509.MarshalPKCS1PrivateKey(priv)
	block := &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Bytes:   derStream,
	}

	mem := pem.EncodeToMemory(block)
	return string(mem)
}

func (r *Rsa) loadKey() (*rsa.PrivateKey, error) {
	privPath := filepath.Join(r.path, "private.pem")
	file, err := os.Open(privPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	info, _ := file.Stat()
	privBytes := make([]byte,info.Size())
	_, err = file.Read(privBytes)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(privBytes)
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	r.priv = priv
	return  priv, nil
}

func (r *Rsa) generateKey() (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	r.priv = priv

	// 公私钥证书写入本地文件夹
	//dir := filepath.Dir(r.path)
	//name := filepath.Base(r.path)
	err = os.MkdirAll(r.path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	derStream := x509.MarshalPKCS1PrivateKey(priv)
	block := &pem.Block{
		Type:    "RSA PRIVATE KEY",
		Bytes:   derStream,
	}
	privPath := filepath.Join(r.path, "private.pem")
	file, err := os.Create(privPath)
	if err != nil {
		return nil, err
	}

	err = pem.Encode(file, block)
	if err != nil {
		return nil, err
	}

	pubKey := &priv.PublicKey
	derPubStream := x509.MarshalPKCS1PublicKey(pubKey)
	block = &pem.Block{
		Type:    "RSA PUBLIC KEY",
		Bytes:   derPubStream,
	}
	pubPath := filepath.Join(r.path, "public.pem")
	file, err = os.Create(pubPath)
	if err != nil {
		return nil, err
	}

	err = pem.Encode(file, block)
	if err != nil {
		return nil, err
	}

	return priv, err
}

func (r *Rsa) isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil{
		if os.IsExist(err){
			return true
		}
		if os.IsNotExist(err){
			return false
		}
		log.Print(err)
		return false
	}
	return true
}

func (r *Rsa) Encrypto(msg []byte) ([]byte, error) {
	pubkey := &r.priv.PublicKey

	return rsa.EncryptPKCS1v15(rand.Reader, pubkey, msg)
}

func (r *Rsa) Decrypto(msg []byte) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, r.priv, msg)
}

func (r *Rsa) SignData(msg []byte) ([]byte, error) {
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, r.priv, crypto.SHA256, bytes)
}

func (r *Rsa) VerifySignature(msg []byte, sig []byte) error {
	pubkey := &r.priv.PublicKey
	hash := sha256.New()
	hash.Write(msg)
	bytes := hash.Sum(nil)
	return rsa.VerifyPKCS1v15(pubkey, crypto.SHA256, bytes, sig)
}




