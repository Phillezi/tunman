package cli

import (
	"fmt"
	"os"

	"github.com/Phillezi/tunman-remaster/internal/connection"
	"github.com/Phillezi/tunman-remaster/internal/parser"
	"github.com/Phillezi/tunman-remaster/internal/ssh"
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
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		port := viper.GetString("port")
		userVal := viper.GetString("user")
		pw := viper.GetString("password")

		localRemoteMap, err := parser.ParsePublishes(viper.GetStringSlice("publish"))
		if err != nil {
			zap.L().Error("failed to parse", zap.Error(err))
			os.Exit(1)
		}
		if len(localRemoteMap) == 0 {
			zap.L().Error("no fwds provided")
			os.Exit(1)
		}

		cfg := ssh.Resolve(host)

		if cfg.UseAgent {
			//zap.L().Warn("using ssh agent is not impl yet")
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
				User: utils.Or(userVal, cfg.User, os.Getenv("USER")),
				Host: cfg.Host,
				Port: utils.ParsePort(utils.Or(port, cfg.Port)),
				Pw:   pw,
				Privkey: func() []byte {
					if len(cfg.PrivateKey) == 0 {
						return nil
					}
					return cfg.PrivateKey
				}(),
				AddressPair: addrPairs,
			}}})
			if err != nil {
				zap.L().Error("failed to execute open command", zap.Error(err))
				return
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
