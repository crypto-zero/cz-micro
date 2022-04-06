package debug

import (
	"fmt"
	"time"

	"c-z.dev/go-micro"
	"c-z.dev/go-micro/debug/service"
	log "c-z.dev/go-micro/logger"
	"github.com/urfave/cli/v2"
)

const (
	// logUsage message for logs command
	traceUsage = "Required usage: micro trace example"
)

func getTrace(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Trace("debug")

	// TODO look for trace id

	if ctx.Args().Len() == 0 {
		fmt.Println("Require service name")
		return
	}

	name := ctx.Args().Get(0)

	// must specify service name
	if len(name) == 0 {
		fmt.Println(traceUsage)
		return
	}

	// initialise a new service log
	// TODO: allow "--source" e.g. kubernetes
	srv := service.NewClient(name)

	spans, err := srv.Trace()
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(spans) == 0 {
		return
	}

	fmt.Println("Id\tName\tTime\tDuration\tStatus")

	for _, span := range spans {
		fmt.Printf("%s\t%s\t%s\t%v\t%s\n",
			span.Trace,
			span.Name,
			time.Unix(0, int64(span.Started)).String(),
			time.Duration(span.Duration),
			"",
		)
	}
}
