package dto

// BlockHeader defines values and the json of a header for block payloads
type BlockHeader struct {
	PrevBlockHash    string `json:"prev-block-hash"`
	TransactionsHash string `json:"transactions-hash"`
	Time             string `json:"time"`
	Nonce            string `json:"nonce"`
}

// BlockRequest defines the values and json of a block payload
type BlockRequest struct {
	OriginNodePublicKey string                   `json:"originNodePublicKey"`
	ProofOfWorkHash     string                   `json:"proofOfWorkHash"`
	Header              *BlockHeader             `json:"header"`
	Transactions        []*TransactionSubmission `json:"transactions"`
}

// NodeSignatures defines a block with collected node signatures.
type NodeSignatures struct {
	Block      *BlockRequest    `json:"block"`
	Signatures []*NodeSignature `json:"nodeSignatures"`
}

// NodeSignature defines a signature from a node. This could be the node that created the block or the nodes that have signed that they agree the block is valid.
type NodeSignature struct {
	PublicKey          string
	SignedBlockRequest string
}
