package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/ton-community/compressed-nft-api/types"
)

type NodeProvider struct {
	pool *pgxpool.Pool
}

func NewNodeProvider(pool *pgxpool.Pool) *NodeProvider {
	return &NodeProvider{
		pool: pool,
	}
}

var _ provider.NodeProvider = (*NodeProvider)(nil)

func (np *NodeProvider) GetNode(index uint64, version int) (types.Node, error) {
	ctx := context.Background()
	row := np.pool.QueryRow(ctx, "SELECT hash, depth FROM nodes WHERE index = $1 AND version <= $2 ORDER BY version DESC LIMIT 1", index, version)
	var hash []byte
	var depth uint16
	err := row.Scan(&hash, &depth)
	if err != nil {
		if err == pgx.ErrNoRows {
			return types.Node{}, provider.ErrNodeNotExist
		}
		return types.Node{}, err
	}

	return types.NewNode(hash, depth), nil
}

func (np *NodeProvider) SetNode(index uint64, version int, node types.Node) error {
	ctx := context.Background()
	_, err := np.pool.Exec(ctx, "INSERT INTO nodes (index, version, hash, depth) VALUES ($1, $2, $3, $4) ON CONFLICT (index, version) DO UPDATE SET hash = EXCLUDED.hash, depth = EXCLUDED.depth", index, version, node.Hash[:], node.CellDepth)

	return err
}
