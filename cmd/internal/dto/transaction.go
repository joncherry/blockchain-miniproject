package dto

// Transaction defines the values and json of the transaction that the from-user signs which creates BodySigned on the TransactionSubmission struct
type Transaction struct {
	Key        string  `json:"key"`
	Value      string  `json:"value"`
	From       string  `json:"from"`
	To         string  `json:"to"`
	CoinAmount float64 `json:"coinAmount"`
}

// TransactionSubmission defines the values and json of a transaction payload
type TransactionSubmission struct {
	ID                string       `json:"id"`
	Timestamp         string       `json:"timestamp"`
	TransactionStatus string       `json:"transactionStatus"`
	DroppedReason     string       `json:"droppedReason"`
	BodySigned        string       `json:"bodySigned"`
	Submitted         *Transaction `json:"submit"`
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
	"bodySigned": "signedResult"
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
