package file

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"

	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/ton-community/compressed-nft-api/types"
)

type StateProvider struct {
	Path string
}

var _ provider.StateProvider = (*StateProvider)(nil)

func (sp *StateProvider) GetState() (*types.State, error) {
	f, err := os.Open(sp.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &types.State{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var s types.State
	err = json.NewDecoder(f).Decode(&s)

	return &s, err
}

func (sp *StateProvider) SetState(state *types.State) error {
	f, err := os.Create(sp.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(state)

	return err
}
