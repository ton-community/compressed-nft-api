package types

import (
	"encoding/hex"
	"encoding/json"

	"github.com/ton-community/compressed-nft-api/address"
)

type State struct {
	LastIndex uint64
	Version   int
	Root      NodeHash
	Address   *address.Address
}

type NodeHash []byte

func (nh *NodeHash) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(*nh))
}

func (nh *NodeHash) UnmarshalJSON(b []byte) error {
	var s string

	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	h, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	*nh = h

	return nil
}
