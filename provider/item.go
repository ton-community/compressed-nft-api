package provider

import "github.com/ton-community/compressed-nft-api/data"

type ItemProvider interface {
	GetItem(index uint64) (*data.ItemMetadata, error)
	GetItems(from uint64, count uint64) ([]*data.ItemMetadata, error)
	Count() (uint64, error)
}
