package main

import (
	"os"

	"github.com/Phillezi/tunman/cmd/tunmand/cli"
)

func main() {
	if err := cli.ExecuteE(); err != nil {
		os.Exit(1)
	}
}
