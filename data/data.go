package data

import (
	"strconv"

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
	c := d.ToCell()
	return types.NewNode(c.Hash(), c.Depth())
}

type ItemData struct {
	Metadata *ItemMetadata `json:"metadata"`
	DataCell *cell.Cell    `json:"data_cell"`
	Index    string        `json:"index"`
}

func NewItemData(index uint64, metadata *ItemMetadata) *ItemData {
	return &ItemData{
		Metadata: metadata,
		DataCell: metadata.ToCell(),
		Index:    strconv.FormatUint(index, 10),
	}
}
