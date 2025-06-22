package cli

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/Phillezi/tunman-remaster/config"
	"github.com/Phillezi/tunman-remaster/internal/defaults"
	"github.com/Phillezi/tunman-remaster/internal/lock"
	"github.com/Phillezi/tunman-remaster/interrupt"
	"github.com/Phillezi/tunman-remaster/log"
	"github.com/Phillezi/tunman-remaster/pkg/controller"
	"github.com/Phillezi/tunman-remaster/pkg/manager"
	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use: "tunmand",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Setup()
		interrupt.GetInstance().AddShutdownHook(func() { zap.L().Info("daemon shutdown") })
	},
	Long: tundmand,
	Run: func(cmd *cobra.Command, args []string) {
		lock.Acquire()
		defer lock.Release()

		startLog()
		zap.L().Debug("start")
		defer interrupt.GetInstance().Shutdown()

		if viper.GetBool("pprof") {
			go func() {
				// Start pprof server on localhost:6060
				zap.L().Info("starting pprof server on localhost:6060")
				if err := http.ListenAndServe("localhost:6060", nil); err != nil {
					zap.L().Error("pprof server error", zap.Error(err))
				}
			}()
		}

		man := manager.New()
		interrupt.GetInstance().AddShutdownHook(func() { zap.L().Info("manager shutdown"); man.Shutdown() })

		go func() {
			if err := controller.ListenAndServe(utils.Or(defaults.SocketPath), man, nil); err != nil {
				zap.L().Error("error serving", zap.Error(err))
			}
		}()

		<-interrupt.GetInstance().Context().Done()
		interrupt.GetInstance().Wait(5 * time.Second)
	},
}

func init() {
	cobra.OnInitialize(func() { config.InitConfig("tunmand") })

	rootCmd.PersistentFlags().String("loglevel", "info", "Set the logging level (info, warn, error, debug)")
	viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel"))

	rootCmd.PersistentFlags().String("profile", "", "Set the logging profile (production or empty)")
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))

	rootCmd.PersistentFlags().Bool("stacktrace", false, "Show the stack trace in error logs")
	viper.BindPFlag("stacktrace", rootCmd.PersistentFlags().Lookup("stacktrace"))

	rootCmd.PersistentFlags().Bool("insecure", false, "Dont validate against known_hosts")
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure"))

	rootCmd.PersistentFlags().Bool("pprof", false, "Enable pprof profiling HTTP server")
	viper.BindPFlag("pprof", rootCmd.PersistentFlags().Lookup("pprof"))

	rootCmd.PersistentFlags().String("dbpath", defaults.DefaultDBPath, "Set the path for the db")
	viper.BindPFlag("dbpath", rootCmd.PersistentFlags().Lookup("dbpath"))
}

func ExecuteE() error {
	return rootCmd.Execute()
}

func GetRootCMD() *cobra.Command {
	return rootCmd
}
