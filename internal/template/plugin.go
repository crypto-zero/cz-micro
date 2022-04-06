package template

var (
	Plugin = `package main
{{if .Plugins}}
import ({{range .Plugins}}
	_ "github.com/crypto-zero/go-plugins/{{.}}"{{end}}
){{end}}
`
)
