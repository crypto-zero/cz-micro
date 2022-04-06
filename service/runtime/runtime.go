// Package runtime is the micro runtime
package runtime

import (
	"os"

	"c-z.dev/go-micro"
	"c-z.dev/go-micro/config/cmd"
	log "c-z.dev/go-micro/logger"
	"c-z.dev/go-micro/runtime"
	pb "c-z.dev/go-micro/runtime/service/proto"
	"c-z.dev/micro/service/runtime/handler"
	"c-z.dev/micro/service/runtime/manager"
	"c-z.dev/micro/service/runtime/profile"
	"github.com/urfave/cli/v2"
)

var (
	// Name of the runtime
	Name = "go.micro.runtime"
	// Address of the runtime
	Address = ":8088"
)

// Run the runtime service
func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Init(log.WithFields(map[string]interface{}{"service": "runtime"}))

	// Get the profile
	var prof []string
	switch ctx.String("profile") {
	case "local":
		prof = profile.Local()
	case "server":
		prof = profile.Server()
	case "kubernetes":
		prof = profile.Kubernetes()
	case "platform":
		prof = profile.Platform()
	}

	if len(ctx.String("address")) > 0 {
		Address = ctx.String("address")
	}

	if len(ctx.String("server_name")) > 0 {
		Name = ctx.String("server_name")
	}

	if len(Address) > 0 {
		srvOpts = append(srvOpts, micro.Address(Address))
	}

	// create runtime
	muRuntime := *cmd.DefaultCmd.Options().Runtime
	if ctx.IsSet("source") {
		muRuntime.Init(runtime.WithSource(ctx.String("source")))
	}

	// append name
	srvOpts = append(srvOpts, micro.Name(Name))

	// new service
	service := micro.NewService(srvOpts...)

	// create a new runtime manager
	manager := manager.New(muRuntime,
		manager.Store(service.Options().Store),
		manager.Profile(prof),
	)

	// start the manager
	if err := manager.Start(); err != nil {
		log.Errorf("failed to start: %s", err)
		os.Exit(1)
	}

	// register the runtime handler
	pb.RegisterRuntimeHandler(service.Server(), &handler.Runtime{
		// Client to publish events
		Client: micro.NewEvent("go.micro.runtime.events", service.Client()),
		// using the micro runtime
		Runtime: manager,
	})

	// start runtime service
	if err := service.Run(); err != nil {
		log.Errorf("error running service: %v", err)
	}

	// stop the manager
	if err := manager.Stop(); err != nil {
		log.Errorf("failed to stop: %s", err)
		os.Exit(1)
	}
}

// Flags is shared flags so we don't have to continually re-add
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "source",
			Usage: "Set the source url of the service",
		},
		&cli.StringFlag{
			Name:  "image",
			Usage: "Set the image to use for the container",
		},
		&cli.StringFlag{
			Name:  "command",
			Usage: "Command to exec",
		},
		&cli.StringFlag{
			Name:  "args",
			Usage: "Command args",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "The type of service operate on",
		},
		&cli.StringSliceFlag{
			Name:  "env_vars",
			Usage: "Set the environment variables e.g. foo=bar",
		},
	}
}

func Commands(options ...micro.Option) []*cli.Command {
	command := []*cli.Command{
		{
			Name:  "runtime",
			Usage: "Run the micro runtime",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "address",
					Usage:   "Set the registry http address e.g 0.0.0.0:8088",
					EnvVars: []string{"MICRO_SERVER_ADDRESS"},
				},
				&cli.StringFlag{
					Name:    "profile",
					Usage:   "Set the runtime profile to use for services e.g local, kubernetes, platform",
					EnvVars: []string{"MICRO_RUNTIME_PROFILE"},
				},
				&cli.StringFlag{
					Name:    "source",
					Usage:   "Set the runtime source, e.g. micro/services",
					EnvVars: []string{"MICRO_RUNTIME_SOURCE"},
				},
				&cli.IntFlag{
					Name:    "retries",
					Usage:   "Set the max retries per service",
					EnvVars: []string{"MICRO_RUNTIME_RETRIES"},
				},
			},
			Action: func(ctx *cli.Context) error {
				Run(ctx, options...)
				return nil
			},
		},
	}
	return command
}
