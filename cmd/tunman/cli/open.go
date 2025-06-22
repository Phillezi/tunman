package cli

import (
	"fmt"

	"github.com/Phillezi/tunman-remaster/internal/connection"
	"github.com/Phillezi/tunman-remaster/internal/parser"
	"github.com/Phillezi/tunman-remaster/interrupt"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var openCmd = &cobra.Command{
	Use:   "open [host]",
	Short: "Open a tunnel to a remote host",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u, h, p := parser.ParseTargetLoose(args[0])
		host := h
		port := utils.Or(viper.GetString("port"), p)
		userVal := utils.Or(viper.GetString("user"), u)
		pw := viper.GetString("password")

		localRemoteMap, err := parser.ParsePublishes(viper.GetStringSlice("publish"))
		if err != nil {
			return fmt.Errorf("failed to parse, err: %s", err.Error())
		}
		if len(localRemoteMap) == 0 {
			return fmt.Errorf("no forwards provided")
		}

		addrPairs := make(map[string]*ctrlpb.AddrPair, len(localRemoteMap))
		for l, r := range localRemoteMap {
			addrPairs[l] = &ctrlpb.AddrPair{
				LocalAddr:  l,
				RemoteAddr: r,
			}
		}

		if conn := connection.C(); conn != nil {
			resp, err := conn.OpenFwd(interrupt.GetInstance().Context(), &ctrlpb.OpenRequest{Tunnels: []*ctrlpb.Tunnel{{
				User:        utils.Or(userVal),
				Host:        host,
				Port:        utils.ParsePort(utils.Or(port)),
				Pw:          pw,
				AddressPair: addrPairs,
			}}})
			if err != nil {
				fmt.Println(err.Error())
				// not a input error, it is a connection error
				return nil
			}
			if len(resp.Errors) > 0 {
				for _, err := range resp.Errors {
					zap.L().Error("error occurred when opening tunnel", zap.Error(fmt.Errorf("%s", err)))
				}
			}
			for _, id := range resp.OpenedIds {
				fmt.Println(id)
			}
		}
		return nil
	},
}

func init() {
	flags := openCmd.Flags()
	flags.String("user", "", "SSH username (fallback to ~/.ssh/config)")
	flags.StringP("port", "P", "", "SSH port (default 22 or from ~/.ssh/config)")
	flags.String("password", "", "SSH password")
	flags.StringSliceP("publish", "p", nil, "Publish forwards, syntax <local-addr>:<local-port>:<remote-addr>:<local-port>, if \"<local-addr>:\" or \"<remote-addr>:\" is omitted then 0.0.0.0 will be used")

	_ = viper.BindPFlags(flags)

	rootCmd.AddCommand(openCmd)
}
