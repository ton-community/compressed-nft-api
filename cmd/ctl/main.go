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
	"time"

	"github.com/cameo-engineering/tonconnect"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"
	"github.com/ton-community/compressed-nft-api/config"
	"github.com/ton-community/compressed-nft-api/hash"
	"github.com/ton-community/compressed-nft-api/migrations"
	"github.com/ton-community/compressed-nft-api/types"
	"github.com/ton-community/compressed-nft-api/updates"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const API_VERSION = 1

var itemCode *cell.Cell
var collectionCode *cell.Cell

func init() {
	itemCodeB, err := hex.DecodeString("b5ee9c724102130100033b000114ff00f4a413f4bcf2c80b0102016207020201200403001dbc7e7f8017c217c20fc21fc227c2340201580605000db7b07e005f08f0000db5631e005f08b00202ce0b080201200a0900373e11fe11be107232cffe10f3c5be1133c5b33e1173c5b2cff27b552000613b513434cfc07e187e90007e18dc3e188835d2708023859ffe18be90007e1935007e19be90007e1974cfcc3e19e44c38a004bd46c2220c700915be001d0d303fa4030f002f842b38e1c31f84301c705f2e195fa4001f864d401f866fa4030f86570f867f003e002d31f0271b0e30201d33f8210d0c3bfea5230bae302821004ded1485230bae3023082102fcb26a25220ba81211100c03fa8e4031f841c8cbfff843cf1680107082108b7717354015504403804003c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb00e082101f04537a5220bae30282106f89f5e35220ba8e165bf84501c705f2e191f847c000f2e193f823f867f003e08210d136d3b35220bae30230310f0e0d002082105fcc3d14ba93f2c19dde840ff2f0008e31f84422c705f2e191820afaf08070fb028010708210d53276db102455026d830603c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb00009231f84422c705f2e1918010708210d53276db102455026d830603c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb008b02f8648b02f865f00300c632f8445003c705f2e191fa40d4d30030f847f841c8cbfff844cf1613cc12cb3f5210cb0001c30094f84601ccde801078b17082100524c7ae405503804003c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb0000c26c12fa40d4d30030f847f841c8cbff5006cf16f844cf1612cc14cb3f5230cb0003c30096f8465003cc02de801078b17082100dd607e3403514804003c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb0000943031d31f82100524c7ae12ba8e39d33f308010f844708210c18e86d255036d804003c8cb1f12cb3f216eb39301cf179131e2c97105c8cb055004cf1658fa0213cb6accc901fb009130e280f3bdcd")
	if err != nil {
		panic(err)
	}

	itemCode, err = cell.FromBOC(itemCodeB)
	if err != nil {
		panic(err)
	}

	collectionCodeB, err := hex.DecodeString("b5ee9c724102640100076b000114ff00f4a413f4bcf2c80b010201620d0202012006030201480504000db50d9e007f0830001bb60b7e007f08ba0fe03a861f08900201200c0702012009080015b4f47e007f087e00de00f00201480b0a001daf6bf801fc23686987e987fd2018400017ae9ff801fc23e86983ea1840002db8b5d31f003f845d0d431d430d071c8cb0701cf16ccc980202cc1b0e020120140f0201201110002d4d0d4d430f84112f00a01f009f8424330f00df861f0048020120131200933b68bb7ec8b5d97000238888be4000be4004aea4d6f6cc780075ce7cb81ab4c3cc2040406ebcb81abc00bcb81aa38640b40074007500b5092950cc3c0340750c00750c00a944bc0378a0003d3e107c02be1094883c02fc0160827270e032140133c584b30073c5b27c02200201201815020120171600372964c830bfe38480b414c4ab5c6c25350c750c24b50c3880a97a16e00011007c02562ebcb81a600201201a19002335ce7cb819f4c1c07000fcb81a3534ffcc20003d3e10c4fc01883c01dde0063232c15633c59400fe8084b2daf333325c7ec020020120231c020120201d0201201f1e001b3e401c1d3232c0b281f2fff27420002d007232cffe0a33c5b25c083232c044fd003d0032c032600201202221000f343e90353e900c2000393e11fe11be117e10fe10be107232fff2c1f33e1133c5b33333327b55200201206224020120262500473b513434ffc07e1874c1c07e18b5007e18fe90007e1935007e1975007e19b5007e19f4600119221e3d039be87cb81af5c2ffe0270201c7432802012034290201202d2a0201482c2b00412924e95a1b53a326c469c2934ef9d6d61276d45b26d175be95e0954d4ab45f8ca00041372c457f2b3c9896bb269fbf1f42e0966fb062dc8d85968ee9bc1d90ddbf5e5f60020120312e020120302f0041141ae9ccd6f0c29613d6fd6a779909e34d406b6fa0d422c2b1117f91f57165dee00041117958c6c8e399b60950abd4e8c0012da388b971a3d5de2b9f512d58718a98e4e002012033320041067999a06a7a50296ddd3845a4bb26ccafcfd50def3d413751a63cb54e0670cd2000413319f3ec0c7df03ce75a18abec64f8c965250a7e8a193af9a051f4ebc2f64cad600201203c35020120393602012038370041247cc2769bb44e9a2d04e2bc8166036cd9ae1875f3bea688c175fe950d232c6da00041132b8507aa832e81f3af722b0786cd94fb5c3db44e0247a859f665ab754969b5a00201203b3a004118f939b4855228f5a4b965fdd0c446e9204c533179d034d198eb60fd1ef77118e000412725bce396275113929e99995a04bfe8ed9f9910430632b636c98b6fbdae50b0e0020120403d0201203f3e00410f2003946b7df8b0272279a045e9e916ae8fb0ca48f406684a3ba5611585756f20004137036394a4ad9fabf4098f700c1a2b26a2c6a4122225fbbd419a2d3d3f45206a200201204241004107c5c013022e6c980851802f902c24e93a3e7e6b0f93e8c21cd14ef202e770efe000413ee8a745d3d291599269536c1630915e4daddb3d5acd3a97e400e3b7303569a76002012053440201204c450201204946020120484700410c9c73124c98bdef3465ecc20fd915aec03169e9796144f012a825c61bf7dac560004136b354771fae9458b017731834835d5d2017763a07d8bb3d79851dd790725689a00201204b4a00412bc9e953d5087aa25544752426d41f39547aa22c6073abd51261f8bb8d09c0616000411a7026bef2a1f6578b692fa5e728a570e195a0b7f562e9ef54471f102ccf0c8020020120504d0201204f4e00412b466b46039f7156683b6652fdadab78382b0d79bc1deb6cad3eb5a5350e5bb1a000412a0b55af8a33479ec565a21ba52c9c9139b3f6079b73a0e8de242fde239b9fab2002012052510041033ed2411b5fc91088caf7eb13da9d48842f0809bf1c37b83a7eff511ff19b0f200041163a81eada1caddedfb9cdb455e771080bb9e288b3efe5e8082050af0463d915a00201205b540201205855020120575600412dfbdcbddd8301f96afb72bc36dd2a57e1eb141b22eaf15e45d4d6040547c38660004107cc387c90de5176110fcc8a63a99629b3d93f0b138795b2ec3f4df9bf756be7a00201205a590041139f226107c6bea682dc41b23360b2e736ae1f2ac9d33186ee539ca6c9a8c451a000413e6f4348078f15e0fd3977e7c5f5d61d417a0b0007960c5682eea515f6eaf01fe00201205f5c0201205e5d00410526d4e74a28305ad88dc6f5bafb655c54075306c564211dadb5654e7b9c92a86000412b04ac54adb5437b7db87fc2373d926ddb3455784e7d91f44d0c5983c1cc87b5a0020120616000413c92216fe16fce3d2afca7c3aca45d10e5e1d1903fe79dfd549b18600323b7c2a000410e1254b628016fc636df30b61c6fe19c7f8e8d19631954f5d218ce28106be18a2001bd43322c700925f03e0d0d3030171b0925f03e0fa4030f00302d31fd33f2282093a3ca6ba8e12345b82100510ff40bef2e066d3ffd430f00ce03321820a3cd52cba9d5bf84412c705f2e064d430f00ee0328210693d3950bae3025b840ff2f0863004cf846d08210a8cb00ad708010c8cb055005cf1624fa0214cb6a13cb1fcb3f01cf16c98040fb00d3d2222b")
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
	depth  uint16
	update bool
}

func buildUpdateCell(m map[uint64]updateCellElement, index uint64, depth int) (*cell.Cell, *cell.Cell) {
	if e, ok := m[index]; ok {
		var old *cell.Cell
		if e.update {
			old = hash.ZeroNodes[depth].ToCell()
		} else {
			old = types.MakePrunedBranch(e.node, e.depth)
		}
		return old, types.MakePrunedBranch(e.node, e.depth)
	}

	oldLeft, newLeft := buildUpdateCell(m, 2*index, depth-1)
	oldRight, newRight := buildUpdateCell(m, 2*index+1, depth-1)

	return cell.BeginCell().MustStoreRef(oldLeft).MustStoreRef(oldRight).EndCell(), cell.BeginCell().MustStoreRef(newLeft).MustStoreRef(newRight).EndCell()
}

func sendMessage(addr string, amount string, body, stateInit *cell.Cell) error {
	link := fmt.Sprintf("ton://transfer/%v?amount=%v", addr, amount)
	if body != nil {
		link += fmt.Sprintf("&bin=%v", base64.RawURLEncoding.EncodeToString(body.ToBOC()))
	}
	if stateInit != nil {
		link += fmt.Sprintf("&init=%v", base64.RawURLEncoding.EncodeToString(stateInit.ToBOC()))
	}
	fmt.Printf("ton deeplink:\n%v\n\n", link)

	tcs, err := tonconnect.NewSession()
	if err != nil {
		return err
	}

	connreq, err := tonconnect.NewConnectRequest("https://raw.githubusercontent.com/ton-defi-org/tonconnect-manifest-temp/main/tonconnect-manifest.json")
	if err != nil {
		return err
	}

	deeplink, err := tcs.GenerateDeeplink(*connreq, tonconnect.WithNoneReturnStrategy())
	if err != nil {
		return err
	}

	fmt.Printf("collection address: %v\n\ntonconnect deeplink:\n%v\n\ntonconnect qr code:\n", addr, deeplink)
	qrterminal.GenerateHalfBlock(deeplink, qrterminal.L, os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	wallets := make([]tonconnect.Wallet, 0)
	for _, wallet := range tonconnect.Wallets {
		wallets = append(wallets, wallet)
	}

	_, err = tcs.Connect(ctx, wallets...)
	if err != nil {
		return err
	}

	msg, err := tonconnect.NewMessage(addr, amount)
	if err != nil {
		return err
	}

	if stateInit != nil {
		msg.StateInit = stateInit.ToBOC()
	}
	if body != nil {
		msg.Payload = body.ToBOC()
	}

	tx, err := tonconnect.NewTransaction(tonconnect.WithMessage(*msg), tonconnect.WithTimeout(10*time.Minute))
	if err != nil {
		return err
	}

	_, err = tcs.SendTransaction(ctx, *tx)
	if err != nil {
		return err
	}

	err = tcs.Disconnect(ctx)
	if err != nil {
		return err
	}

	return nil
}

func genupd(cmd *cobra.Command, args []string) error {
	config.LoadConfig()

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

		sendMessage(addr.String(), "50000000", nil, stateInitCell)
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
				depth:  u.Node.CellDepth,
				update: true,
			}
		}

		for i, h := range upd.Hashes {
			m[i] = updateCellElement{
				node:   h.Hash[:],
				depth:  h.CellDepth,
				update: false,
			}
		}

		oldUpd, newUpd := buildUpdateCell(m, 1, config.Config.Depth)

		bodyCell := cell.BeginCell().
			MustStoreUInt(0x23cd52c, 32).
			MustStoreUInt(0, 64).
			MustStoreRef(cell.BeginCell().MustStoreRef(types.MakeMerkleProof(oldUpd)).MustStoreRef(types.MakeMerkleProof(newUpd)).EndCell()).
			EndCell()

		sendMessage(args[1], "150000000", bodyCell, nil)
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

func doMigrate() error {
	config.LoadConfig()

	d, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("migrations", d, strings.Replace(config.Config.Database, "postgres", "pgx5", 1))
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	err1, err2 := m.Close()
	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}

func migr(cmd *cobra.Command, args []string) error {
	return doMigrate()
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

	var migrateCmd = &cobra.Command{
		Use:  "migrate",
		RunE: migr,
	}

	rootCmd.AddCommand(genupdCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(migrateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
