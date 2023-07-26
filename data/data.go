package data

import (
	"github.com/ton-community/compressed-nft-api/address"
	"github.com/ton-community/compressed-nft-api/types"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type ItemMetadata struct {
	Owner             *address.Address `json:"owner"`
	IndividualContent *cell.Cell       `json:"individual_content"`
}

func (d *ItemMetadata) ToCell() *cell.Cell {
	return cell.BeginCell().MustStoreAddr(d.Owner.Address).MustStoreRef(d.IndividualContent).EndCell()
}

func (d *ItemMetadata) ToNode() types.Node {
	return types.NewNode(d.ToCell().Hash())
}

type ItemData struct {
	Metadata *ItemMetadata `json:"metadata"`
	DataCell *cell.Cell    `json:"data_cell"`
	Index    uint64        `json:"index"`
}

func NewItemData(index uint64, metadata *ItemMetadata) *ItemData {
	return &ItemData{
		Metadata: metadata,
		DataCell: metadata.ToCell(),
		Index:    index,
	}
}
