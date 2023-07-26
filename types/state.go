package types

import "github.com/ton-community/compressed-nft-api/address"

type State struct {
	LastIndex uint64
	Version   int
	Root      Node
	Address   *address.Address
}
