package handlers

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/mining"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/autograph"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/searchindexing"
)

type blockSigner struct {
	prevBlockHashRunner *mining.PreviousBlockHashRunner
	searchIndex         *searchindexing.SearchIndexer
	PrivateKey          *rsa.PrivateKey
	PublicKey           *rsa.PublicKey
}

// NewBlockSigner returns an instance of the blockSigner struct for handling the block sign endpoint.
func NewBlockSigner(prevBlockHashRunner *mining.PreviousBlockHashRunner, searchIndex *searchindexing.SearchIndexer) (*blockSigner, error) {
	privateKey, publicKey, err := autograph.NewSig()
	if err != nil {
		return nil, err
	}
	return &blockSigner{
		prevBlockHashRunner: prevBlockHashRunner,
		searchIndex:         searchIndex,
		PrivateKey:          privateKey,
		PublicKey:           publicKey,
	}, nil
}

// VerifyAndSign is the handler for the block sign endpoint. VerifyAndSign will add the node signature to the block if it deems the block is valid.
// To be deemed valid by this node the block must acquire the claim on the previous hash within this node.
func (b *blockSigner) VerifyAndSign(resp http.ResponseWriter, req *http.Request) {
	reqBodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not read request body", "error":"%s"}`, err.Error())))
		return
	}

	signRequest := &dto.NodeSignatures{}
	err = json.Unmarshal(reqBodyBytes, signRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not unmarshal json of request body", "error":"%s"}`, err.Error())))
		return
	}

	blockReqBytes, err := json.Marshal(signRequest.Block)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal json of the block for signing", "error":"%s"}`, err.Error())))
		return
	}

	requestValidated := b.validateSignRequest(resp, signRequest, blockReqBytes)
	if !requestValidated {
		return
	}

	blockValidated := b.validateBlock(resp, signRequest.Block)
	if !blockValidated {
		return
	}

	// if no other node has sent me a block that adds to the previous hash and I have verified this block, claim the previous hash for 120 seconds
	blockReq := signRequest.Block
	err = b.prevBlockHashRunner.SetPrevBlockHashAsClaimedFromSignRequest(blockReq.OriginNodePublicKey, blockReq.ProofOfWorkHash, blockReq.Header.PrevBlockHash)
	if err != nil {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(fmt.Sprintf(`{"message":"the previous block hash is already claimed or trying to claim the wrong prevBlockHash", "error":"%s"}`, err.Error())))
		return
	}

	signedBlockReq, err := autograph.Sign(b.PrivateKey, blockReqBytes)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not sign block with node keys", "error":"%s"}`, err.Error())))
		return
	}

	nodeSigned := &dto.NodeSignature{
		PublicKey:          string(autograph.PublicKeyToBytes(b.PublicKey)),
		SignedBlockRequest: fmt.Sprintf("%x", signedBlockReq),
	}

	signRequest.Signatures = append(signRequest.Signatures, nodeSigned)

	signedResponse, err := json.Marshal(signRequest)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal json of signed block response", "error":"%s"}`, err.Error())))
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(signedResponse)
}

func (b *blockSigner) validateSignRequest(resp http.ResponseWriter, signRequest *dto.NodeSignatures, blockReqBytes []byte) (success bool) {
	// If we were not avoiding using packages here, we would add a 20 minute cache of requests and reject duplicates sent within 20 minutes
	// this would prevent a malicious node from sending the same block and proof of work repeatedly
	// which would tie up this node by holding the prevBlockHash claim indefinitely
	// requestCacheKey := signRequest.Block.OriginNodePublicKey + signRequest.Block.ProofOfWorkHash

	if signRequest.Block.Header.PrevBlockHash != b.prevBlockHashRunner.GetPrevBlockHash() {
		// use status 401 to mean un verified
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(fmt.Sprintf(`{"message":"PrevBlockHash does not match last written block hash!"}`)))
		return
	}

	if signRequest.Block.OriginNodePublicKey == "" {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"OriginNodePublicKey is empty! waddaya tryina pull?"}`)))
		return
	}

	if signRequest.Block.OriginNodePublicKey == string(autograph.PublicKeyToBytes(b.PublicKey)) {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"don't request a signature from your self!"}`)))
		return
	}

	// check that the node submitting the block actually signed the block
	if len(signRequest.Signatures) == 0 || signRequest.Signatures[0].PublicKey != signRequest.Block.OriginNodePublicKey {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(`{"message":"block is not signed by the origin node"}`))
		return
	}

	publicKey := autograph.BytesToPublicKey([]byte(signRequest.Signatures[0].PublicKey))
	signedBlock, err := autograph.SignedBodyToBytes(signRequest.Signatures[0].SignedBlockRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(`{"message":"could not scan the signed block from the origin node with formatting directive '%x'"}`))
		return
	}
	err = autograph.Verify(blockReqBytes, signedBlock, publicKey)
	if err != nil {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(`{"message":"invalid signature from the origin node"}`))
		return
	}

	return true
}

func (b *blockSigner) validateBlock(resp http.ResponseWriter, blockReq *dto.BlockRequest) (success bool) {
	blockHeaderBytes, err := json.Marshal(blockReq.Header)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal json of block header to verify hash", "error":"%s"}`, err.Error())))
		return
	}

	if fmt.Sprintf("%x", sha256.Sum256(blockHeaderBytes)) != blockReq.ProofOfWorkHash || !strings.HasPrefix(blockReq.ProofOfWorkHash, "00000") {
		resp.WriteHeader(http.StatusUnauthorized)
		resp.Write([]byte(`{"message":"invalid proof of work or mismatching block header hash"}`))
		return
	}

	// check for negative ballance of new transactions
	usersBalances := make(map[string]float64)

	for _, transactionSub := range blockReq.Transactions {
		submittedBytes, err := json.Marshal(transactionSub.Submitted)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(fmt.Sprintf(`{"message":"could not marshal json of the transaction for verification", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
			return
		}

		signedBodyBytes, err := autograph.SignedBodyToBytes(transactionSub.BodySigned)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			resp.Write([]byte(fmt.Sprintf(`{"message":"could not scan the signedBody into bytes for verification", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
			return
		}

		pubKey := autograph.BytesToPublicKey([]byte(transactionSub.Submitted.From))

		err = autograph.Verify(submittedBytes, signedBodyBytes, pubKey)
		if err != nil {
			resp.WriteHeader(http.StatusUnauthorized)
			resp.Write([]byte(fmt.Sprintf(`{"message":"could not verify the transaction with the public key", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
			return
		}

		if transactionSub.TransactionStatus == dto.StatusDropped {
			// we won't evaluate the coin amount if the transaction is dropped
			continue
		}

		if transactionSub.Submitted.CoinAmount < 0 {
			resp.WriteHeader(http.StatusUnauthorized)
			resp.Write([]byte(fmt.Sprintf(`{"message":"transaction has negative coin", "transaction.ID":"%s"}`, transactionSub.ID)))
			return
		}

		senderBalance, foundSenderBalance := usersBalances[transactionSub.Submitted.From]
		if !foundSenderBalance {
			senderBalance, err = b.searchIndex.GetWrittenUserBalance(transactionSub.Submitted.From)
			if err != nil {
				resp.WriteHeader(http.StatusUnauthorized)
				resp.Write([]byte(fmt.Sprintf(`{"message":"Could not get the From-User balance from the written blocks", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
				return
			}
			usersBalances[transactionSub.Submitted.From] = senderBalance
		}

		// the receiver might be the sender on following transactions
		receiverBalance, foundReceiverBalance := usersBalances[transactionSub.Submitted.To]
		if !foundReceiverBalance {
			receiverBalance, err = b.searchIndex.GetWrittenUserBalance(transactionSub.Submitted.To)
			if err != nil {
				resp.WriteHeader(http.StatusUnauthorized)
				resp.Write([]byte(fmt.Sprintf(`{"message":"Could not get the From-User balance from the written blocks", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
				return
			}
			usersBalances[transactionSub.Submitted.To] = receiverBalance
		}

		if senderBalance-transactionSub.Submitted.CoinAmount < 0 {
			resp.WriteHeader(http.StatusUnauthorized)
			resp.Write([]byte(fmt.Sprintf(`{"message":"Not enough Coin in user balance", "transaction.ID":"%s"}`, transactionSub.ID)))
			return
		}

		// update the balances map with the new amounts
		// so that we are ready to check the next transaction in this block
		usersBalances[transactionSub.Submitted.From] = senderBalance - transactionSub.Submitted.CoinAmount
		usersBalances[transactionSub.Submitted.To] = receiverBalance + transactionSub.Submitted.CoinAmount
	}

	return true
}
