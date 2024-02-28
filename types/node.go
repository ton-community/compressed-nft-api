package types

import (
	"encoding/hex"
	"encoding/json"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

const MAX_LEVELS = 30
const NODE_LENGTH = 32

type Node struct {
	Hash      [NODE_LENGTH]byte
	CellDepth uint16
}

func (n *Node) UnmarshalJSON(b []byte) error {
	var m map[string]any

	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	hash, err := hex.DecodeString(m["hash"].(string))
	if err != nil {
		return err
	}

	copy(n.Hash[:], hash)

	n.CellDepth = uint16(m["depth"].(float64))

	return nil
}

func (n *Node) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"hash":  hex.EncodeToString(n.Hash[:]),
		"depth": n.CellDepth,
	})
}

func NewNode(hash []byte, depth uint16) Node {
	return Node{
		Hash:      [NODE_LENGTH]byte(hash),
		CellDepth: depth,
	}
}

func (n *Node) ToCell() *cell.Cell {
	return MakePrunedBranch(n.Hash[:], n.CellDepth)
}

func MakePrunedBranch(hash []byte, depth uint16) *cell.Cell {
	c := cell.BeginCell().MustStoreUInt(0x0101, 16).MustStoreSlice(hash, uint(len(hash)*8)).MustStoreUInt(uint64(depth), 16).EndCell()
	c.UnsafeModify(cell.LevelMask{Mask: 1}, true)
	return c
}

func MakeMerkleProof(inner *cell.Cell) *cell.Cell {
	h := inner.Hash(0)
	c := cell.BeginCell().MustStoreUInt(3, 8).MustStoreSlice(h, uint(len(h)*8)).MustStoreUInt(uint64(inner.Depth(0)), 16).MustStoreRef(inner).EndCell()
	c.UnsafeModify(cell.LevelMask{}, true)
	return c
}
