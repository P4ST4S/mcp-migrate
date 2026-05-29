package main

import (
	"os"

	"github.com/P4ST4S/mcp-migrate/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
