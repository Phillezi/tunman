package cli

import (
	"fmt"

	"github.com/Phillezi/tunman/config"
	"github.com/Phillezi/tunman/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:  "tunman",
	Long: tunman,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Setup()
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	cobra.OnInitialize(func() { config.InitConfig("tunman") })

	rootCmd.PersistentFlags().String("loglevel", "info", "Set the logging level (info, warn, error, debug)")
	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.PersistentFlags().String("profile", "", "Set the logging profile (production or empty)")
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	rootCmd.PersistentFlags().Bool("stacktrace", false, "Show the stack trace in error logs")
	viper.BindPFlag("stacktrace", rootCmd.PersistentFlags().Lookup("stacktrace"))

	rootCmd.AddCommand(versionCmd)
}

func ExecuteE() error {
	return rootCmd.Execute()
}

func GetRootCMD() *cobra.Command {
	return rootCmd
}
