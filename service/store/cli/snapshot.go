package cli

import (
	"errors"
	"fmt"
	"net/url"

	"c-z.dev/go-micro/logger"
	"c-z.dev/micro/service/store/snapshot"
	"github.com/urfave/cli/v2"
)

// Snapshot in the entrypoint for micro store snapshot
func Snapshot(ctx *cli.Context) error {
	s, err := makeStore(ctx)
	if err != nil {
		return fmt.Errorf("couldn't construct a store: %w", err)
	}
	log := logger.DefaultLogger
	dest := ctx.String("destination")
	var sn snapshot.Snapshot

	if len(dest) == 0 {
		return errors.New("destination flag must be set")
	}
	u, err := url.Parse(dest)
	if err != nil {
		return fmt.Errorf("destination is invalid: %w", err)
	}
	switch u.Scheme {
	case "file":
		sn = snapshot.NewFileSnapshot(snapshot.Destination(dest))
	default:
		return fmt.Errorf("unsupported destination scheme: %s", u.Scheme)
	}
	err = sn.Init()
	if err != nil {
		return fmt.Errorf("failed to initialise the snapshotter: %w", err)
	}

	log.Logf(logger.InfoLevel, "Snapshotting store %s", s.String())
	recordChan, err := sn.Start()
	if err != nil {
		return fmt.Errorf("couldn't start the snapshotter: %w", err)
	}
	keys, err := s.List()
	if err != nil {
		return fmt.Errorf("couldn't List() from store %s: %w", s.String(), err)
	}
	log.Logf(logger.DebugLevel, "Snapshotting %d keys", len(keys))

	for _, key := range keys {
		r, err := s.Read(key)
		if err != nil {
			return fmt.Errorf("couldn't read key %s: %w", key, err)
		}
		if len(r) != 1 {
			return fmt.Errorf("reading %s from %s returned 0 records", key, s.String())
		}
		recordChan <- r[0]
	}
	close(recordChan)
	sn.Wait()
	return nil
}
