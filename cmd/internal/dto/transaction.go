package dto

type Transaction struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	CoinAmount float64 `json:"coinAmount"`
}

type TransactionSignature struct {
	PublicKey  string `json:"publicKey"`
	BodySigned string `json:"bodySigned"`
}

type TransactionSubmission struct {
	ID                string                `json:"id"`
	Timestamp         string                `json:"timestamp"`
	TransactionStatus string                `json:"transactionStatus"`
	DroppedReason     string                `json:"droppedReason"`
	Signed            *TransactionSignature `json:"sign"`
	Submitted         *Transaction          `json:"submit"`
}

const (
	// StatusDropped indicates a transaction or block has been dropped
	StatusDropped = "dropped"
	// StatusWritten indicates a transaction or block has been accepted and written to the blockchain files
	StatusWritten = "written"
)

/*
example request body
{
	"sign": {
		"publicKey":"testPublicKeySender",
		"bodySigned": "signedResult"
	},
	"submit": {
		"id":"randomstr",
		"key":"searchkey",
		"value":"anything",
		"from":"testPublicKeySender",
		"to":"testPublicKeyRecipient",
		"coinAmount":0.03,
		"timestamp":"a unix timestamp"
	}
}
*/
