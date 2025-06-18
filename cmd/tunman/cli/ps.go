package cli

import (
	"fmt"

	"github.com/Phillezi/tunman-remaster/internal/connection"
	"github.com/Phillezi/tunman-remaster/interrupt"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var psCmd = &cobra.Command{
	Use: "ps",
	Run: func(cmd *cobra.Command, args []string) {
		if conn := connection.C(); conn != nil {
			resp, err := conn.Ps(interrupt.GetInstance().Context(), &ctrlpb.PsRequest{})
			if err != nil {
				zap.L().Error("failed to do ps command", zap.Error(err))
				return
			}
			if len(resp.Errors) > 0 {
				for _, err := range resp.Errors {
					zap.L().Error("error occurred when doing ps command", zap.Error(fmt.Errorf("%s", err)))
				}
				return
			}
			if len(resp.Fwds) <= 0 {
				fmt.Println("no active forwards")
				return
			}
			fmt.Println("ID\tHOST\t\tFWD")
			for _, fwd := range resp.Fwds {
				fmt.Printf("%s\t[%s]\t[%s]:[%s]\n", fwd.Id, fwd.Parent.Addr, fwd.Addrs.LocalAddr, fwd.Addrs.RemoteAddr)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(psCmd)
}
