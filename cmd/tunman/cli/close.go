package cli

import (
	"fmt"

	"github.com/Phillezi/tunman-remaster/internal/connection"
	"github.com/Phillezi/tunman-remaster/interrupt"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var closeCmd = &cobra.Command{
	Use:   "close [ids...]",
	Short: "Close a tunnels",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if conn := connection.C(); conn != nil {
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
		}
	},
}

func init() {
	rootCmd.AddCommand(closeCmd)
}
