package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
	"github.com/ton-community/compressed-nft-api/config"
	"github.com/ton-community/compressed-nft-api/updates"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const API_VERSION = 1

var itemCode *cell.Cell
var collectionCode *cell.Cell

func init() {
	itemCodeB, err := hex.DecodeString("b5ee9c7241020e010001dc000114ff00f4a413f4bcf2c80b0102016203020009a11f9fe0050202ce07040201200605001d00f232cfd633c58073c5b3327b5520003b3b513434cffe900835d27080269fc07e90350c04090408f80c1c165b5b60020120090800113e910c30003cb8536002cf0c8871c02497c0f83434c0c05c6c2497c0f83e903e900c7e800c5c75c87e800c7e800c1cea6d003c00812ce3850c1b088d148cb1c17cb865407e90350c0408fc00f801b4c7f4cfe08417f30f45148c2eb8c08c0d0d0d4d60840bf2c9a884aeb8c097c12103fcbc200b0a00727082108b77173505c8cbff5004cf1610248040708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb0002ac3210375e3240135135c705f2e191fa4021f001fa40d20031fa0020d749c200f2e2c4820afaf0801ba121945315a0a1de22d70b01c300209206a19136e220c2fff2e1922194102a375be30d0293303234e30d5502f0030d0c006a26f0018210d53276db103744006d71708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb00007c821005138d91c85009cf16500bcf16712449145446a0708010c8cb055007cf165005fa0215cb6a12cb1fcb3f226eb39458cf17019132e201c901fb001047a4bb9948")
	if err != nil {
		panic(err)
	}

	itemCode, err = cell.FromBOC(itemCodeB)
	if err != nil {
		panic(err)
	}

	collectionCodeB, err := hex.DecodeString("b5ee9c72410225010002a5000114ff00f4a413f4bcf2c80b010201620d0202012006030201480504000db50d9e005f0830001bb60b7e005f08ba0fe03a861f08900201200c0702012009080015b4f47e005f087e00fe01100201480b0a001daf6bf8017c23686987e987fd2018400017ae9ff8017c23e86983ea1840002db8b5d31f002f845d0d431d430d071c8cb0701cf16ccc980202cc170e020120100f0055d37803837c6384b7c2152a9085ccdae37c09078020937c600d274187c217805fc20895d797032fc30f801c02012014110201201312005b00b434800067f48034ffcc0064db084838009be040783508e955088c3c02c0b50c0129510c3c02d67c01167c012000413e1048be403e1089440d007c017cb8197c01a0827270e0321400f3c5b3327c02600201201615003d3e10c4fc01c83c021de0063232c15633c59400fe8084b2daf333325c7ec020001b3e401c1d3232c0b281f2fff274200201201f180201201c190201201b1a002d007232cffe0a33c5b25c083232c044fd003d0032c03260000b343e90350c200201201e1d005d1c013424d4d06e638794c92b5c6c260835c2ffd4013c0125c835c2ffc53c013880f5d3340129013a040d17c1006ea00013007232fff2fff27e40200201202320020120222100393e11fe11be117e10fe10be107232fff2c1f33e1133c5b33333327b552000473b513434ffc07e1874c1c07e18b5007e18fe90007e1935007e1975007e19b5007e19f46001cb43322c700925f03e0d0d3030171b0925f03e0fa4030f00202d31fd33f2282093a3ca6ba8e19345b82100510ff40bef2e066d401d001d3ff3001d4d430f00ae03321820a3cd52cba9d5bf84412c705f2e064d430f00ce0328210693d3950bae3025b840ff2f0824004cf846d08210a8cb00ad708010c8cb055005cf1624fa0214cb6a13cb1fcb3f01cf16c98040fb00829fb365")
	if err != nil {
		panic(err)
	}

	collectionCode, err = cell.FromBOC(collectionCodeB)
	if err != nil {
		panic(err)
	}
}

type updateCellElement struct {
	node   []byte
	update bool
}

func buildUpdateCell(m map[uint64]updateCellElement, index uint64) *cell.Cell {
	if e, ok := m[index]; ok {
		return cell.BeginCell().
			MustStoreBoolBit(true).
			MustStoreBoolBit(!e.update).
			MustStoreSlice(e.node, 256).
			EndCell()
	}

	left := buildUpdateCell(m, 2*index)
	right := buildUpdateCell(m, 2*index+1)

	return cell.BeginCell().
		MustStoreBoolBit(false).
		MustStoreRef(left).
		MustStoreRef(right).
		EndCell()
}

func genupd(cmd *cobra.Command, args []string) error {
	updpath := args[0]
	f, err := os.Open(updpath)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	var m map[string]any
	err = json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	typ := m["type"].(string)
	switch typ {
	case "create":
		if len(args) != 1+7 {
			return errors.New("not enough args to generate a 'create' body; need: owner, collectionmeta, commonitemmeta, royaltybase, royaltyfactor, royaltyrecipient, apilink")
		}

		var upd updates.Create
		err = json.Unmarshal(b, &upd)
		if err != nil {
			return err
		}

		root := big.NewInt(0)
		_, ok := root.SetString(upd.Root, 16)
		if !ok {
			return errors.New("invalid root")
		}

		owner, err := address.ParseAddr(args[1])
		if err != nil {
			return err
		}

		collectionMeta := args[2]
		commonItemMeta := args[3]

		royaltyBase, err := strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			return err
		}

		royaltyFactor, err := strconv.ParseUint(args[5], 10, 64)
		if err != nil {
			return err
		}

		royaltyRecipient, err := address.ParseAddr(args[6])
		if err != nil {
			return err
		}

		if royaltyBase > royaltyFactor || royaltyFactor == 0 {
			return errors.New("invalid royalty params")
		}

		contentCell := cell.BeginCell().
			MustStoreRef(
				cell.BeginCell().
					MustStoreUInt(0x01, 8).
					MustStoreStringSnake(collectionMeta).
					EndCell(),
			).
			MustStoreRef(
				cell.BeginCell().
					MustStoreStringSnake(commonItemMeta).
					EndCell(),
			).
			EndCell()
		royaltyCell := cell.BeginCell().
			MustStoreUInt(royaltyBase, 16).
			MustStoreUInt(royaltyFactor, 16).
			MustStoreAddr(royaltyRecipient).
			EndCell()
		apiDataCell := cell.BeginCell().
			MustStoreUInt(API_VERSION, 8).
			MustStoreRef(
				cell.BeginCell().
					MustStoreStringSnake(args[7]).
					EndCell(),
			).
			EndCell()

		dataCell := cell.BeginCell().
			MustStoreBigUInt(root, 256).
			MustStoreUInt(uint64(upd.Depth), 8).
			MustStoreRef(itemCode).
			MustStoreAddr(owner).
			MustStoreRef(contentCell).
			MustStoreRef(royaltyCell).
			MustStoreRef(apiDataCell).
			EndCell()

		stateInit := &tlb.StateInit{
			Code: collectionCode,
			Data: dataCell,
		}

		stateInitCell, err := tlb.ToCell(stateInit)
		if err != nil {
			return err
		}

		addr := address.NewAddress(0, 0, stateInitCell.Hash())

		link := fmt.Sprintf("ton://transfer/%v?amount=50000000&init=%v", addr.String(), base64.RawURLEncoding.EncodeToString(stateInitCell.ToBOC()))

		fmt.Printf("collection address: %v\n\ndeploy link:\n%v\n", addr.String(), link)
	case "update":
		if len(args) != 1+1 {
			return errors.New("not enough args to create an 'update' body; need: collection")
		}

		var upd updates.Update
		err = json.Unmarshal(b, &upd)
		if err != nil {
			return err
		}

		m := map[uint64]updateCellElement{}

		for _, u := range upd.Updates {
			m[u.Index] = updateCellElement{
				node:   u.Node.Hash[:],
				update: true,
			}
		}

		for i, h := range upd.Hashes {
			m[i] = updateCellElement{
				node:   h.Hash[:],
				update: false,
			}
		}

		bodyCell := cell.BeginCell().
			MustStoreUInt(0x23cd52c, 32).
			MustStoreUInt(0, 64).
			MustStoreRef(buildUpdateCell(m, 1)).
			EndCell()

		link := fmt.Sprintf("ton://transfer/%v?amount=150000000&bin=%v", args[1], base64.RawURLEncoding.EncodeToString(bodyCell.ToBOC()))

		fmt.Printf("update link:\n%v\n", link)
	}

	return nil
}

func add(cmd *cobra.Command, args []string) error {
	config.LoadConfig()

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, config.Config.Database)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, "SELECT COUNT(*) FROM items")
		var index uint64
		err := row.Scan(&index)
		if err != nil {
			return err
		}

		for scanner.Scan() {
			txt := scanner.Text()
			if len(txt) == 0 {
				continue
			}

			addr, err := address.ParseAddr(txt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while parsing address \"%v\": %v", txt, err)
				continue
			}

			_, err = tx.Exec(ctx, "INSERT INTO items (id, owner) VALUES ($1, $2)", index, addr.String())
			if err != nil {
				return err
			}

			index++
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use: "ctl",
	}

	var genupdCmd = &cobra.Command{
		Use:  "genupd updatefile",
		RunE: genupd,
		Args: cobra.MinimumNArgs(1),
	}

	var addCmd = &cobra.Command{
		Use:  "add listfile",
		Args: cobra.ExactArgs(1),
		RunE: add,
	}

	rootCmd.AddCommand(genupdCmd)
	rootCmd.AddCommand(addCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
