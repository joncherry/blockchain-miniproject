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

type blockAcceptor struct {
	prevBlockHashRunner *mining.PreviousBlockHashRunner
	searchIndex         *searchindexing.SearchIndexer
	PublicKey           *rsa.PublicKey
	writeChan           chan *dto.BlockRequest
}

// NewBlockAcceptor returns a blockAcceptor struct for handling the new block endpoint.
func NewBlockAcceptor(prevBlockHashRunner *mining.PreviousBlockHashRunner, searchIndex *searchindexing.SearchIndexer, publicKey *rsa.PublicKey, writeChan chan *dto.BlockRequest) *blockAcceptor {
	return &blockAcceptor{
		prevBlockHashRunner: prevBlockHashRunner,
		searchIndex:         searchIndex,
		PublicKey:           publicKey,
		writeChan:           writeChan,
	}
}

// VerifyAndAppend handles the new block endpoint. VerifyAndAppend will receive a block on the request and add it to the written block chain if it deems the block is valid.
// To be deemed valid by this node the block must acquire the claim on the previous hash within this node.
func (b *blockAcceptor) VerifyAndAppend(resp http.ResponseWriter, req *http.Request) {
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

	requestValidated := b.validateAcceptRequest(resp, signRequest, blockReqBytes)
	if !requestValidated {
		return
	}

	blockValidated := b.validateBlock(resp, signRequest.Block)
	if !blockValidated {
		return
	}

	// verify we have enough valid signatures from other nodes
	foundValidNodes := 0
	for _, nodeSig := range signRequest.Signatures {
		publicKey := autograph.BytesToPublicKey([]byte(nodeSig.PublicKey))
		signedBlock, err := autograph.SignedBodyToBytes(nodeSig.SignedBlockRequest)
		if err != nil {
			// TODO: maybe do something or print something about the invalid signature
			continue
		}
		err = autograph.Verify(blockReqBytes, signedBlock, publicKey)
		if err != nil {
			// TODO: maybe do something or print something about the invalid signature
			continue
		}
		if foundValidNodes > 5 {
			// TODO: If we know of enough valid nodes that have had a block accepted, reject the signature if we don't recognize the public key from the node
		}
		foundValidNodes++
	}
	// TODO: if we know of enough nodes that have had a block accepted for proof of work then check if we have enough signatures from those known nodes

	// if no other node has sent me a block that adds to the previous hash and I have verified everything, claim the previous hash
	blockReq := signRequest.Block
	err = b.prevBlockHashRunner.SetPrevBlockHashAsClaimed(blockReq.OriginNodePublicKey, blockReq.ProofOfWorkHash, blockReq.Header.PrevBlockHash)
	if err != nil {
		// If the block is not the same block claimed when signing then respond with error
		_, claimedBy, claimedByBlockID := b.prevBlockHashRunner.GetPrevBlockHashClaimed()
		if claimedBy != blockReq.OriginNodePublicKey || claimedByBlockID != blockReq.ProofOfWorkHash {
			resp.WriteHeader(http.StatusUnauthorized)
			resp.Write([]byte(fmt.Sprintf(`{"message":"the previous block hash is already claimed or trying to claim the wrong prevBlockHash", "error":"%s"}`, err.Error())))
			return
		}
	}

	// writing the block will release the claim on the previous block hash
	b.writeChan <- blockReq
}

func (b *blockAcceptor) validateAcceptRequest(resp http.ResponseWriter, signRequest *dto.NodeSignatures, blockReqBytes []byte) (success bool) {
	if signRequest.Block.Header.PrevBlockHash != b.prevBlockHashRunner.GetPrevBlockHash() {
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
		resp.Write([]byte(fmt.Sprintf(`{"message":"don't request acceptance of a block from your self! That makes duplicates and other problems!"}`)))
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

func (b *blockAcceptor) validateBlock(resp http.ResponseWriter, blockReq *dto.BlockRequest) (success bool) {
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
				if transactionSub.Submitted.CoinAmount != 0 {
					resp.WriteHeader(http.StatusUnauthorized)
					resp.Write([]byte(fmt.Sprintf(`{"message":"Could not get the From-User balance from the written blocks", "transaction.ID":"%s", "error":"%s"}`, transactionSub.ID, err.Error())))
					return
				}
				senderBalance = 0
			}
			usersBalances[transactionSub.Submitted.From] = senderBalance
		}

		// the receiver might be the sender on following transactions
		receiverBalance, foundReceiverBalance := usersBalances[transactionSub.Submitted.To]
		if !foundReceiverBalance {
			receiverBalance, err = b.searchIndex.GetWrittenUserBalance(transactionSub.Submitted.To)
			if err != nil {
				receiverBalance = 0
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
