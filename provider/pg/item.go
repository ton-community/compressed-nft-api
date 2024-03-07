package pg

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	myaddr "github.com/ton-community/compressed-nft-api/address"
	"github.com/ton-community/compressed-nft-api/config"
	"github.com/ton-community/compressed-nft-api/data"
	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type ItemProvider struct {
	pool *pgxpool.Pool
}

func NewItemProvider(pool *pgxpool.Pool) *ItemProvider {
	return &ItemProvider{
		pool: pool,
	}
}

var _ provider.ItemProvider = (*ItemProvider)(nil)

func (ip *ItemProvider) Count() (uint64, error) {
	ctx := context.Background()
	row := ip.pool.QueryRow(ctx, "SELECT COUNT(*) FROM items")
	var count uint64
	err := row.Scan(&count)

	return count, err
}

func makeMetadata(index uint64, owner *address.Address) *data.ItemMetadata {
	return &data.ItemMetadata{
		Owner:             &myaddr.Address{Address: owner},
		IndividualContent: cell.BeginCell().MustStoreStringSnake(strconv.FormatUint(index, 10) + ".json").EndCell(),
		Authority:         &myaddr.Address{Address: config.Config.Authority},
	}
}

func (ip *ItemProvider) GetItem(index uint64) (*data.ItemMetadata, error) {
	ctx := context.Background()
	row := ip.pool.QueryRow(ctx, "SELECT owner FROM items WHERE id = $1", index)
	var addrString string
	err := row.Scan(&addrString)
	if err != nil {
		return nil, err
	}

	addr, err := address.ParseAddr(addrString)
	if err != nil {
		return nil, err
	}

	return makeMetadata(index, addr), nil
}

func (ip *ItemProvider) GetItems(from, count uint64) ([]*data.ItemMetadata, error) {
	ctx := context.Background()
	rows, err := ip.pool.Query(ctx, "SELECT id, owner FROM items WHERE id >= $1 AND id < $2 ORDER BY id ASC", from, from+count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	require := from
	datas := make([]*data.ItemMetadata, 0, count)
	var index uint64
	var addrString string
	for rows.Next() {
		err = rows.Scan(&index, &addrString)
		if err != nil {
			return nil, err
		}

		addr, err := address.ParseAddr(addrString)
		if err != nil {
			return nil, err
		}

		for i := uint64(0); i < index-require; i++ {
			datas = append(datas, nil)
		}
		require = index + 1

		datas = append(datas, makeMetadata(index, addr))
	}

	for i := uint64(0); i < count-uint64(len(datas)); i++ {
		datas = append(datas, nil)
	}

	return datas, nil
}
