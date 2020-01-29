package mining

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/autograph"
	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"
	"github.com/joncherry/blockchain-miniproject/cmd/internal/searchindexing"
)

// PreviousBlockHashRunner is the struct that governs the claims on the previous hash with a mutex lock
type PreviousBlockHashRunner struct {
	mx             *sync.Mutex
	prevHashString string
	claimed        bool
	claimedBy      string
	blockIDHash    string
}

// NewPrevBlockHashRunner returns an empty instance of the PreviousBlockHashRunner struct.
func NewPrevBlockHashRunner() *PreviousBlockHashRunner {
	return &PreviousBlockHashRunner{
		mx:             &sync.Mutex{},
		prevHashString: "",
		claimed:        false,
		claimedBy:      "",
		blockIDHash:    "",
	}
}

// GetPrevBlockHash will use the mutex lock to return the current previous block hash.
func (r *PreviousBlockHashRunner) GetPrevBlockHash() string {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.prevHashString
}

// don't export so that only the file writer can set
func (r *PreviousBlockHashRunner) setPrevBlockHash(hash string) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.prevHashString = hash
}

// GetPrevBlockHashClaimed returns if the previous hash is claimed, which node claimed, and with which block header hash they claimed
func (r *PreviousBlockHashRunner) GetPrevBlockHashClaimed() (bool, string, string) {
	r.mx.Lock()
	defer r.mx.Unlock()
	return r.claimed, r.claimedBy, r.blockIDHash
}

// setPrevBlockHashAsUnclaimed set the previous hash to claimed = false for node and block header hash that was claimed earlier
func (r *PreviousBlockHashRunner) setPrevBlockHashAsUnclaimed(publicKeyStr, proofOfWorkHash string) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if publicKeyStr != r.claimedBy || proofOfWorkHash != r.blockIDHash {
		log.Fatalln("don't call setPrevBlockHashAsUnclaimed() with a block that didn't make the claim")
		return
	}

	r.claimed = false
	r.claimedBy = ""
	r.blockIDHash = ""
}

// SetPrevBlockHashAsClaimed set the previous hash To claimed, which node claimed, and with which block header hash they claimed
func (r *PreviousBlockHashRunner) SetPrevBlockHashAsClaimed(publicKeyStr, proofOfWorkHash, prevBlockHash string) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.claimed == true {
		return fmt.Errorf("SetPrevBlockHashAsClaimed() failed because already claimed")
	}
	if publicKeyStr == "" {
		return fmt.Errorf("SetPrevBlockHashAsClaimed() failed because publicKeyStr was empty")
	}
	if proofOfWorkHash == "" {
		return fmt.Errorf("SetPrevBlockHashAsClaimed() failed because proofOfWorkHash was empty")
	}
	if prevBlockHash != r.prevHashString {
		return fmt.Errorf("SetPrevBlockHashAsClaimed() failed because prevBlockHash did not match")
	}
	r.claimed = true
	r.claimedBy = publicKeyStr
	r.blockIDHash = proofOfWorkHash
	return nil
}

// SetPrevBlockHashAsClaimedFromSignRequest will claim the prevBlockHash for 120 seconds.
// This prevents a node from holding on to the claim forever.
// With no timeout the origin node could hold the claim forever by requesting a signature,
// and then never submitting the block for acceptance.
func (r *PreviousBlockHashRunner) SetPrevBlockHashAsClaimedFromSignRequest(publicKeyStr, proofOfWorkHash, prevBlockHash string) error {
	err := r.SetPrevBlockHashAsClaimed(publicKeyStr, proofOfWorkHash, prevBlockHash)
	if err != nil {
		return err
	}

	// use a goroutine so that we don't sleep for 120 while handling the sign request endpoint
	// we could perhaps call this whole function as a goroutine if we did not need to return the error
	go r.setPrevBlockHashAsUnclaimedFromSignRequest(publicKeyStr, proofOfWorkHash)

	return nil
}

// setPrevBlockHashAsUnclaimedFromSignRequest runs as a goroutine
// so that SetPrevBlockHashAsClaimedFromSignRequest() will not cause the /block-sign handler to sleep for 120 seconds
func (r *PreviousBlockHashRunner) setPrevBlockHashAsUnclaimedFromSignRequest(publicKeyStr, proofOfWorkHash string) {
	time.Sleep(120 * time.Second)

	r.mx.Lock()
	defer r.mx.Unlock()

	if publicKeyStr != r.claimedBy || proofOfWorkHash != r.blockIDHash {
		// setPrevBlockHashAsUnclaimedFromSignRequest doesn't need log.Fatalln()
		// because the claim can be released when the block is accepted before 120 seconds is up
		return
	}

	r.claimed = false
	r.claimedBy = ""
	r.blockIDHash = ""
}

type blockBuilder struct {
	timerChan            chan struct{}
	resetTimerChan       chan struct{}
	transactionsWaiting  chan []*dto.TransactionSubmission
	writeChan            chan *dto.BlockRequest
	prevBlockHashRunner  *PreviousBlockHashRunner
	searchIndex          *searchindexing.SearchIndexer
	maxTransactions      int64
	timeLimitInMinutes   int64
	BlockChainOutputPath string
	privateKey           *rsa.PrivateKey
	publicKey            *rsa.PublicKey
}

// NewBlockBuilder returns a new instance of the blockBuilder struct with the given arguments.
func NewBlockBuilder(
	prevBlockHashRunner *PreviousBlockHashRunner,
	searchIndex *searchindexing.SearchIndexer,
	writeChan chan *dto.BlockRequest,
	maxTransactions,
	timeLimit int64,
	blockChainOutputPath string,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
) *blockBuilder {
	return &blockBuilder{
		timerChan:            make(chan struct{}, 1),
		resetTimerChan:       make(chan struct{}, 1),
		transactionsWaiting:  make(chan []*dto.TransactionSubmission, 0),
		writeChan:            writeChan,
		prevBlockHashRunner:  prevBlockHashRunner,
		searchIndex:          searchIndex,
		maxTransactions:      maxTransactions,
		timeLimitInMinutes:   timeLimit,
		BlockChainOutputPath: blockChainOutputPath,
		privateKey:           privateKey,
		publicKey:            publicKey,
	}
}

// BlockTimer sends a signal on the timerChan when the TIME_LIMIT environment variable minutes have elapsed. The timer is reset every time we receive max transactions for a block.
func (b *blockBuilder) BlockTimer() {
	countDownTimer := b.timeLimitInMinutes * 60
	timer := time.Tick(time.Second)
	for {
		select {
		case <-b.resetTimerChan:
			countDownTimer = b.timeLimitInMinutes * 60
		case <-timer:
			countDownTimer--
			if countDownTimer == 0 {
				countDownTimer = b.timeLimitInMinutes * 60
				b.timerChan <- struct{}{}
			}
		}
	}
}

// BuildNewTransactionsList uses the signal from the timerChan and the MAX_TRANSACTIONS environment variable to group transactions into batches. Each batch will be added to 1 future block or be dropped.
func (b *blockBuilder) BuildNewTransactionsList(tranChan <-chan *dto.TransactionSubmission) {
	blockTransactions := make([]*dto.TransactionSubmission, 0)
	for {
		select {
		case transactionSub := <-tranChan:
			if len(blockTransactions) < int(b.maxTransactions) {
				blockTransactions = append(blockTransactions, transactionSub)
			} else {
				b.resetTimerChan <- struct{}{}
				b.transactionsWaiting <- blockTransactions
				blockTransactions = []*dto.TransactionSubmission{
					transactionSub,
				}
			}
		case <-b.timerChan:
			if len(blockTransactions) > 0 {
				b.transactionsWaiting <- blockTransactions
				blockTransactions = []*dto.TransactionSubmission{}
			}
		}
	}
}

// CreateNewBlocks is the bulk of the node's job because it handles the block mining.
// CreateNewBlocks Will verify there are no negative balances on its list of transactions,
// create a header for the block, find proof of work for that header, and then claim the previous block hash if available.
// CreateNewBlocks will create a header and find proof of work up to 10 times if it can not claim the previous block hash.
// If CreateNewBlocks never succeeds at claiming the previous block hash, the block will be written locally as a dropped block with dropped transactions.
func (b *blockBuilder) CreateNewBlocks() {
	transactionsWaitingLoopCount := 0

TransactionsWaitingLoop:
	for blockTransactions := range b.transactionsWaiting {
		// if a transaction sets a user ballance to negative, mark transaction as dropped
		b.verifySpendIsAllowed(blockTransactions)

		// TODO: add last transaction with self award for mining.
		// Should also verify other nodes are not awarding themselves too much.

		// get blockTransactionsBytes for proof of work
		blockTransactionsBytes, err := json.Marshal(blockTransactions)
		if err != nil {
			log.Fatalln("can't marshal the transactions to create a hash! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
			return
		}

		transactionsHash := fmt.Sprintf("%x", sha256.Sum256(blockTransactionsBytes))

		for retry := 0; retry < 10; retry++ {
			blockHeader := &dto.BlockHeader{
				PrevBlockHash:    b.prevBlockHashRunner.GetPrevBlockHash(),
				TransactionsHash: transactionsHash,
				Time:             strconv.FormatInt(time.Now().Unix(), 10),
			}

			proofOfWorkHash := b.getProofOfWork(blockHeader)

			block := &dto.BlockRequest{
				OriginNodePublicKey: string(autograph.PublicKeyToBytes(b.publicKey)),
				ProofOfWorkHash:     proofOfWorkHash,
				Header:              blockHeader,
				Transactions:        blockTransactions,
			}

			sendOffBlock := b.getSendOffBlock(block)

			// if no other node has sent me a block that adds to the previous hash, claim the previous hash
			err = b.prevBlockHashRunner.SetPrevBlockHashAsClaimed(string(autograph.PublicKeyToBytes(b.publicKey)), sendOffBlock.Block.ProofOfWorkHash, sendOffBlock.Block.Header.PrevBlockHash)
			if err != nil {
				continue
			}

			// simulate network delay in getting the claim to the other nodes
			// time.Sleep(20 * time.Second)

			// send out the block with the proof of work we found in hopes that we can claim the previous hash on other nodes quickly enough
			err = getSignaturesAndDistrubute(sendOffBlock)
			if err != nil {
				log.Println("retrying because not a single node accepted, last response:", err.Error())
				// Not enough other nodes thought that I found Proof of Work first so try again
				b.prevBlockHashRunner.setPrevBlockHashAsUnclaimed(string(autograph.PublicKeyToBytes(b.publicKey)), sendOffBlock.Block.ProofOfWorkHash)
				continue
			}

			// simulate network delay in getting the claim success back from the other nodes
			// time.Sleep(20 * time.Second)

			// if the other nodes agreed that I found proof of work first write the block to the chain
			// and release the claim so that I accept blocks from other nodes again
			b.writeChan <- sendOffBlock.Block
			break TransactionsWaitingLoop
		}

		// write dropped block if we fail all retries
		b.writeDroppedBlock(blockTransactions)

		transactionsWaitingLoopCount++
	}
}

func (b *blockBuilder) verifySpendIsAllowed(blockTransactions []*dto.TransactionSubmission) {
	// check for negative ballance of new transactions
	usersBalances := make(map[string]float64)

	for _, transactionForNewBlock := range blockTransactions {
		if transactionForNewBlock.Submitted.CoinAmount < 0 {
			// we should never reach this point because we check this on the transaction handler
			transactionForNewBlock.TransactionStatus = dto.StatusDropped
			transactionForNewBlock.DroppedReason = "CoinAmount is negative"
			continue
		}

		senderBalance, foundSenderBalance := usersBalances[transactionForNewBlock.Submitted.From]
		if !foundSenderBalance {
			senderBalance, err := b.searchIndex.GetWrittenUserBalance(transactionForNewBlock.Submitted.From)
			if err != nil {
				transactionForNewBlock.TransactionStatus = dto.StatusDropped
				transactionForNewBlock.DroppedReason = err.Error()
				continue
			}
			usersBalances[transactionForNewBlock.Submitted.From] = senderBalance
		}

		// the receiver might be the sender on following transactions
		receiverBalance, foundReceiverBalance := usersBalances[transactionForNewBlock.Submitted.To]
		if !foundReceiverBalance {
			receiverBalance, err := b.searchIndex.GetWrittenUserBalance(transactionForNewBlock.Submitted.To)
			if err != nil {
				transactionForNewBlock.TransactionStatus = dto.StatusDropped
				transactionForNewBlock.DroppedReason = err.Error()
				continue
			}
			usersBalances[transactionForNewBlock.Submitted.To] = receiverBalance
		}

		if senderBalance-transactionForNewBlock.Submitted.CoinAmount < 0 {
			transactionForNewBlock.TransactionStatus = dto.StatusDropped
			transactionForNewBlock.DroppedReason = "Not enough Coin in user balance"
			continue
		}

		// update the balances map with the new amounts
		// so that we are ready to check the next transaction in this block
		usersBalances[transactionForNewBlock.Submitted.From] = senderBalance - transactionForNewBlock.Submitted.CoinAmount
		usersBalances[transactionForNewBlock.Submitted.To] = receiverBalance + transactionForNewBlock.Submitted.CoinAmount
	}
}

// mark the block and transactions as dropped before writing the block to a file
func (b *blockBuilder) writeDroppedBlock(blockTransactions []*dto.TransactionSubmission) {
	for _, droppedTransaction := range blockTransactions {
		droppedTransaction.TransactionStatus = dto.StatusDropped
		droppedTransaction.DroppedReason = "exceeded retries and dropped block"
	}

	// not needed if we aren't going to hash the transactions
	// blockTransactionsBytes, err := json.Marshal(blockTransactions)
	// if err != nil {
	// 	log.Fatalln("can't marshal the transactions to create a hash! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
	// 	return
	// }

	// don't really need this since they are just getting dropped to the nodes local file system
	// transactionsHash := fmt.Sprintf("%x", sha256.Sum256(blockTransactionsBytes))

	blockHeader := &dto.BlockHeader{
		PrevBlockHash: "scrubbed",
		// TransactionsHash: transactionsHash,
		Time: strconv.FormatInt(time.Now().Unix(), 10),
	}

	block := &dto.BlockRequest{
		OriginNodePublicKey: string(autograph.PublicKeyToBytes(b.publicKey)),
		ProofOfWorkHash:     dto.StatusDropped,
		Header:              blockHeader,
		Transactions:        blockTransactions,
	}

	b.writeChan <- block
}

// find a hash of the block header that has enough leading 0's
func (b *blockBuilder) getProofOfWork(blockHeader *dto.BlockHeader) string {
	rand.Seed(time.Now().Unix())
	nonceCount := 100 + rand.Int63()

	proofOfWorkHash := ""
	for {
		nonceCount++
		blockHeader.Nonce = base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(nonceCount, 10)))

		blockHeaderBytes, err := json.Marshal(blockHeader)
		if err != nil {
			log.Fatalln("can't marshal the block header to create a hash! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
			return ""
		}

		proofOfWorkHash = fmt.Sprintf("%x", sha256.Sum256(blockHeaderBytes))
		if strings.HasPrefix(proofOfWorkHash, "00000") {
			return proofOfWorkHash
		}
	}
}

// Sign the block and send it off to the other nodes for signing and adding to the block chain
func (b *blockBuilder) getSendOffBlock(block *dto.BlockRequest) *dto.NodeSignatures {
	blockBytes, err := json.Marshal(block)
	if err != nil {
		log.Fatalln("can't marshal the block struct to write to file! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
		return nil
	}

	signedBlockReq, err := autograph.Sign(b.privateKey, blockBytes)
	if err != nil {
		log.Fatalln("Failed to sign block before sending to the other nodes! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
		return nil
	}

	sendOffBlock := &dto.NodeSignatures{
		Block: block,
		Signatures: []*dto.NodeSignature{
			{
				PublicKey:          string(autograph.PublicKeyToBytes(b.publicKey)),
				SignedBlockRequest: fmt.Sprintf("%x", signedBlockReq),
			},
		},
	}
	return sendOffBlock
}

// WriteBlocks receives accepted and dropped blocks on writeChan,
// verifies the block header has previous hash as the hash of the last written block,
// updates the search index, and finally writes the block to a file.
// The file name for the block is the hash of the block plus the number of the blocks written to file including that block.
func (b *blockBuilder) WriteBlocks() {
	blocksReceived := 0
	previousBlockHash := ""
	for blockToWrite := range b.writeChan {
		blocksReceived++

		if blockToWrite.ProofOfWorkHash != dto.StatusDropped {
			previousBlockHashFromLock := b.prevBlockHashRunner.GetPrevBlockHash()
			if (previousBlockHashFromLock != "" && blockToWrite.Header.PrevBlockHash != previousBlockHashFromLock) ||
				(previousBlockHash != "" && blockToWrite.Header.PrevBlockHash != previousBlockHash) {
				panic(
					fmt.Sprintf(
						"You done Gooofed! Actual previously written block hash: %s, prevBlockHash from lock: %s, trying to write block with prevBlockHash %s in header",
						previousBlockHash,
						previousBlockHashFromLock,
						blockToWrite.Header.PrevBlockHash,
					),
				)
			}
		} else /* block is dropped */ {
			for _, droppedTransaction := range blockToWrite.Transactions {
				if droppedTransaction.TransactionStatus != dto.StatusDropped {
					panic(fmt.Sprintf("How do we have a dropped block with a non dropped transaction? transactionID: %s", droppedTransaction.ID))
				}
			}
		}

		blockBytes, err := json.Marshal(blockToWrite)
		if err != nil {
			log.Fatalln("can't marshal the block struct to write to file! it's the end of the worrrlllldd!!!! aaaaaaaahhhhhhhhh!!!!", err.Error())
			return
		}

		fileName := fmt.Sprintf("%x_%d", sha256.Sum256(blockBytes), blocksReceived)

		// save indexes for searching the block chain files
		for transactionIndex, transaction := range blockToWrite.Transactions {
			// transaction IDs
			b.searchIndex.SetTransactionPathByID(transaction.ID, fileName, transactionIndex)

			// keys
			b.searchIndex.SetTransactionPathsByKeyword(transaction.Submitted.Key, fileName, transactionIndex)

			// users giving coin
			b.searchIndex.SetTransactionPathsByUserID(transaction.Submitted.From, fileName, transactionIndex)

			// users receiving coin
			b.searchIndex.SetTransactionPathsByUserID(transaction.Submitted.To, fileName, transactionIndex)
		}

		err = os.MkdirAll(b.BlockChainOutputPath, 0744)
		if err != nil {
			log.Fatalln(err)
			return
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s.json", b.BlockChainOutputPath, fileName), blockBytes, 0644)
		if err != nil {
			log.Fatalln(err)
			return
		}

		if blockToWrite.ProofOfWorkHash != dto.StatusDropped {
			previousBlockHash = blockToWrite.ProofOfWorkHash
			b.prevBlockHashRunner.setPrevBlockHash(blockToWrite.ProofOfWorkHash)
			b.prevBlockHashRunner.setPrevBlockHashAsUnclaimed(blockToWrite.OriginNodePublicKey, blockToWrite.ProofOfWorkHash)
		}
	}
}

// found a handy permissions chart on stack overflow
/*
	+-----+---+--------------------------+
	| rwx | 7 | Read, write and execute  |
	| rw- | 6 | Read, write              |
	| r-x | 5 | Read, and execute        |
	| r-- | 4 | Read,                    |
	| -wx | 3 | Write and execute        |
	| -w- | 2 | Write                    |
	| --x | 1 | Execute                  |
	| --- | 0 | no permissions           |
	+------------------------------------+

	+------------+------+-------+
	| Permission | Octal| Field |
	+------------+------+-------+
	| rwx------  | 0700 | User  |
	| ---rwx---  | 0070 | Group |
	| ------rwx  | 0007 | Other |
	+------------+------+-------+
*/
