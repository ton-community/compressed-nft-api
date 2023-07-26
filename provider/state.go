package provider

import "github.com/ton-community/compressed-nft-api/types"

type StateProvider interface {
	GetState() (*types.State, error)
	SetState(state *types.State) error
}
