package controller

import (
	ctrlpb "github.com/Phillezi/tunman/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Dial(socket string) (ctrlpb.TunnelServiceClient, error) {
	conn, err := grpc.NewClient(socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return ctrlpb.NewTunnelServiceClient(conn), nil
}
