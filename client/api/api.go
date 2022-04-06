// Package api is an API Gateway
package api

import (
	"c-z.dev/go-micro"
	"fmt"
	"net/http"
	"strings"

	ahandler "c-z.dev/go-micro/api/handler"
	aapi "c-z.dev/go-micro/api/handler/api"
	"c-z.dev/go-micro/api/handler/event"
	ahttp "c-z.dev/go-micro/api/handler/http"
	arpc "c-z.dev/go-micro/api/handler/rpc"
	"c-z.dev/go-micro/api/handler/web"
	"c-z.dev/go-micro/api/resolver"
	"c-z.dev/go-micro/api/resolver/grpc"
	"c-z.dev/go-micro/api/resolver/host"
	"c-z.dev/go-micro/api/resolver/path"
	"c-z.dev/go-micro/api/router"
	regRouter "c-z.dev/go-micro/api/router/registry"
	"c-z.dev/go-micro/api/server"
	httpapi "c-z.dev/go-micro/api/server/http"
	log "c-z.dev/go-micro/logger"
	"c-z.dev/micro/client/api/auth"
	"c-z.dev/micro/internal/handler"
	"c-z.dev/micro/internal/helper"
	"c-z.dev/micro/internal/namespace"
	rrmicro "c-z.dev/micro/internal/resolver/api"
	"c-z.dev/micro/internal/stats"
	"github.com/gorilla/mux"
	"github.com/urfave/cli/v2"
)

var (
	Name      = "go.micro.api"
	Address   = ":8080"
	Handler   = "meta"
	Resolver  = "micro"
	RPCPath   = "/rpc"
	APIPath   = "/"
	ProxyPath = "/{service:[a-zA-Z0-9]+}"
	Namespace = "go.micro"
	Type      = "api"
	EnableRPC = false
)

func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Init(log.WithFields(map[string]interface{}{"service": "api"}))

	if len(ctx.String("server_name")) > 0 {
		Name = ctx.String("server_name")
	}
	if len(ctx.String("address")) > 0 {
		Address = ctx.String("address")
	}
	if len(ctx.String("handler")) > 0 {
		Handler = ctx.String("handler")
	}
	if len(ctx.String("resolver")) > 0 {
		Resolver = ctx.String("resolver")
	}
	if len(ctx.String("enable_rpc")) > 0 {
		EnableRPC = ctx.Bool("enable_rpc")
	}
	if len(ctx.String("type")) > 0 {
		Type = ctx.String("type")
	}
	if len(ctx.String("namespace")) > 0 {
		// remove the service type from the namespace to allow for
		// backwards compatability
		Namespace = strings.TrimSuffix(ctx.String("namespace"), "."+Type)
	}

	// apiNamespace has the format: "go.micro.api"
	apiNamespace := Namespace + "." + Type

	// append name to opts
	srvOpts = append(srvOpts, micro.Name(Name))

	// initialise service
	service := micro.NewService(srvOpts...)

	// Init API
	var opts []server.Option

	if ctx.Bool("enable_tls") {
		config, err := helper.TLSConfig(ctx)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		opts = append(opts, server.EnableTLS(true))
		opts = append(opts, server.TLSConfig(config))
	}

	if ctx.Bool("enable_cors") {
		opts = append(opts, server.EnableCORS(true))
	}

	// create the router
	var h http.Handler
	r := mux.NewRouter()
	h = r

	if ctx.Bool("enable_stats") {
		st := stats.New()
		r.HandleFunc("/stats", st.StatsHandler)
		h = st.ServeHTTP(r)
		st.Start()
		defer st.Stop()
	}

	// return version and list of services
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			return
		}

		response := fmt.Sprintf(`{"version": "%s"}`, ctx.App.Version)
		w.Write([]byte(response))
	})

	// strip favicon.ico
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})

	// register rpc handler
	if EnableRPC {
		log.Infof("Registering RPC Handler at %s", RPCPath)
		r.HandleFunc(RPCPath, handler.RPC)
	}

	// create the namespace resolver
	nsResolver := namespace.NewResolver(Type, Namespace)

	// resolver options
	ropts := []resolver.Option{
		resolver.WithNamespace(nsResolver.ResolveWithType),
		resolver.WithHandler(Handler),
	}

	// default resolver
	rr := rrmicro.NewResolver(ropts...)

	switch Resolver {
	case "host":
		rr = host.NewResolver(ropts...)
	case "path":
		rr = path.NewResolver(ropts...)
	case "grpc":
		rr = grpc.NewResolver(ropts...)
	}

	switch Handler {
	case "rpc":
		log.Infof("Registering API RPC Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(arpc.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		rp := arpc.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(rp)
	case "api":
		log.Infof("Registering API Request Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(aapi.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ap := aapi.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(ap)
	case "event":
		log.Infof("Registering API Event Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(event.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ev := event.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(ev)
	case "http", "proxy":
		log.Infof("Registering API HTTP Handler at %s", ProxyPath)
		rt := regRouter.NewRouter(
			router.WithHandler(ahttp.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		ht := ahttp.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(ProxyPath).Handler(ht)
	case "web":
		log.Infof("Registering API Web Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithHandler(web.Handler),
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		w := web.NewHandler(
			ahandler.WithNamespace(apiNamespace),
			ahandler.WithRouter(rt),
			ahandler.WithClient(service.Client()),
		)
		r.PathPrefix(APIPath).Handler(w)
	default:
		log.Infof("Registering API Default Handler at %s", APIPath)
		rt := regRouter.NewRouter(
			router.WithResolver(rr),
			router.WithRegistry(service.Options().Registry),
		)
		r.PathPrefix(APIPath).Handler(handler.Meta(service, rt, nsResolver.ResolveWithType))
	}

	// create the auth wrapper and the server
	authWrapper := auth.Wrapper(rr, nsResolver)
	api := httpapi.NewServer(Address, server.WrapHandler(authWrapper))

	api.Init(opts...)
	api.Handle("/", h)

	// Start API
	if err := api.Start(); err != nil {
		log.Fatal(err)
	}

	// Run server
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

	// Stop API
	if err := api.Stop(); err != nil {
		log.Fatal(err)
	}
}

func Commands(options ...micro.Option) []*cli.Command {
	command := &cli.Command{
		Name:  "api",
		Usage: "Run the api gateway",
		Action: func(ctx *cli.Context) error {
			Run(ctx, options...)
			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "Set the api address e.g 0.0.0.0:8080",
				EnvVars: []string{"MICRO_API_ADDRESS"},
			},
			&cli.StringFlag{
				Name:    "handler",
				Usage:   "Specify the request handler to be used for mapping HTTP requests to services; {api, event, http, rpc}",
				EnvVars: []string{"MICRO_API_HANDLER"},
			},
			&cli.StringFlag{
				Name:    "namespace",
				Usage:   "Set the namespace used by the API e.g. com.example",
				EnvVars: []string{"MICRO_API_NAMESPACE"},
			},
			&cli.StringFlag{
				Name:    "type",
				Usage:   "Set the service type used by the API e.g. api",
				EnvVars: []string{"MICRO_API_TYPE"},
			},
			&cli.StringFlag{
				Name:    "resolver",
				Usage:   "Set the hostname resolver used by the API {host, path, grpc}",
				EnvVars: []string{"MICRO_API_RESOLVER"},
			},
			&cli.BoolFlag{
				Name:    "enable_rpc",
				Usage:   "Enable call the backend directly via /rpc",
				EnvVars: []string{"MICRO_API_ENABLE_RPC"},
			},
			&cli.BoolFlag{
				Name:    "enable_cors",
				Usage:   "Enable CORS, allowing the API to be called by frontend applications",
				EnvVars: []string{"MICRO_API_ENABLE_CORS"},
				Value:   true,
			},
		},
	}

	return []*cli.Command{command}
}
