package licences

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	adminPubKeyDir	= "admin"
	rsaDir			= "rsacerts"
	formatString 	= "2006-01-02 15:04:05"
	rsaPrivateFile  = "private.pem"
	rsaPublicFile	= "public.pem"
	activeCodeFile  = "codes.txt"
)

type Licences struct {
	path 		string
	adminPubKey *rsa.PublicKey
	rsaPriv 	*rsa.PrivateKey
	rsaPub		*rsa.PublicKey
	activeCode  string
}

func NewLicences(path string) *Licences {
	return &Licences{path: path}
}

func (l *Licences)CheckLicences() error {
	err := l.loadAdminPub()
	if err != nil {
		return err
	}

	err = l.loadRsaCerts()
	if err != nil {
		return err
	}

	err = l.loadActiveCode()
	if err != nil {
		return err
	}

	err = l.verifyActiveCode()
	if err != nil {
		return err
	}

	return nil
}

func (l *Licences) loadAdminPub() error {
	adminPubKeyPath := filepath.Join(l.path, adminPubKeyDir, rsaPublicFile)
	file, err := os.Open(adminPubKeyPath)
	if err != nil {
		return err
	}

	defer file.Close()
	info, _ := file.Stat()
	pubBytes := make([]byte,info.Size())
	_, err = file.Read(pubBytes)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(pubBytes)
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return err
	}

	l.adminPubKey = pub
	return nil
}

func (l *Licences) loadRsaCerts() error {
	rsaPrivateKeyPath := filepath.Join(l.path, rsaDir, rsaPrivateFile)
	file, err := os.Open(rsaPrivateKeyPath)
	if err != nil {
		return err
	}

	defer file.Close()
	info, _ := file.Stat()
	privBytes := make([]byte,info.Size())
	_, err = file.Read(privBytes)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(privBytes)
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	l.rsaPriv = priv
	l.rsaPub = &priv.PublicKey
	return nil
}

func (l *Licences) loadActiveCode() error {
	activeCodePath := filepath.Join(l.path, rsaDir, activeCodeFile)
	file, err := os.Open(activeCodePath)
	if err != nil {
		return err
	}

	defer file.Close()
	info, _ := file.Stat()
	activeCodeBytes := make([]byte,info.Size())
	_, err = file.Read(activeCodeBytes)
	if err != nil {
		return err
	}

	l.activeCode = string(activeCodeBytes)
	return nil
}

func (l *Licences) verifyActiveCode() error {

	codes := strings.Split(l.activeCode, "+")
	if len(codes) != 2 {
		return errors.New("invalid active code")
	}

	cryptoTime := codes[0]
	sig := codes[1]

	hash := sha256.New()
	hash.Write([]byte(cryptoTime))
	bytes := hash.Sum(nil)
	// 对签名进行验签
	err := rsa.VerifyPKCS1v15(l.adminPubKey, crypto.SHA256, bytes, []byte(sig))
	if err != nil {
		return err
	}

	// 激活时间数据进行解密
	timeDuration, err := rsa.DecryptPKCS1v15(rand.Reader, l.rsaPriv, []byte(cryptoTime))
	if err != nil {
		return err
	}

	log.Print("active code time : ", string(timeDuration))

	timeDs := strings.Split(string(timeDuration), "~")
	if len(timeDs) != 2 {
		return errors.New("invalid active duration time")
	}

	timeStart := timeDs[0]
	timeEnd := timeDs[1]

	nowTime := time.Now().Format(formatString)
	if nowTime <= timeStart || nowTime >= timeEnd {
		return errors.New("invalid active time")
	}

	return nil
}


