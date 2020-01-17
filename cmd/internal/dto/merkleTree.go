package dto

// TODO: implement a merkle tree check for downloading blocks from other nodes when rejoining the network
// and offer the updated merkle tree when responding to requests for blockchain history.
// Also implement logic for finding the longest verified block chain when rejoining the network

type BranchData struct {
	Hash1Map       map[string]BranchData `json:"hash1Branch"`
	Hash2Map       map[string]BranchData `json:"hash2Branch"`
	Hash1Block     string                `json:"hash1Block"`
	Hash2Block     string                `json:"hash2Block"`
	NodeTreeHeight int64                 `json:"nodeTreeHeight"`
}

type Merkle struct {
	Top       string                `json:"top"`
	TopHeight int64                 `json:"topHeight"`
	Tree      map[string]BranchData `json:"tree"`
}
