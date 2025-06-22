package cli

import (
	"fmt"

	"github.com/Phillezi/tunman/internal/connection"
	"github.com/Phillezi/tunman/interrupt"
	ctrlpb "github.com/Phillezi/tunman/proto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var closeCmd = &cobra.Command{
	Use:   "close [ids...]",
	Short: "Close a tunnel or multiple tunnels by ID or all",
	Long: `The close command is used to terminate active tunnels previously opened by the daemon.
You can close a specific tunnel or multiple tunnels by providing their IDs as arguments. These IDs are printed to stdout when a tunnel is opened.

If you want to close **all** tunnels at once, you can either use the --all flag or pass "all" as the only argument.

Note: Closing all tunnels using the "all" keyword or the --all flag will terminate every active tunnel managed by the daemon.`,
	Example: `tunman close MTdlOTk3NTE4YzVhZTRjYw.YmJlZTA1MzNiOTMwMzEwNQ
# The command above will close the tunnel with the given ID

tunman close MTdlOTk3NTE4YzVhZTRjYw.YmJlZTA1MzNiOTMwMzEwNQ ODE4ZmY5ODlmYWE5ZTIwOA.YzdjMWMyZDc2ODhiOWJkZA
# The command above will close multiple tunnels by their IDs

tunman close all
# This will close all tunnels`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if conn := connection.C(); conn != nil {
			if !viper.GetBool("all") && args[0] != "all" {
				resp, err := conn.CloseFwd(interrupt.GetInstance().Context(), &ctrlpb.CloseRequest{
					Ids: args,
				})
				if err != nil {
					zap.L().Error("failed to execute close command", zap.Error(err))
					return
				}
				if len(resp.Errors) > 0 {
					for _, err := range resp.Errors {
						zap.L().Error("error occurred when closing tunnel", zap.Error(fmt.Errorf("%s", err)))
					}
					return
				}
				for _, id := range resp.ClosedIds {
					fmt.Println(id)
				}
			} else {
				resp, err := conn.CloseAllFwds(interrupt.GetInstance().Context(), &ctrlpb.CloseAllRequest{})
				if err != nil {
					zap.L().Error("failed to execute close all command", zap.Error(err))
					return
				}
				if resp.Error != "" {
					zap.L().Error("error occurred when closing all tunnels", zap.Error(fmt.Errorf("%s", resp.Error)))
					return
				} else if resp.Ok {
					fmt.Println("all tunnels were closed")
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)

	closeCmd.Flags().BoolP("all", "a", false, "Close all tunnels")
	viper.BindPFlag("all", closeCmd.Flags().Lookup("all"))
}
