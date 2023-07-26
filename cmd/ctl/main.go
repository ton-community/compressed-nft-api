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
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
	"github.com/ton-community/compressed-nft-api/config"
	"github.com/ton-community/compressed-nft-api/migrations"
	"github.com/ton-community/compressed-nft-api/updates"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

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

	collectionCodeB, err := hex.DecodeString("b5ee9c724102290100033a000114ff00f4a413f4bcf2c80b010201620d0202012006030201480504000db50d9e005f0830001bb60b7e005f08ba0fe03a861f08900201200a0702012009080015b4f47e005f087e013e0150001db5dafe005f08da1a61fa61ff4806100201620c0b002dadae98f8017c22e86a18ea186838e4658380e78b6664c00017aefe78017c24686983ea18400202cc1b0e020120140f020148131001893e11fe10aba96fbcb419fe11d4882f3cb81a3e11e93e10aba83e109fc8dc24c8f08022ba04d7c0cd3e10a92baebcb81a009c7c017e106ebcb81a005c7c017e187e19fc00e01101fe53378020f40e6fa18e3b3302d3fff84225a15220ac5207baf2e068f84225a1ae16a005d3ff3021ab0022a55280f00558f004542770f00601a55220f00523f004542260f0068e313022c3ff8e2822ab0023a55270f0055374f005f004542770f00623a55230f0055235f00514f004542260f0064500de4055e25120f00403a5120004455300513e11d48c2f3cb419be1048be403e1089440d007c01fcb8197c022082bebc20321400f3c5b3327c02e002012018150201201716003d3e10c4fc02483c029de0063232c15633c59400fe8084b2daf333325c7ec020001b3e401c1d3232c0b281f2fff274200201201a19002d007232cffe0a33c5b25c083232c044fd003d0032c03260000b343e90350c20020120231c020120201d0201201f1e004f1c24d4c06e638654c82b5c6c2614d03c0154013c0125d4d03c01453c013880e93a040d17c1006ea000113232ffc0a0083d10e00201202221001b0060083d039be87cb81975c2ffe00013007232fff2fff27e40200201202724020120262500413e123e11fe11be117e10fe10be107232fff2c1f33e1133c5b33332fff3327b552000513b513434ffc07e1874c1c07e18b5007e18fe90007e1935007e1975007e19b4ffc07e19f5007e1a346001bf46c2220c700915be001d0d3030171b0915be0fa4030f00201d31fd33f2282093a3ca6ba9c6c31d430d0d3ffd4d430f00ce022820a3cd52cba8e146c21f84412c705f2e064d430d0d4d3ffd430f00de0308210693d395012bae3025b840ff2f0828004cf846d08210a8cb00ad708010c8cb055005cf1624fa0214cb6a13cb1fcb3f01cf16c98040fb0013cb0ca4")
	if err != nil {
		panic(err)
	}

	collectionCode, err = cell.FromBOC(collectionCodeB)
	if err != nil {
		panic(err)
	}
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
		if len(args) != 1+6 {
			return errors.New("not enough args to generate a 'create' body; need: owner, collectionmeta, commonitemmeta, royaltybase, royaltyfactor, royaltyrecipient")
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

		contentCell := cell.BeginCell().MustStoreRef(cell.BeginCell().MustStoreUInt(0x01, 8).MustStoreStringSnake(collectionMeta).EndCell()).MustStoreRef(cell.BeginCell().MustStoreStringSnake(commonItemMeta).EndCell()).EndCell()
		royaltyCell := cell.BeginCell().MustStoreUInt(royaltyBase, 16).MustStoreUInt(royaltyFactor, 16).MustStoreAddr(royaltyRecipient).EndCell()

		dataCell := cell.BeginCell().MustStoreBigUInt(root, 256).MustStoreUInt(uint64(upd.Depth), 8).MustStoreRef(itemCode).MustStoreAddr(owner).MustStoreRef(contentCell).MustStoreRef(royaltyCell).MustStoreUInt(upd.LastIndex, 256).EndCell()

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

		updDict := cell.NewDict(32)
		for d, u := range upd.Updates {
			updDict.SetIntKey(big.NewInt(int64(d)), cell.BeginCell().MustStoreUInt(u.Index, 256).MustStoreSlice(u.Node.Hash[:], 256).EndCell())
		}

		hashDict := cell.NewDict(32)
		for i, h := range upd.Hashes {
			hashDict.SetIntKey(big.NewInt(int64(i)), cell.BeginCell().MustStoreSlice(h.Hash[:], 256).EndCell())
		}

		bodyCell := cell.BeginCell().MustStoreUInt(0x23cd52c, 32).MustStoreUInt(0, 64).MustStoreRef(cell.BeginCell().MustStoreUInt(upd.NewLastIndex, 256).MustStoreRef(updDict.MustToCell()).MustStoreRef(hashDict.MustToCell()).EndCell()).EndCell()

		link := fmt.Sprintf("ton://transfer/%v?amount=1000000000&bin=%v", args[1], base64.RawURLEncoding.EncodeToString(bodyCell.ToBOC()))

		fmt.Printf("update link:\n%v\n", link)
	}

	return nil
}

func migrateFunc(cmd *cobra.Command, args []string) error {
	config.LoadConfig()

	d, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("migrations", d, strings.Replace(config.Config.Database, "postgres", "pgx5", 1))
	if err != nil {
		return err
	}
	defer m.Close()

	m.Up()

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

	var migrateCmd = &cobra.Command{
		Use:  "migrate",
		RunE: migrateFunc,
	}

	var addCmd = &cobra.Command{
		Use:  "add listfile",
		Args: cobra.ExactArgs(1),
		RunE: add,
	}

	rootCmd.AddCommand(genupdCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(addCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
