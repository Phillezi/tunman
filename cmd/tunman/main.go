package main

import (
	"os"

	"github.com/Phillezi/tunman-remaster/cmd/tunman/cli"
	"github.com/Phillezi/tunman-remaster/log"
)

func init() {
	log.Setup()
}

func main() {
	if err := cli.ExecuteE(); err != nil {
		os.Exit(1)
	}
	/*ctrl, err := controller.Dial("/tmp/tunman.sock")
	if err != nil {
		zap.L().Error("failed to connect", zap.Error(err))
		os.Exit(1)
	}
	resp, err := ctrl.Ps(interrupt.GetInstance().Context(), &ctrlpb.PsRequest{})
	if err != nil {
		zap.L().Error("failed to req ps", zap.Error(err))
		os.Exit(1)
	}
	if len(resp.Fwds) > 0 {
		for _, fwd := range resp.Fwds {
			fmt.Println(fwd)
		}
	} else {
		fmt.Println("no tunnels")
	}

	respe, err := ctrl.OpenFwd(interrupt.GetInstance().Context(), &ctrlpb.OpenRequest{
		Tunnels: []*ctrlpb.Tunnel{{
			User: "philip",
			Addr: "localhost:22",
			Pw:   "mypassword",
			AddressPair: []*ctrlpb.AddrPair{{
				LocalAddr:  "localhost:9090",
				RemoteAddr: "localhost:8080",
			}},
		}}})

	if err != nil {
		zap.L().Error("failed to req open", zap.Error(err))
		os.Exit(1)
	}

	if len(respe.OpenedIds) > 0 {
		for _, opn := range respe.OpenedIds {
			fmt.Println(opn)
		}
	}

	for _, e := range respe.Errors {
		zap.L().Error(e)
	}*/
}
