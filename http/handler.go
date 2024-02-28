package http

import (
	"encoding/hex"
	"errors"
	"math/bits"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	myaddress "github.com/ton-community/compressed-nft-api/address"
	"github.com/ton-community/compressed-nft-api/data"
	"github.com/ton-community/compressed-nft-api/hash"
	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/ton-community/compressed-nft-api/state"
	"github.com/ton-community/compressed-nft-api/types"
	"github.com/ton-community/compressed-nft-api/updates"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const ITEMS_LIMIT = 10000

type Handler struct {
	StateProvider provider.StateProvider
	NodeProvider  provider.NodeProvider
	ItemProvider  provider.ItemProvider

	StateHolder *state.StateHolder

	Depth int

	NewStates chan *types.State
	Addresses chan *address.Address

	UpdateRecorder updates.Recorder
}

func (h *Handler) getItemsInternal(from uint64, count uint64) (*ItemsResponse, error) {
	stateHolder := h.StateHolder

	state := stateHolder.GetFullState()

	if from > state.CurrentState.LastIndex {
		count = 0
	} else if from+count > state.CurrentState.LastIndex+1 {
		count = state.CurrentState.LastIndex + 1 - from
	}

	ip := h.ItemProvider

	items, err := ip.GetItems(from, count)
	if err != nil {
		return nil, err
	}

	to := count
	if uint64(len(items)) < to {
		to = uint64(len(items))
	}

	fi := make([]*data.ItemData, 0, int(to))
	for i := uint64(0); i < to; i++ {
		fi = append(fi, data.NewItemData(i+from, items[i]))
	}

	return &ItemsResponse{
		Items:     fi,
		Root:      state.CurrentState.Root,
		LastIndex: strconv.FormatUint(state.CurrentState.LastIndex, 10),
	}, nil
}

type ItemsRequest struct {
	From  uint64 `query:"from"`
	Count uint64 `query:"count"`
}

type ItemsResponse struct {
	Items     []*data.ItemData `json:"items"`
	LastIndex string           `json:"last_index"`
	Root      types.NodeHash   `json:"root"`
}

func (h *Handler) getItems(c echo.Context) error {
	ir := new(ItemsRequest)
	if err := c.Bind(ir); err != nil {
		log.Err(err).Msg("bad items request")
		return c.String(http.StatusBadRequest, "bad request")
	}

	if ir.Count == 0 {
		log.Error().Msg("items request has count set to 0")
		return c.String(http.StatusBadRequest, "bad request")
	}

	if ir.Count > ITEMS_LIMIT {
		ir.Count = ITEMS_LIMIT
	}

	resp, err := h.getItemsInternal(ir.From, ir.Count)
	if err != nil {
		log.Err(err).Msg("could not get items")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getItemInternal(state *state.FullState, index uint64) (*ItemResponse, error) {
	ip := h.ItemProvider
	np := h.NodeProvider
	depth := h.Depth

	item, err := ip.GetItem(index)
	if err != nil {
		return nil, err
	}

	curNode := item.ToCell()
	nodeIndex := uint64(1<<depth) + index
	for i := 0; i < depth; i++ {
		nodeIndex ^= 1
		node, err := np.GetNode(nodeIndex, state.CurrentState.Version)
		if err != nil {
			if err == provider.ErrNodeNotExist {
				node = hash.ZeroNodes[i]
			} else {
				return nil, err
			}
		}

		if nodeIndex&1 > 0 {
			curNode = cell.BeginCell().MustStoreRef(curNode).MustStoreRef(node.ToCell()).EndCell()
		} else {
			curNode = cell.BeginCell().MustStoreRef(node.ToCell()).MustStoreRef(curNode).EndCell()
		}

		nodeIndex >>= 1
	}

	return &ItemResponse{
		Item:      data.NewItemData(index, item),
		Root:      state.CurrentState.Root,
		ProofCell: types.MakeMerkleProof(curNode),
	}, nil
}

type ItemRequest struct {
	Index uint64 `param:"index"`
}

type ItemResponse struct {
	Item      *data.ItemData `json:"item"`
	Root      types.NodeHash `json:"root"`
	ProofCell *cell.Cell     `json:"proof_cell"`
}

func (h *Handler) getItem(c echo.Context) error {
	ir := new(ItemRequest)
	if err := c.Bind(ir); err != nil {
		log.Err(err).Msg("bad item request")
		return c.String(http.StatusBadRequest, "bad request")
	}

	sh := h.StateHolder

	state := sh.GetFullState()

	if ir.Index > state.CurrentState.LastIndex {
		log.Error().Msg("item index too large")
		return c.String(http.StatusNotFound, "item index too large")
	}

	resp, err := h.getItemInternal(state, ir.Index)
	if err != nil {
		log.Err(err).Msg("could not get item")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, resp)
}

type StateResponse struct {
	Depth     int                `json:"depth"`
	Capacity  string             `json:"capacity"`
	LastIndex string             `json:"last_index"`
	Root      types.NodeHash     `json:"root"`
	Address   *myaddress.Address `json:"address"`
}

func (h *Handler) getState(c echo.Context) error {
	sh := h.StateHolder

	state := sh.GetFullState()

	resp := &StateResponse{
		Depth:     h.Depth,
		Root:      state.CurrentState.Root,
		Capacity:  strconv.Itoa(1 << h.Depth),
		LastIndex: strconv.FormatUint(state.CurrentState.LastIndex, 10),
		Address:   &myaddress.Address{Address: state.CurrentState.Address.Address},
	}

	return c.JSON(http.StatusOK, resp)
}

func setItemHashes(ip provider.ItemProvider, np provider.NodeProvider, depth int, from, to uint64, version int) error {
	nodeIndexOffset := uint64(1 << depth)
	for i := from; i <= to; i++ {
		item, err := ip.GetItem(i)
		if err != nil {
			return err
		}

		err = np.SetNode(i+nodeIndexOffset, version, item.ToNode())
		if err != nil {
			return err
		}
	}

	return nil
}

func setNodes(np provider.NodeProvider, depth int, nodeFrom, nodeTo uint64, version int) error {
	for d := depth - 1; d >= 0; d-- {
		nodeFrom >>= 1
		nodeTo >>= 1

		for node := nodeFrom; node <= nodeTo; node++ {
			nl, err := np.GetNode(2*node, version)
			if err != nil {
				if err == provider.ErrNodeNotExist {
					nl = hash.ZeroNodes[depth-d-1]
				} else {
					return err
				}
			}

			nr, err := np.GetNode(2*node+1, version)
			if err != nil {
				if err == provider.ErrNodeNotExist {
					nr = hash.ZeroNodes[depth-d-1]
				} else {
					return err
				}
			}

			err = np.SetNode(node, version, hash.Nodes(nl, nr))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *Handler) discoverFirst(c echo.Context) error {
	ip := h.ItemProvider
	np := h.NodeProvider
	depth := h.Depth
	newStates := h.NewStates
	upr := h.UpdateRecorder

	itemCount, err := ip.Count()
	if err != nil {
		return err
	}

	version := 1

	err = setItemHashes(ip, np, depth, 0, itemCount-1, version)
	if err != nil {
		return err
	}

	err = setNodes(np, depth, uint64(1<<depth), uint64(1<<depth)-1+itemCount, version)
	if err != nil {
		return err
	}

	root, err := np.GetNode(1, version)
	if err != nil {
		return err
	}

	state := &types.State{
		LastIndex: itemCount - 1,
		Version:   version,
		Root:      root.Hash[:],
	}

	newStates <- state

	var upd updates.Create
	upd.Type = "create"
	upd.Root = hex.EncodeToString(state.Root)
	upd.Depth = h.Depth
	upd.LastIndex = state.LastIndex

	err = upr.Record(upd, state.Version)

	return err
}

func getNodesToUpdate(start, end uint64, depth int, cn uint64, cd int) []uint64 {
	cns := cn << (depth - cd)
	cne := cns + (1 << (depth - cd)) - 1

	if start > cne || end < cns {
		return nil
	}

	if cns >= start && cne <= end {
		return []uint64{cn}
	}

	left := getNodesToUpdate(start, end, depth, 2*cn, cd+1)
	right := getNodesToUpdate(start, end, depth, 2*cn+1, cd+1)

	return append(left, right...)
}

func getNodesToProvide(nodes []uint64) []uint64 {
	d := map[uint64]byte{}
	for _, n := range nodes {
		d[n] = 1
		nn := n ^ 1
		if _, ok := d[nn]; !ok {
			d[nn] = 2
		}
	}

	for _, n := range nodes {
		n >>= 1
		for n > 1 {
			d[n] = 3
			nn := n ^ 1
			if _, ok := d[nn]; !ok {
				d[nn] = 2
			}
			n >>= 1
		}
	}

	l := make([]uint64, 0)
	for k, v := range d {
		if v == 2 {
			l = append(l, k)
		}
	}

	return l
}

var ErrNothingToRediscover = errors.New("nothing to rediscover")

func (h *Handler) rediscoverFromState(c echo.Context, state *state.FullState) error {
	ip := h.ItemProvider
	np := h.NodeProvider
	depth := h.Depth
	newStates := h.NewStates
	upr := h.UpdateRecorder

	prevLastIndex := state.CurrentState.LastIndex
	newLastIndex, err := ip.Count()
	if err != nil {
		return err
	}
	newLastIndex--

	if newLastIndex == prevLastIndex {
		return ErrNothingToRediscover
	}

	newVersion := state.CurrentState.Version + 1

	err = setItemHashes(ip, np, depth, prevLastIndex+1, newLastIndex, newVersion)
	if err != nil {
		return err
	}

	setNodesFrom := uint64(1<<depth) + prevLastIndex + 1

	err = setNodes(np, depth, setNodesFrom, uint64(1<<depth)+newLastIndex, newVersion)
	if err != nil {
		return err
	}

	root, err := np.GetNode(1, newVersion)
	if err != nil {
		return err
	}

	newState := &types.State{
		LastIndex: newLastIndex,
		Version:   newVersion,
		Root:      root.Hash[:],
	}

	nodesToUpd := getNodesToUpdate(setNodesFrom, uint64(1<<(depth+1))-1, depth, 1, 0)

	nodesToProv := getNodesToProvide(nodesToUpd)

	updatesMap := map[int]updates.NodeUpdate{}
	for _, n := range nodesToUpd {
		nd := 64 - bits.LeadingZeros64(n) - 1
		node, err := np.GetNode(n, newVersion)
		if err != nil {
			if err == provider.ErrNodeNotExist {
				node = hash.ZeroNodes[depth-nd]
			} else {
				return err
			}
		}

		updatesMap[nd] = updates.NodeUpdate{
			Index: n,
			Node:  &node,
		}
	}

	prov := map[uint64]*types.Node{}
	for _, n := range nodesToProv {
		nd := 64 - bits.LeadingZeros64(n) - 1
		node, err := np.GetNode(n, newVersion-1)
		if err != nil {
			if err == provider.ErrNodeNotExist {
				node = hash.ZeroNodes[depth-nd]
			} else {
				return err
			}
		}

		prov[n] = &node
	}

	var upd updates.Update
	upd.Type = "update"
	upd.Root = hex.EncodeToString(newState.Root)
	upd.Updates = updatesMap
	upd.Hashes = prov
	upd.NewLastIndex = newState.LastIndex

	newStates <- newState

	err = upr.Record(upd, newState.Version)

	return err
}

func (h *Handler) rediscover(c echo.Context) error {
	sh := h.StateHolder

	state := sh.GetFullState()

	var err error
	if state.CurrentState.Version == 0 {
		err = h.discoverFirst(c)
	} else {
		err = h.rediscoverFromState(c, state)
	}

	if err != nil {
		log.Err(err).Msg("could not rediscover")
		if err == ErrNothingToRediscover {
			return c.String(http.StatusNotAcceptable, "nothing to rediscover")
		}
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.String(http.StatusOK, "ok")
}

type SetAddrRequest struct {
	AddressString string `param:"addr"`
}

func (h *Handler) setAddr(c echo.Context) error {
	sar := new(SetAddrRequest)
	if err := c.Bind(sar); err != nil {
		log.Err(err).Msg("bad setaddr request")
		return c.String(http.StatusBadRequest, "bad request")
	}

	addrs := h.Addresses

	parsed, err := address.ParseAddr(sar.AddressString)
	if err != nil {
		log.Err(err).Msg("could not parse address")
		return c.String(http.StatusBadRequest, "bad request")
	}

	addrs <- parsed

	return c.String(http.StatusOK, "ok")
}
