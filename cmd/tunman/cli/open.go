package cli

import (
	"fmt"
	"strings"

	"github.com/Phillezi/tunman-remaster/internal/connection"
	"github.com/Phillezi/tunman-remaster/internal/parser"
	"github.com/Phillezi/tunman-remaster/interrupt"
	sshutil "github.com/Phillezi/tunman-remaster/pkg/ssh"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var openCmd = &cobra.Command{
	Use:   "open [target]",
	Short: "Open a tunnel to a remote target",
	Long: `The open command allows you to open a single or multiple tunnels to a specified ssh target. 
This target can be specified using the familiar syntax of ssh (<user>@<host>:<port>) with a combination of flags, for example --user or --port, 
things specified with flags take priority. You can also refer to custom hosts specified within your ssh config file (~/.ssh/config),
this config will be read and parsed to open the ssh connection, it even works with proxy-jumps.

Specifying ports to "publish" takes inspiration from how it is done within the docker cli, using -p or --publish per pair you want to publish and ":" as a delimiter.
Bind addresses are optionally specified, if omitted they default to 0.0.0.0.`,
	Example: `tunman open testserver -p 8080:8080 -p 9090:7070 -p 5050:10.0.12.1:5050 -p localhost:4040:4040
# The command above will look up testserver in the users (the user running the daemon) ~/.ssh/config and open a tunnel
# it will then forward the published port address combinations that are specified

tunman open root@localhost:2222 -p 8080:8090
# The command above will open a tunnel and forward port 8090 inside the ssh host to 8080 of the host running the command.`,
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var completions []cobra.Completion
		for _, c := range sshutil.GetHosts() {
			if strings.HasPrefix(c, toComplete) {
				completions = append(completions, c)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		u, h, p := parser.ParseTargetLoose(args[0])
		host := h
		port := utils.Or(viper.GetString("port"), p)
		userVal := utils.Or(viper.GetString("uservalue"), u)
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
	openCmd.Flags().StringP("user", "u", "", "SSH username")
	viper.BindPFlag("userval", openCmd.Flags().Lookup("user"))

	openCmd.Flags().StringP("port", "P", "", "SSH port")
	viper.BindPFlag("port", openCmd.Flags().Lookup("port"))

	openCmd.Flags().String("password", "", "SSH password")
	viper.BindPFlag("password", openCmd.Flags().Lookup("password"))

	openCmd.Flags().StringSliceP("publish", "p", nil, "Publish forwards, syntax <local-addr>:<local-port>:<remote-addr>:<local-port>, if \"<local-addr>:\" or \"<remote-addr>:\" is omitted then 0.0.0.0 will be used")
	viper.BindPFlag("publish", openCmd.Flags().Lookup("publish"))

	rootCmd.AddCommand(openCmd)
}
