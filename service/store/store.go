package store

import (
	"fmt"

	"c-z.dev/go-micro"
	log "c-z.dev/go-micro/logger"
	"c-z.dev/go-micro/store"
	pb "c-z.dev/go-micro/store/service/proto"
	mcli "c-z.dev/micro/client/cli"
	"c-z.dev/micro/internal/helper"
	"c-z.dev/micro/service/store/handler"
	"github.com/urfave/cli/v2"
)

var (
	// Name of the store service
	Name = "go.micro.store"
	// Address is the store address
	Address = ":8002"
)

// Run runs the micro server
func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Init(log.WithFields(map[string]interface{}{"service": "store"}))

	if len(ctx.String("server_name")) > 0 {
		Name = ctx.String("server_name")
	}
	if len(ctx.String("address")) > 0 {
		Address = ctx.String("address")
	}

	// Initialise service
	service := micro.NewService(
		micro.Name(Name),
	)

	// the store handler
	storeHandler := &handler.Store{
		Default: service.Options().Store,
		Stores:  make(map[string]bool),
	}

	table := "store"
	if v := ctx.String("store_table"); len(v) > 0 {
		table = v
	}

	// set to store table
	storeHandler.Default.Init(
		store.Table(table),
	)

	backend := storeHandler.Default.String()
	options := storeHandler.Default.Options()

	log.Infof("Initialising the [%s] store with opts: %+v", backend, options)

	// set the new store initialiser
	storeHandler.New = func(database string, table string) (store.Store, error) {
		// Record the new database and table in the internal store
		if err := storeHandler.Default.Write(&store.Record{
			Key:   "databases/" + database,
			Value: []byte{},
		}, store.WriteTo("micro", "internal")); err != nil {
			return nil, fmt.Errorf("micro store couldn't store new database in internal table: %w", err)
		}
		if err := storeHandler.Default.Write(&store.Record{
			Key:   "tables/" + database + "/" + table,
			Value: []byte{},
		}, store.WriteTo("micro", "internal")); err != nil {
			return nil, fmt.Errorf("micro store couldn't store new table in internal table: %w", err)
		}

		return storeHandler.Default, nil
	}

	pb.RegisterStoreHandler(service.Server(), storeHandler)

	// start the service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

// Commands is the cli interface for the store service
func Commands(options ...micro.Option) []*cli.Command {
	command := &cli.Command{
		Name:  "store",
		Usage: "Run the micro store service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Set the micro tunnel address :8002",
				EnvVars: []string{"MICRO_SERVER_ADDRESS"},
			},
		},
		Action: func(ctx *cli.Context) error {
			if err := helper.UnexpectedSubcommand(ctx); err != nil {
				return err
			}
			Run(ctx, options...)
			return nil
		},
		Subcommands: mcli.StoreCommands(),
	}

	return []*cli.Command{command}
}
