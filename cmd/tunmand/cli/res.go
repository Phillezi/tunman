package cli

import (
	"fmt"
)

const (
	tundmand = `
  __                                      
_/  |_ __ __  ____   _____ _____    ____  
\   __\  |  \/    \ /     \\__  \  /    \ 
 |  | |  |  /   |  \  Y Y  \/ __ \|   |  \
 |__| |____/|___|  /__|_|  (____  /___|  /
                 \/      \/     \/     \/ `
)

var (
	version = "v0.0.1"
)

func startLog() {
	fmt.Printf("%s\nVersion:\t\t\t%s\n", tundmand, version)
}
