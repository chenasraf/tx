package main

import (
	_ "embed"
	"strings"

	"github.com/chenasraf/tx/internal/cli"
)

//go:embed version.txt
var version string

func main() {
	cli.Version = strings.TrimSpace(version)
	cli.Execute()
}
