package updates

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	myaddr "github.com/ton-community/compressed-nft-api/address"
	"github.com/ton-community/compressed-nft-api/config"
	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/ton-community/compressed-nft-api/state"
	"github.com/ton-community/compressed-nft-api/types"
	"github.com/xssnick/tonutils-go/address"
)

const GET_METHOD_NAME = "get_merkle_root"

func Watcher(newStates <-chan *types.State, addrs <-chan *address.Address, sh *state.StateHolder, sp provider.StateProvider) {
	var addr *address.Address
	var newState *types.State

	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case a := <-addrs:
			addr = a
		case state := <-newStates:
			newState = state
		case <-ticker.C:
			if newState == nil {
				continue
			}

			if addr == nil {
				continue
			}

			rootb, err := getMerkleRoot(addr)
			if err != nil {
				log.Err(err).Msg("could not get merkle root")
				continue
			}

			if !bytes.Equal(rootb, newState.Root.Hash[:]) {
				continue
			}

			newState.Address = &myaddr.Address{Address: addr}

			err = sp.SetState(newState)
			if err != nil {
				log.Err(err).Msg("could not set state")
				continue
			}

			sh.SetFullState(&state.FullState{
				CurrentState: newState,
			})

			log.Info().Int("version", newState.Version).Msg("commited state")

			newState = nil
		}
	}
}

func getMerkleRoot(addr *address.Address) ([]byte, error) {
	var r struct {
		Address *address.Address `json:"address"`
		Method  string           `json:"method"`
		Stack   []any            `json:"stack"`
	}

	r.Address = addr
	r.Method = GET_METHOD_NAME
	r.Stack = []any{}

	b, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(config.Config.Toncenter+"runGetMethod", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m map[string]any

	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	if !m["ok"].(bool) {
		return nil, errors.New("response is not successful")
	}

	ns := (((m["result"].(map[string]any))["stack"].([]any))[0].([]any))[1].(string)

	x := big.NewInt(0)

	x.SetString(ns, 0)

	b = make([]byte, 32)
	x.FillBytes(b)

	return b, nil
}
