# Code Walkthrough

`./cmd/blockchainminiproject/main.go` is just wrapping flags around our true program start point, `cmd/internal/resources/server.go`.

func Serve() in server.go inits all our channels and structs that have methods, launches the block builder methods as goroutines, then defines and handles our endpoints. Let's first talk about our simplest endpoint, `/transaction` and then segway into how we build blocks and handle consensus.

## transactions

This blockchain follows a first come first serve ideal. Valid transactions should not get lost, and it should be difficult for them to be dropped. This means the block builder will collect a group of transactions for the block and then work on getting that group of transactions added to the chain until a retry limit. Only after the retry limit is hit may the block builder move on to transactions that came into the pipes later.

Transactions use the public key as user IDs and must be signed by the user losing/giving the coin. The coin amount must be positive for the transaction to be valid and must not be more than the balance of the giving user. Invalid transactions must be marked as dropped when the node is building a block. If a node is sent a block and finds an invalid transaction that is not marked as dropped, it will reject the block. If a node reaches the retry limit when building the block, the group of transactions will be recorded as dropped in a dropped block.

Where to look:
- [./cmd/internal/resources/server.go](./cmd/internal/resources/server.go)
- [./cmd/internal/handlers/transaction.go](./cmd/internal/handlers/transaction.go)
- [./cmd/internal/mining/blockBuilding.go](./cmd/internal/mining/blockBuilding.go) BlockTimer(), BuildNewTransactionsList(), CreateNewBlocks()

## blocks

The block builder works on Proof of work for the previous hash. When it finds proof of work for the previous hash, it checks if the claim Mutex has already been claimed by incoming blocks from other nodes, if there is not a claim on the previous hash in its own chain, it will claim the previous hash and send the block out to the network, if it fails to distribute the block to the network, it will retry to find proof of work for its group of transactions. Each retry, it will get the previous hash again for its proof of work, making the assumption the previous hash was updated by incoming blocks.

This should allow all nodes to stay in sync with each other, if a node falls behind and is trying to build on an old previous hash, then it can never get a block accepted by the other nodes, nor can it accept blocks from other nodes, because the previous hashs don't match. So as a network, there are no forks allowed, but as an individual node, it's fork of the chain is the only one that is true. If it can't get 70% to 100% of the network to agree, then it can only write dropped transactions. The one exception to the node only trusting itself would be if the node had down time (not currently a supported option), then it needs to download the difference from the longest chain, which should be the chain that 70% to 100% of the network nodes are using.

Where to look:
- [./cmd/internal/mining/blockBuilding.go](./cmd/internal/mining/blockBuilding.go)
- [./cmd/internal/mining/getSignaturesAndDistribute.go](./cmd/internal/mining/getSignaturesAndDistribute.go)
- [./cmd/internal/handlers/blockSigner.go](./cmd/internal/handlers/blockSigner.go)
- [./cmd/internal/handlers/acceptBlocks.go](./cmd/internal/handlers/acceptBlocks.go)

## search indexer and spending

Each block is saved as a single json file. The search indexer records the file and transaction array index of each transaction. It also gives us a map for keyword and user to transaction indexes. This allows us to search by transaction ID, keyword, and user ID. We can calculate a user balance that has already been written as block files because we can search for transactions by user ID. For the balance on incoming blocks or blocks that we are writing, we take the user balance that has been written, and loop over all transactions to update the user balance in a temporary map.

Where to look (creation):
- [./cmd/internal/searchIndexing/searchIndexer.go](./cmd/internal/searchIndexing/searchIndexer.go)
- [./cmd/internal/mining/blockBuilding.go](./cmd/internal/mining/blockBuilding.go) WriteBlocks()
Where to look (using):
- [./cmd/internal/handlers/search.go](./cmd/internal/handlers/search.go) GetTransactionsFromFiles(), GetTransactionsFromSingleFile(), GetWrittenUserBalance()
- [./cmd/internal/mining/blockBuilding.go](./cmd/internal/mining/blockBuilding.go) CreateNewBlocks()
- [./cmd/internal/handlers/blockSigner.go](./cmd/internal/handlers/blockSigner.go) validateBlock()
- [./cmd/internal/handlers/acceptBlocks.go](./cmd/internal/handlers/acceptBlocks.go) validateBlock()


## Downloading the difference to catch up after downtime, or downloading to become a new node
This feature is still not built. There could be a number of problems, such as if blocks can be written to the chain quickly by hitting the maximum really fast, then downloading the chain from another node might be so slow that you can never catch up and rejoin the network.