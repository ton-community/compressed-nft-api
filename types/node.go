package types

import (
	"encoding/hex"
	"encoding/json"
)

const MAX_LEVELS = 30
const NODE_LENGTH = 32

type Node struct {
	Hash [NODE_LENGTH]byte
}

func (n *Node) UnmarshalJSON(b []byte) error {
	var s string

	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	hash, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	copy(n.Hash[:], hash)

	return nil
}

func (n *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(n.Hash[:]))
}

func NewNode(hash []byte) Node {
	return Node{
		Hash: [NODE_LENGTH]byte(hash),
	}
}
