package ssh

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"io/ioutil"

	"github.com/SamsungSLAV/slav/logger"
)

const (
	privKeyType = "RSA PRIVATE KEY"
)

func CreateKeyFiles(key *rsa.PrivateKey, path string) error {
	privDER := x509.MarshalPKCS1PrivateKey(key)
	privBlock := pem.Block{
		Type:    privKeyType,
		Headers: nil,
		Bytes:   privDER,
	}
	privKey := pem.EncodeToMemory(&privBlock)

	pub, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		logger.WithError(err).Error("failed to create pub")
		return err
	}
	pubKey := ssh.MarshalAuthorizedKey(pub)
	if err = ioutil.WriteFile(path, privKey, 0600); err != nil {
		logger.WithError(err).Error("failed to save privkey")
		return err
	}
	if err = ioutil.WriteFile(path+".pub", pubKey, 0644); err != nil {
		logger.WithError(err).Error("failed to save pubkey")
		return err
	}
	return nil
}
