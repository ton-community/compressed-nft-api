package state

import (
	"sync"

	"github.com/ton-community/compressed-nft-api/types"
)

type StateHolder struct {
	mu    *sync.RWMutex
	state *FullState
}

func (sh *StateHolder) GetFullState() *FullState {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.state
}

func (sh *StateHolder) SetFullState(fs *FullState) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.state = fs
}

func NewStateHolder(state *types.State) *StateHolder {
	return &StateHolder{
		state: &FullState{
			CurrentState: state,
		},
		mu: &sync.RWMutex{},
	}
}
