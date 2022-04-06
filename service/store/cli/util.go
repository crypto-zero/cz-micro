package cli

import (
	"fmt"
	"strings"

	"c-z.dev/go-micro/config/cmd"
	"c-z.dev/go-micro/store"
	"github.com/urfave/cli/v2"
)

// makeStore is a helper function that creates a store for snapshot and restore
func makeStore(ctx *cli.Context) (store.Store, error) {
	builtinStore, err := getStore(ctx.String("store"))
	if err != nil {
		return nil, fmt.Errorf("makeStore: %w", err)
	}
	s := builtinStore(
		store.Nodes(strings.Split(ctx.String("nodes"), ",")...),
		store.Database(ctx.String("database")),
		store.Table(ctx.String("table")),
	)
	if err := s.Init(); err != nil {
		return nil, fmt.Errorf("couldn't init %s store: %w", ctx.String("store"), err)
	}
	return s, nil
}

// makeStores is a helper function that sets up 2 stores for sync
func makeStores(ctx *cli.Context) (store.Store, store.Store, error) {
	fromBuilder, err := getStore(ctx.String("from-backend"))
	if err != nil {
		return nil, nil, fmt.Errorf("from store: %w", err)
	}
	toBuilder, err := getStore(ctx.String("to-backend"))
	if err != nil {
		return nil, nil, fmt.Errorf("to store: %w", err)
	}
	from := fromBuilder(
		store.Nodes(strings.Split(ctx.String("from-nodes"), ",")...),
		store.Database(ctx.String("from-database")),
		store.Table(ctx.String("from-table")),
	)
	if err := from.Init(); err != nil {
		return nil, nil, fmt.Errorf("from: couldn't init %s: %w", ctx.String("from-backend"), err)
	}
	to := toBuilder(
		store.Nodes(strings.Split(ctx.String("to-nodes"), ",")...),
		store.Database(ctx.String("to-database")),
		store.Table(ctx.String("to-table")),
	)
	if err := to.Init(); err != nil {
		return nil, nil, fmt.Errorf("to: couldn't init %s: %w", ctx.String("to-backend"), err)
	}
	return from, to, nil
}

func getStore(s string) (func(...store.Option) store.Store, error) {
	builtinStore, exists := cmd.DefaultStores[s]
	if !exists {
		return nil, fmt.Errorf("store %s is not an implemented store - check your plugins", s)
	}
	return builtinStore, nil
}
