package updates

import "github.com/ton-community/compressed-nft-api/types"

type NodeUpdate struct {
	Index uint64      `json:"index"`
	Node  *types.Node `json:"node"`
}

type Update struct {
	Type         string                 `json:"type"`
	Root         string                 `json:"root"`
	Updates      map[int]NodeUpdate     `json:"updates"`
	Hashes       map[uint64]*types.Node `json:"hashes"`
	NewLastIndex uint64                 `json:"new_last_index"`
}

type Create struct {
	Type      string `json:"type"`
	Root      string `json:"root"`
	Depth     int    `json:"depth"`
	LastIndex uint64 `json:"last_index"`
}
