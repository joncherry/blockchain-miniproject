package searchindexing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"
)

// SearchIndexer is the struct that keeps track of where transactions have been saved with a mutex lock for the written-to-file blocks
type SearchIndexer struct {
	mx                   *sync.Mutex
	transactionIDs       map[string]*singleTransactionPath
	keys                 map[string]map[string][]int
	users                map[string]map[string][]int
	BlockChainOutputPath string
}

type singleTransactionPath struct {
	fileName string
	index    int
}

// NewSearchIndexer returns a new empty instance of the SearchIndexer struct with the file output path set.
func NewSearchIndexer(blockChainOutputPath string) *SearchIndexer {
	return &SearchIndexer{
		mx:                   &sync.Mutex{},
		transactionIDs:       make(map[string]*singleTransactionPath),
		keys:                 make(map[string]map[string][]int),
		users:                make(map[string]map[string][]int),
		BlockChainOutputPath: blockChainOutputPath,
	}
}

// Getters

// GetTransactionPathByID returns the file name and block transaction index for the specified transaction ID
func (s *SearchIndexer) GetTransactionPathByID(transactionID string) (string, int, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	path, pathExists := s.transactionIDs[transactionID]
	if !pathExists {
		return "", 0, fmt.Errorf("transaction ID does not exist in blockchain location index")
	}

	return path.fileName, path.index, nil
}

// GetTransactionPathsByKeyword returns a map of filenames and block transaction indexes for the specified keyword
func (s *SearchIndexer) GetTransactionPathsByKeyword(keyword string) (map[string][]int, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	paths, pathExists := s.keys[keyword]
	if !pathExists {
		return nil, fmt.Errorf("keyword does not exist in blockchain location index")
	}

	return paths, nil
}

// GetTransactionPathsByUserID returns a map of filenames and block transaction indexes for the specified user public key
func (s *SearchIndexer) GetTransactionPathsByUserID(userID string) (map[string][]int, error) {
	s.mx.Lock()
	defer s.mx.Unlock()

	paths, pathExists := s.users[userID]
	if !pathExists {
		return nil, fmt.Errorf("userID does not exist in blockchain location index")
	}

	return paths, nil
}

// Setters

// SetTransactionPathByID assigns the filename and block transaction index on the SearchIndexer struct for the specified transaction ID
func (s *SearchIndexer) SetTransactionPathByID(transactionID, fileName string, index int) {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.transactionIDs[transactionID] = &singleTransactionPath{
		fileName: fileName,
		index:    index,
	}
}

// SetTransactionPathsByKeyword assigns the filename and block transaction index on the SearchIndexer struct for the specified keyword
func (s *SearchIndexer) SetTransactionPathsByKeyword(keyword, fileName string, index int) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if len(s.keys[keyword]) == 0 {
		s.keys[keyword] = make(map[string][]int)
	}

	if len(s.keys[keyword][fileName]) == 0 {
		s.keys[keyword][fileName] = make([]int, 0)
	}

	s.keys[keyword][fileName] = append(s.keys[keyword][fileName], index)
}

// SetTransactionPathsByUserID assigns the filename and block transaction index on the SearchIndexer struct for the specified user public key
func (s *SearchIndexer) SetTransactionPathsByUserID(userID, fileName string, index int) {
	s.mx.Lock()
	defer s.mx.Unlock()

	if len(s.users[userID]) == 0 {
		s.users[userID] = make(map[string][]int)
	}

	if len(s.users[userID][fileName]) == 0 {
		s.users[userID][fileName] = make([]int, 0)
	}

	s.users[userID][fileName] = append(s.users[userID][fileName], index)
}

// GetTransactionsFromFiles reads files of the written block chain with the given map of file names and returns the transactions specified by the map transaction index
func (s *SearchIndexer) GetTransactionsFromFiles(fileNames map[string][]int) ([]*dto.TransactionSubmission, error) {
	transactionList := make([]*dto.TransactionSubmission, 0)
	for fileName, transactionIndexes := range fileNames {
		fileTransactions, err := s.GetTransactionsFromSingleFile(fileName, transactionIndexes)
		if err != nil {
			return nil, err
		}
		transactionList = append(transactionList, fileTransactions...)
	}

	return transactionList, nil
}

// GetTransactionsFromSingleFile reads the file specified and returns the transactions from the block in the file specified by transaction indexes
func (s *SearchIndexer) GetTransactionsFromSingleFile(fileName string, getTransactionsAt []int) ([]*dto.TransactionSubmission, error) {
	fileBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", s.BlockChainOutputPath, fileName))
	if err != nil {
		return nil, err
	}

	block := &dto.BlockRequest{}

	err = json.Unmarshal(fileBytes, block)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.TransactionSubmission, 0)
	for _, transactionIndex := range getTransactionsAt {
		result = append(result, block.Transactions[transactionIndex])
	}

	return result, nil
}

// GetWrittenUserBalance uses the built search index to search for all the transactions for the specified user public key in the blocks that have been written to files,
// and returns the sum of the coin gains and losses.
func (s *SearchIndexer) GetWrittenUserBalance(userID string) (userBalance float64, err error) {
	transactionPaths, err := s.GetTransactionPathsByUserID(userID)
	if err != nil {
		return 0, err
	}

	userBalance = 0
	for fileName, transactionIndexes := range transactionPaths {
		fileTransactions, err := s.GetTransactionsFromSingleFile(fileName, transactionIndexes)
		if err != nil {
			return 0, err
		}
		for _, transaction := range fileTransactions {
			if transaction.TransactionStatus == dto.StatusDropped {
				// if the transaction was dropped then ignore its coin amount
				continue
			}
			if userID == transaction.Submitted.From {
				userBalance -= transaction.Submitted.CoinAmount
			} else if userID == transaction.Submitted.To {
				userBalance += transaction.Submitted.CoinAmount
			}
		}
	}

	return userBalance, nil
}
