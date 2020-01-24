package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/autograph"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"
)

type transactionRunner struct {
	TranChan chan *dto.TransactionSubmission
}

// NewTransactionRunner initiates transactionRunner with a channel for passing to the transaction queue
func NewTransactionRunner(tranChan chan *dto.TransactionSubmission) *transactionRunner {
	return &transactionRunner{
		TranChan: tranChan,
	}
}

/*
example request:

curl --request POST \
  --url http://127.0.0.1:8080/transaction \
  --header 'content-type: application/json' \
  --data '{
	"sign": {
		"publicKey": "-----BEGIN RSA PUBLIC KEY-----\nMIGf...\n-----END RSA PUBLIC KEY-----",
		"bodySigned": "8a48a..."
	},
	"submit": {
		"key": "searchkey",
		"value": "anything",
		"from": "-----BEGIN RSA PUBLIC KEY-----\nMIGf...\n-----END RSA PUBLIC KEY-----",
		"to": "testPublicKeyRecipient",
		"coinAmount": 0.03
	}
}'

response:

{
  "submission": "success"
}
*/

func (r *transactionRunner) Transaction(resp http.ResponseWriter, req *http.Request) {
	reqBodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not read request body", "error":"%s"}`, err.Error())))
		return
	}

	transactionSub := &dto.TransactionSubmission{}
	err = json.Unmarshal(reqBodyBytes, transactionSub)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not unmarshal json of request body", "error":"%s"}`, err.Error())))
		return
	}

	// don't allow negative coinAmounts, but 0 coin is fine
	if transactionSub.Submitted.CoinAmount < 0 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(`{"message":"don't send a negative coin amount"}`))
		return
	}

	// get the bytes of the submitted transaction for verifying
	submittedBytes, err := json.Marshal(transactionSub.Submitted)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal json of the transaction for verification", "error":"%s"}`, err.Error())))
		return
	}

	// get the signed body as bytes for verifying
	signedBodyBytes, err := autograph.SignedBodyToBytes(transactionSub.Signed.BodySigned)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not scan the signedBody into bytes for verification", "error":"%s"}`, err.Error())))
		return
	}

	pubKey := autograph.BytesToPublicKey([]byte(transactionSub.Signed.PublicKey))

	// verify
	err = autograph.Verify(submittedBytes, signedBodyBytes, pubKey)
	if err != nil {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not verify the transaction with the public key", "error":"%s"}`, err.Error())))
		return
	}

	// add the timestamp and transaction ID
	transactionSub.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	transactionBytes, err := json.Marshal(transactionSub)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal the transaction with the timestamp to create the transaction ID", "error":"%s"}`, err.Error())))
		return
	}
	transactionSub.ID = fmt.Sprintf("%x", sha256.Sum256(transactionBytes))

	r.TranChan <- transactionSub

	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(fmt.Sprintf(`{"submission":"success", "transaction_id":"%s"}`, transactionSub.ID)))
}
