package cli

import (
	"errors"
	"fmt"
	"net/url"

	"c-z.dev/go-micro/logger"
	"c-z.dev/micro/service/store/snapshot"
	"github.com/urfave/cli/v2"
)

// Restore is the entrypoint for micro store restore
func Restore(ctx *cli.Context) error {
	s, err := makeStore(ctx)
	if err != nil {
		return fmt.Errorf("couldn't construct a store: %w", err)
	}
	log := logger.DefaultLogger
	var rs snapshot.Restore
	source := ctx.String("source")

	if len(source) == 0 {
		return errors.New("source flag must be set")
	}
	u, err := url.Parse(source)
	if err != nil {
		return fmt.Errorf("source is invalid: %w", err)
	}
	switch u.Scheme {
	case "file":
		rs = snapshot.NewFileRestore(snapshot.Source(source))
	default:
		return fmt.Errorf("unsupported source scheme: %s", u.Scheme)
	}

	err = rs.Init()
	if err != nil {
		return fmt.Errorf("failed to initialise the restorer: %w", err)
	}

	recordChan, err := rs.Start()
	if err != nil {
		return fmt.Errorf("couldn't start the restorer: %w", err)
	}
	counter := uint64(0)
	for r := range recordChan {
		err := s.Write(r)
		if err != nil {
			log.Logf(logger.ErrorLevel, "couldn't write key %s to store %s", r.Key, s.String())
		} else {
			counter++
		}
	}
	log.Logf(logger.DebugLevel, "Restored %d records", counter)
	return nil
}
