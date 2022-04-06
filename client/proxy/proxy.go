// Package proxy is a cli proxy
package proxy

import (
	"os"
	"strings"

	"c-z.dev/go-micro"
	"c-z.dev/go-micro/auth"
	bmem "c-z.dev/go-micro/broker/memory"
	"c-z.dev/go-micro/client"
	mucli "c-z.dev/go-micro/client"
	"c-z.dev/go-micro/config/cmd"
	log "c-z.dev/go-micro/logger"
	"c-z.dev/go-micro/proxy"
	"c-z.dev/go-micro/proxy/http"
	"c-z.dev/go-micro/proxy/mucp"
	"c-z.dev/go-micro/registry"
	rmem "c-z.dev/go-micro/registry/memory"
	"c-z.dev/go-micro/router"
	rs "c-z.dev/go-micro/router/service"
	"c-z.dev/go-micro/server"
	sgrpc "c-z.dev/go-micro/server/grpc"
	"c-z.dev/go-micro/util/mux"
	"c-z.dev/go-micro/util/wrapper"
	"c-z.dev/micro/internal/helper"

	"github.com/urfave/cli/v2"
)

var (
	// Name of the proxy
	Name = "go.micro.proxy"
	// The address of the proxy
	Address = ":8081"
	// the proxy protocol
	Protocol = "grpc"
	// The endpoint host to route to
	Endpoint string
)

func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Init(log.WithFields(map[string]interface{}{"service": "proxy"}))

	// because MICRO_PROXY_ADDRESS is used internally by the go-micro/client
	// we need to unset it so we don't end up calling ourselves infinitely
	os.Unsetenv("MICRO_PROXY_ADDRESS")

	if len(ctx.String("server_name")) > 0 {
		Name = ctx.String("server_name")
	}
	if len(ctx.String("address")) > 0 {
		Address = ctx.String("address")
	}
	if len(ctx.String("endpoint")) > 0 {
		Endpoint = ctx.String("endpoint")
	}
	if len(ctx.String("protocol")) > 0 {
		Protocol = ctx.String("protocol")
	}

	// service opts
	srvOpts = append(srvOpts, micro.Name(Name))

	// new service
	service := micro.NewService(srvOpts...)

	// set the context
	var popts []proxy.Option

	// create new router
	var r router.Router

	routerName := ctx.String("router")
	routerAddr := ctx.String("router_address")

	ropts := []router.Option{
		router.Id(server.DefaultID),
		router.Client(client.DefaultClient),
		router.Address(routerAddr),
		router.Registry(registry.DefaultRegistry),
	}

	// check if we need to use the router service
	switch {
	case routerName == "go.micro.router":
		r = rs.NewRouter(ropts...)
	case routerName == "service":
		r = rs.NewRouter(ropts...)
	case len(routerAddr) > 0:
		r = rs.NewRouter(ropts...)
	default:
		r = router.NewRouter(ropts...)
	}

	// start the router
	if err := r.Start(); err != nil {
		log.Errorf("Proxy error starting router: %s", err)
		os.Exit(1)
	}

	// append router to proxy opts
	popts = append(popts, proxy.WithRouter(r))

	// new proxy
	var p proxy.Proxy
	// setup the default server
	var srv server.Server

	// set endpoint
	if len(Endpoint) > 0 {
		switch {
		case strings.HasPrefix(Endpoint, "grpc://"):
			ep := strings.TrimPrefix(Endpoint, "grpc://")
			popts = append(popts, proxy.WithEndpoint(ep))
			Protocol = "grpc"
		case strings.HasPrefix(Endpoint, "http://"):
			// TODO: strip prefix?
			popts = append(popts, proxy.WithEndpoint(Endpoint))
			Protocol = "http"
		default:
			// TODO: strip prefix?
			popts = append(popts, proxy.WithEndpoint(Endpoint))
			Protocol = "mucp"
		}
	}

	serverOpts := []server.Option{
		server.Address(Address),
		server.Registry(rmem.NewRegistry()),
		server.Broker(bmem.NewBroker()),
	}

	if ctx.Bool("enable_tls") {
		// get certificates from the context
		config, err := helper.TLSConfig(ctx)
		if err != nil {
			log.Fatal(err)
			return
		}
		serverOpts = append(serverOpts, server.TLSConfig(config))
	}

	// add auth wrapper to server
	var authOpts []auth.Option

	a := *cmd.DefaultOptions().Auth
	a.Init(authOpts...)
	authFn := func() auth.Auth { return a }
	authOpt := server.WrapHandler(wrapper.AuthHandler(authFn))
	serverOpts = append(serverOpts, authOpt)

	// set proxy
	switch Protocol {
	case "http":
		p = http.NewProxy(popts...)
		serverOpts = append(serverOpts, server.WithRouter(p))
		// TODO: http server
		srv = server.NewServer(serverOpts...)
	case "mucp":
		popts = append(popts, proxy.WithClient(mucli.NewClient()))
		p = mucp.NewProxy(popts...)

		serverOpts = append(serverOpts, server.WithRouter(p))
		srv = server.NewServer(serverOpts...)
	default:
		p = mucp.NewProxy(popts...)

		serverOpts = append(serverOpts, server.WithRouter(p))
		srv = sgrpc.NewServer(serverOpts...)
	}

	if len(Endpoint) > 0 {
		log.Infof("Proxy [%s] serving endpoint: %s", p.String(), Endpoint)
	} else {
		log.Infof("Proxy [%s] serving protocol: %s", p.String(), Protocol)
	}

	// create a new proxy muxer which includes the debug handler
	muxer := mux.New(Name, p)

	// set the router
	service.Server().Init(
		server.WithRouter(muxer),
	)

	// Start the proxy server
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}

	// Run internal service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

	// Stop the server
	if err := srv.Stop(); err != nil {
		log.Fatal(err)
	}
}

func Commands(options ...micro.Option) []*cli.Command {
	command := &cli.Command{
		Name:  "proxy",
		Usage: "Run the service proxy",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "router",
				Usage:   "Set the router to use e.g default, go.micro.router",
				EnvVars: []string{"MICRO_ROUTER"},
			},
			&cli.StringFlag{
				Name:    "router_address",
				Usage:   "Set the router address",
				EnvVars: []string{"MICRO_ROUTER_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Set the proxy http address e.g 0.0.0.0:8081",
				EnvVars: []string{"MICRO_PROXY_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "protocol",
				Usage:   "Set the protocol used for proxying e.g mucp, grpc, http",
				EnvVars: []string{"MICRO_PROXY_PROTOCOL"},
			},
			&cli.StringFlag{
				Name:    "endpoint",
				Usage:   "Set the endpoint to route to e.g greeter or localhost:9090",
				EnvVars: []string{"MICRO_PROXY_ENDPOINT"},
			},
		},
		Action: func(ctx *cli.Context) error {
			Run(ctx, options...)
			return nil
		},
	}

	return []*cli.Command{command}
}
