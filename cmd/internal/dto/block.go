package dto

type BlockHeader struct {
	PrevBlockHash    string `json:"prev-block-hash"`
	TransactionsHash string `json:"transactions-hash"`
	Time             string `json:"time"`
	Nonce            string `json:"nonce"`
}

type BlockRequest struct {
	OriginNodePublicKey string                   `json:"originNodePublicKey"`
	ProofOfWorkHash     string                   `json:"proofOfWorkHash"`
	Header              *BlockHeader             `json:"header"`
	Transactions        []*TransactionSubmission `json:"transactions"`
}

type NodeSignatures struct {
	Block      *BlockRequest    `json:"block"`
	Signatures []*NodeSignature `json:"nodeSignatures`
}

type NodeSignature struct {
	PublicKey          string
	SignedBlockRequest string
}
