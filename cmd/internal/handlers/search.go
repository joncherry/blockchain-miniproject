package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joncherry/blockchain-miniproject/cmd/internal/searchindexing"
)

type searcher struct {
	searchIndex *searchindexing.SearchIndexer
}

// NewSearcher returns an instance of the searcher struct for searching via the built search index
func NewSearcher(searchIndex *searchindexing.SearchIndexer) *searcher {
	return &searcher{
		searchIndex: searchIndex,
	}
}

// Transaction handles the search transaction endpoint.
// Transaction searches for transactions in the written-to-file blocks.
// Transaction searches via the built search index using the transaction ID as the search key.
func (s *searcher) Transaction(resp http.ResponseWriter, req *http.Request) {
	searchTerms := mux.Vars(req)

	transactionID := searchTerms["transaction_id"]
	if len(transactionID) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("transaction ID is empty"))
		return
	}

	if len(transactionID) != 64 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("transaction ID is not 64 characters"))
		return
	}

	fileName, transactionIndex, err := s.searchIndex.GetTransactionPathByID(transactionID)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transaction: %s", err.Error())))
		return
	}

	transactions, err := s.searchIndex.GetTransactionsFromSingleFile(fileName, []int{transactionIndex})
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transaction: %s", err.Error())))
		return
	}

	resultBytes, err := json.Marshal(transactions)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error marshallig transaction to json: %s", err.Error())))
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(resultBytes)
}

// Keyword handles the search keyword endpoint.
// Keyword searches for transactions with the specified value under "key" in the written-to-file blocks.
// Keyword searches via the built search index using the keyword as the search key.
func (s *searcher) Keyword(resp http.ResponseWriter, req *http.Request) {
	searchTerms := mux.Vars(req)

	key := searchTerms["keyword"]
	if len(key) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("keyword is empty"))
		return
	}

	searchPaths, err := s.searchIndex.GetTransactionPathsByKeyword(key)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transactions: %s", err.Error())))
		return
	}

	transactions, err := s.searchIndex.GetTransactionsFromFiles(searchPaths)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transactions: %s", err.Error())))
		return
	}

	resultBytes, err := json.Marshal(transactions)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error marshallig transactions to json: %s", err.Error())))
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(resultBytes)
}

// User handles the search user endpoint.
// User searches for transactions with the user public key as the from-user or to-user in the written-to-file blocks.
// User searches via the built search index using the user public key as the search key.
func (s *searcher) User(resp http.ResponseWriter, req *http.Request) {
	searchTerms := mux.Vars(req)

	userIDHex := searchTerms["user_publickey_hexencoded"]

	userIDbytes, err := hex.DecodeString(userIDHex)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("user ID Public PEM string should be hexadecimal encoded for the url"))
		return
	}

	userID := string(userIDbytes)

	if len(userID) == 0 {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("user ID is empty"))
		return
	}

	if !strings.HasPrefix(userID, "-----BEGIN RSA PUBLIC KEY-----") {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Write([]byte("user ID should be a Public RSA PEM string"))
		return
	}

	searchPaths, err := s.searchIndex.GetTransactionPathsByUserID(userID)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transactions: %s", err.Error())))
		return
	}

	transactions, err := s.searchIndex.GetTransactionsFromFiles(searchPaths)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error finding transactions: %s", err.Error())))
		return
	}

	resultBytes, err := json.Marshal(transactions)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		resp.Write([]byte(fmt.Sprintf("error marshallig transactions to json: %s", err.Error())))
		return
	}

	resp.WriteHeader(http.StatusOK)
	resp.Write(resultBytes)
}
