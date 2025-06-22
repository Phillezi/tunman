package main

import (
	"os"

	"github.com/Phillezi/tunman/cmd/tunman/cli"
)

func main() {
	if err := cli.ExecuteE(); err != nil {
		os.Exit(1)
	}
}
