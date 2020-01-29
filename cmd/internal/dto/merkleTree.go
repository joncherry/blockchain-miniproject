package dto

// TODO: implement a merkle tree check for downloading blocks from other nodes when rejoining the network
// and offer the updated merkle tree when responding to requests for blockchain history.
// Also implement logic for finding the longest verified block chain when rejoining the network

// BranchData refers to itself as a nested tree of hashes. (I have not tested in referring to yourself even works)
// The nodeTreeHeight defines how many hashes tall each hash pair is.
// When NodeTreeHeight is 0, we look at Hash1Block and Hash2Block and verify against the corresponding blocks.
type BranchData struct {
	Hash1Map       map[string]BranchData `json:"hash1Branch"`
	Hash2Map       map[string]BranchData `json:"hash2Branch"`
	Hash1Block     string                `json:"hash1Block"`
	Hash2Block     string                `json:"hash2Block"`
	NodeTreeHeight int64                 `json:"nodeTreeHeight"`
}

// Merkle defines the top of a tree of hash pairs as map of BranchData where the top hash is the 1 key existing in the map.
// TopHeight should be 1 greater than the NodeTreeHeight of the top pair of hashes.
type Merkle struct {
	Top       string                `json:"top"`
	TopHeight int64                 `json:"topHeight"`
	Tree      map[string]BranchData `json:"tree"`
}
