package cli

import (
	"fmt"
)

const (
	tunmand = `
  __                                          .___
_/  |_ __ __  ____   _____ _____    ____    __| _/
\   __\  |  \/    \ /     \\__  \  /    \  / __ | 
 |  | |  |  /   |  \  Y Y  \/ __ \|   |  \/ /_/ | 
 |__| |____/|___|  /__|_|  (____  /___|  /\____ | 
                 \/      \/     \/     \/      \/ `
)

var (
	version = "v0.0.1"
)

func startLog() {
	fmt.Printf("%s\nVersion:\t\t\t%s\n", tunmand, version)
}
