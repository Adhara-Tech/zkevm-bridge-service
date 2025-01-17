package vectors

// DepositVectorRaw represents the deposit vector
type DepositVectorRaw struct {
	OriginalNetwork    uint32 `json:"originNetwork"`
	TokenAddress       string `json:"tokenAddress"`
	Amount             string `json:"amount"`
	DestinationNetwork uint32 `json:"destinationNetwork"`
	DestinationAddress string `json:"destinationAddress"`
	ExpectedHash       string `json:"leafValue"`
	CurrentHash        string `json:"currentLeafValue"`
	Metadata           string `json:"metadata"`
}

// MTRootVectorRaw represents the root of Merkle Tree
type MTRootVectorRaw struct {
	ExistingLeaves []string         `json:"previousLeafsValues"`
	CurrentRoot    string           `json:"currentRoot"`
	NewLeaf        DepositVectorRaw `json:"newLeaf"`
	NewRoot        string           `json:"newRoot"`
}

// MTClaimVectorRaw represents the merkle proof
type MTClaimVectorRaw struct {
	Deposits     []DepositVectorRaw `json:"leafs"`
	Index        uint32             `json:"index"`
	MerkleProof  []string           `json:"proof"`
	ExpectedRoot string             `json:"root"`
}

// ClaimVectorRaw represents the claim vector
type ClaimVectorRaw struct {
	Index              uint32 `json:"index"`
	OriginalNetwork    uint32 `json:"originNetwork"`
	Token              string `json:"token"`
	Amount             string `json:"amount"`
	DestinationNetwork uint32 `json:"destNetwork"`
	DestinationAddress string `json:"destAddress"`
	BlockNumber        uint64 `json:"blockNumber"`
}

// BlockVectorRaw represents the block vector
type BlockVectorRaw struct {
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	ParentHash  string `json:"parentHash"`
	NetworkID   uint32 `json:"networkID"`
}
