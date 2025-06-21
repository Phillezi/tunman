package cli

import (
	"fmt"
	"runtime"
	"time"
)

const (
	tundmand = `
  __                                      
_/  |_ __ __  ____   _____ _____    ____  
\   __\  |  \/    \ /     \\__  \  /    \ 
 |  | |  |  /   |  \  Y Y  \/ __ \|   |  \
 |__| |____/|___|  /__|_|  (____  /___|  /
                 \/      \/     \/     \/ `
	version = "v0.0.1"
)

func startLog() {
	fmt.Printf("%s\nVersion:\t\t\t%s\n", tundmand, version)
	fmt.Println("======================================")
	fmt.Printf("Start Time: %s\n", time.Now().Format(time.RFC1123))
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("CPUs:       %d\n", runtime.NumCPU())
	fmt.Println("======================================")
}
