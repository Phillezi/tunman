package main

import (
	"os"

	"github.com/Phillezi/tunman-remaster/cmd/tunman/cli"
)

func main() {
	if err := cli.ExecuteE(); err != nil {
		os.Exit(1)
	}
}
