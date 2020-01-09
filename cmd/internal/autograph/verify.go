package autograph

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// Verify verifies the result of Sign() or NewSignedThing(). Errors when verifying fails.
func Verify(body, signedBody []byte, publicKey *rsa.PublicKey) error {
	hash := sha256.New()

	hash.Write([]byte(body))

	hashed := hash.Sum(nil)

	err := rsa.VerifyPSS(publicKey, crypto.SHA256, hashed, signedBody, nil)
	if err != nil {
		return err
	}

	// fmt.Println("Success")
	return nil
}

// SignedBodyToBytes uses fmt Sscanf with scanning directive %x to get the signedBody as bytes
func SignedBodyToBytes(signedBody string) ([]byte, error) {
	signedBodyBytes := []byte{}
	_, err := fmt.Sscanf(signedBody, "%x", &signedBodyBytes)
	if err != nil {
		return nil, fmt.Errorf("could not scan signed-body into bytes")
	}
	return signedBodyBytes, nil
}

// BytesToPublicKey() is copied code from https://gist.github.com/miguelmota/3ea9286bd1d3c2a985b67cac4ba2130a

// BytesToPublicKey takes a PEM key for the public key from the json string converted to bytes and coverts to *rsa.PublicKey
func BytesToPublicKey(pub []byte) *rsa.PublicKey {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
		fmt.Println("public key not ok")
		return nil
	}
	return key
}
