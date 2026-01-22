package main

import (
	"os"

	"github.com/proxikal/hydra/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
