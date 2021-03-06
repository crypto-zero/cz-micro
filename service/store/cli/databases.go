package cli

import (
	"context"
	"errors"
	"fmt"

	"c-z.dev/go-micro/config/cmd"
	storeproto "c-z.dev/go-micro/store/service/proto"
	"github.com/urfave/cli/v2"
)

// Databases is the entrypoint for micro store databases
func Databases(ctx *cli.Context) error {
	client := *cmd.DefaultOptions().Client
	dbReq := client.NewRequest(ctx.String("store"), "Store.Databases", &storeproto.DatabasesRequest{})
	dbRsp := &storeproto.DatabasesResponse{}
	if err := client.Call(context.TODO(), dbReq, dbRsp); err != nil {
		return err
	}
	for _, db := range dbRsp.Databases {
		fmt.Println(db)
	}
	return nil
}

// Tables is the entrypoint for micro store tables
func Tables(ctx *cli.Context) error {
	if len(ctx.String("database")) == 0 {
		return errors.New("database flag is required")
	}
	client := *cmd.DefaultOptions().Client
	tReq := client.NewRequest(ctx.String("store"), "Store.Tables", &storeproto.TablesRequest{
		Database: ctx.String("database"),
	})
	tRsp := &storeproto.TablesResponse{}
	if err := client.Call(context.TODO(), tReq, tRsp); err != nil {
		return err
	}
	for _, table := range tRsp.Tables {
		fmt.Println(table)
	}
	return nil
}
