package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ton-community/compressed-nft-api/config"
	myhttp "github.com/ton-community/compressed-nft-api/http"
	"github.com/ton-community/compressed-nft-api/migrations"
	"github.com/ton-community/compressed-nft-api/provider"
	"github.com/ton-community/compressed-nft-api/provider/file"
	"github.com/ton-community/compressed-nft-api/provider/pg"
	"github.com/ton-community/compressed-nft-api/state"
	"github.com/ton-community/compressed-nft-api/types"
	"github.com/ton-community/compressed-nft-api/updates"
	"github.com/xssnick/tonutils-go/address"
)

func doMigrate() error {
	d, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance("migrations", d, strings.Replace(config.Config.Database, "postgres", "pgx5", 1))
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
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

func main() {
	config.LoadConfig()

	if err := doMigrate(); err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(context.Background(), config.Config.Database)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	var sp provider.StateProvider = &file.StateProvider{
		Path: path.Join(config.Config.DataDir, "state.json"),
	}
	var ip provider.ItemProvider = pg.NewItemProvider(pool)
	var np provider.NodeProvider = pg.NewNodeProvider(pool)

	currentState, err := sp.GetState()
	if err != nil {
		panic(err)
	}

	stateHolder := state.NewStateHolder(currentState)

	addrs := make(chan *address.Address, 16)
	newStates := make(chan *types.State, 16)

	if currentState.Address != nil {
		addrs <- currentState.Address.Address
	}

	go updates.Watcher(newStates, addrs, stateHolder, sp)

	var up updates.Recorder = &updates.FileUpdateRecorder{
		Base: path.Join(config.Config.DataDir, "upd"),
	}

	handler := &myhttp.Handler{
		StateProvider: sp,
		ItemProvider:  ip,
		NodeProvider:  np,

		StateHolder: stateHolder,

		Depth: config.Config.Depth,

		NewStates: newStates,
		Addresses: addrs,

		UpdateRecorder: up,
	}

	handler.RegisterHandlers(e)

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", config.Config.Port)))
}
