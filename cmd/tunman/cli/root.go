package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "tunman",
	Long: `
  __                                      
_/  |_ __ __  ____   _____ _____    ____  
\   __\  |  \/    \ /     \\__  \  /    \ 
 |  | |  |  /   |  \  Y Y  \/ __ \|   |  \
 |__| |____/|___|  /__|_|  (____  /___|  /
                 \/      \/     \/     \/ `,
}

func init() {
	cobra.OnInitialize()
}

func ExecuteE() error {
	return rootCmd.Execute()
}
