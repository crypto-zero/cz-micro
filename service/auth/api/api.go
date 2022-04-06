package api

import (
	"c-z.dev/go-micro"
	log "c-z.dev/go-micro/logger"
	pb "c-z.dev/micro/service/auth/api/proto"

	"github.com/urfave/cli/v2"
)

var (
	// Name of the auth api
	Name = "go.micro.api.auth"
	// Address is the api address
	Address = ":8011"
)

// Run the micro auth api
func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Init(log.WithFields(map[string]interface{}{"service": "auth"}))

	service := micro.NewService(
		micro.Name(Name),
		micro.Address(Address),
	)

	pb.RegisterAuthHandler(service.Server(), NewHandler(service))

	if err := service.Run(); err != nil {
		log.Error(err)
	}
}
