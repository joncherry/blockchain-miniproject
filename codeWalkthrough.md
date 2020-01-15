# Code Walkthrough

`./cmd/blockchainminiproject/main.go` is just wrapping flags around our true program start point, `cmd/internal/resources/server.go`.

func Serve() in server.go inits all our channels and structs that have methods, launches the block builder methods as goroutines, then defines and handles our endpoints. Let's first talk about our simplest endpoint, `/transaction` and then segway into how we build blocks and handle consensus.

## transactions

This blockchain follows a first come first serve ideal. Valid transactions should not get lost, and it should be difficult for them to be dropped. This means the block builder will collect a group of transactions for the block and then work on getting that group of transactions added to the chain until a retry limit. Only after the retry limit is hit may the block builder move on to transactions that came into the pipes later.

Transactions use the public key as user IDs and must be signed by the user losing/giving the coin. The coin amount must be positive for the transaction to be valid and must not be more than the balance of the giving user. Invalid transactions must be marked as dropped when the node is building a block. If a node is sent a block and finds an invalid transaction that is not marked as dropped, it will reject the block. If a node reaches the retry limit when building the block, the group of transactions will be recorded as dropped in a dropped block.