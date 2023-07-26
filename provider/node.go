package provider

import (
	"errors"

	"github.com/ton-community/compressed-nft-api/types"
)

type NodeProvider interface {
	GetNode(index uint64, version int) (types.Node, error)
	SetNode(index uint64, version int, node types.Node) error
}

var ErrNodeNotExist = errors.New("node does not exist")
