// Package plugins includes the plugins we want to load
package plugins

import (
	"c-z.dev/go-micro/config/cmd"

	// import specific plugins
	memStore "c-z.dev/go-micro/store/memory"
)

func init() {
	// TODO: make it so we only have to import them
	cmd.DefaultStores["memory"] = memStore.NewStore
}
