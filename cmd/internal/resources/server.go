package resources

import (
	"fmt"
	"net/http"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/mining"
	"github.com/joncherry/blockchain-miniproject/cmd/internal/searchindexing"

	"github.com/joncherry/blockchain-miniproject/cmd/internal/dto"

	"github.com/gorilla/mux"
	"github.com/joncherry/blockchain-miniproject/cmd/internal/handlers"
	"github.com/urfave/cli/v2"
)

// Serve listens for requests and uses the appropriate handler functions
func Serve(ctx *cli.Context) error {
	tranChan := make(chan *dto.TransactionSubmission, 100)
	writeChan := make(chan *dto.BlockRequest, 1)

	prevBlockHashRunner := mining.NewPrevBlockHashRunner()

	searchIndex := searchindexing.NewSearchIndexer(ctx.String("blockchain-folder-name"))

	transactionRunner := handlers.NewTransactionRunner(tranChan)
	signer, err := handlers.NewBlockSigner(prevBlockHashRunner, searchIndex)
	if err != nil {
		return err
	}
	acceptor := handlers.NewBlockAcceptor(prevBlockHashRunner, searchIndex, signer.PublicKey, writeChan)

	search := handlers.NewSearcher(searchIndex)

	blockBuilder := mining.NewBlockBuilder(
		prevBlockHashRunner,
		searchIndex,
		writeChan,
		ctx.Int64("max-transactions"),
		ctx.Int64("time-limit"),
		ctx.String("blockchain-folder-name"),
		signer.PrivateKey,
		signer.PublicKey,
	)

	go blockBuilder.BlockTimer()
	go blockBuilder.BuildNewTransactionsList(tranChan)
	go blockBuilder.CreateNewBlocks()
	go blockBuilder.WriteBlocks()

	r := mux.NewRouter()
	r.HandleFunc("/healthcheck", func(resp http.ResponseWriter, req *http.Request) { resp.WriteHeader(http.StatusOK) }).Methods("GET")
	r.HandleFunc("/transaction", transactionRunner.Transaction).Methods("POST")
	r.HandleFunc("/block-sign", signer.VerifyAndSign).Methods("POST")
	r.HandleFunc("/block", acceptor.VerifyAndAppend).Methods("POST")
	r.HandleFunc("/search/transaction/{transaction_id}", search.Transaction).Methods("POST")
	r.HandleFunc("/search/key/{keyword}", search.Keyword).Methods("POST")
	r.HandleFunc("/search/user/{user_publickey_hexencoded}", search.User).Methods("POST")
	// r.HandleFunc("/latest-blocks/{block_id}", blockLibrarian.BlocksAfterBlockID).Methods("POST")

	host := ctx.String("host")
	http.Handle("/", r)
	// quick and dirty port handling for localhost. run up to 7 nodes locally
	if len(host) == 0 {
		// TBD. Can't get them to talk to each other from the terminal yet.
		localHostPorts := []string{":8080", ":8081", ":8082", ":8083", ":8084", ":8085", ":8086"}
		for _, port := range localHostPorts {
			url := fmt.Sprintf("http://127.0.0.1%s/healthcheck", port)
			_, err := http.Get(url)
			if err == nil {
				continue
			}

			// this is manipulating data in a goroutine using a pointer.
			// Usually I wouldn't, but quick and dirty and all. example output folder "written8080"
			blockBuilder.BlockChainOutputPath = blockBuilder.BlockChainOutputPath + port[1:]
			searchIndex.BlockChainOutputPath = searchIndex.BlockChainOutputPath + port[1:]

			fmt.Println("listening on localhost port", port)
			http.ListenAndServe(port, nil)
			break
		}
	} else {
		fmt.Println("listening on", host)
		http.ListenAndServe(host, nil)
	}

	return nil
}
