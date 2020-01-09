package autograph

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"log"
)

// NewSig creates a new RSA private and publick key
func NewSig() (privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, err error) {
	privateKey, err = rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}
	publicKey = &privateKey.PublicKey
	return privateKey, publicKey, nil
}

// Sign hashes the body with SHA256 and the signs with rsa PSS using the private key generated from NewSig()
func Sign(privateKey *rsa.PrivateKey, body []byte) (signedBody []byte, err error) {
	rng := rand.Reader

	hash := sha256.New()

	hash.Write([]byte(body))

	hashed := hash.Sum(nil)

	signedThing, err := rsa.SignPSS(rng, privateKey, crypto.SHA256, hashed, nil)
	if err != nil {
		return nil, err
	}

	return signedThing, nil
}

// PublicKeyToBytes() is copied code from https://gist.github.com/miguelmota/3ea9286bd1d3c2a985b67cac4ba2130a

// PublicKeyToBytes converts the public key to bytes as a PEM key
func PublicKeyToBytes(pub *rsa.PublicKey) []byte {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		log.Println(err)
		return nil
	}

	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	return pubBytes
}
