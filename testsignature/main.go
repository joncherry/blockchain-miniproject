package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
)

// Transaction defines the values and json of the transaction that the from-user signs which creates BodySigned on the TransactionSubmission struct
type Transaction struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	CoinAmount float64 `json:"coinAmount"`
}

func main() {
	rng := rand.Reader

	var err error
	var privateKeyStr string
	var publicKeyStr string
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey
	var body string

	// flags
	flag.StringVar(&body, "body", "", "The body to sign")
	flag.StringVar(&privateKeyStr, "private-key", "", "The private key to sign with")
	flag.StringVar(&publicKeyStr, "public-key", "", "The public key matching the private key to sign with")

	flag.Parse()

	if body == "" {
		fmt.Println("body is empty")
		return
	}

	unmarshalBody := &Transaction{}
	err = json.Unmarshal([]byte(body), unmarshalBody)
	if err != nil {
		fmt.Println("error json unmarshalling the body for formatting", err)
		return
	}

	formattedBody, err := json.Marshal(unmarshalBody)
	if err != nil {
		fmt.Println("error json marshalling the body for formatting", err)
		return
	}

	if privateKeyStr == "" || publicKeyStr == "" {
		privateKey, err = rsa.GenerateKey(rng, 1024)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("generating new keys")
		fmt.Println("privateKey", string(privateKeyToBytes(privateKey)))
		fmt.Println("publicKey", string(publicKeyToBytes(&privateKey.PublicKey)))

		publicKey = &privateKey.PublicKey
	} else {
		privateKey = bytesToPrivateKey([]byte(privateKeyStr))
		publicKey = bytesToPublicKey([]byte(publicKeyStr))
	}

	hash := sha256.New()

	hash.Write(formattedBody)

	hashed := hash.Sum(nil)

	signedThing, err := rsa.SignPSS(rng, privateKey, crypto.SHA256, hashed, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("signed body %x\n", signedThing)

	err = rsa.VerifyPSS(publicKey, crypto.SHA256, hashed, signedThing, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("verified with public key")
}

// these 4 keyToBytes/bytesToKey functions are code from https://gist.github.com/miguelmota/3ea9286bd1d3c2a985b67cac4ba2130a

// PrivateKeyToBytes private key to bytes
func privateKeyToBytes(priv *rsa.PrivateKey) []byte {
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	return privBytes
}

// PublicKeyToBytes public key to bytes
func publicKeyToBytes(pub *rsa.PublicKey) []byte {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})

	return pubBytes
}

// BytesToPrivateKey bytes to private key
func bytesToPrivateKey(priv []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(priv)
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
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return key
}

// BytesToPublicKey bytes to public key
func bytesToPublicKey(pub []byte) *rsa.PublicKey {
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
