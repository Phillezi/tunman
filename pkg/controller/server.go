package controller

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"

	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func New(man ctrlpb.TunnelServiceServer) *grpc.Server {
	s := grpc.NewServer()
	ctrlpb.RegisterTunnelServiceServer(s, man)
	return s
}

// ListenAndServe sets up and starts the gRPC server based on the given address and TLS config.
func ListenAndServe(addr string, service ctrlpb.TunnelServiceServer, tlsConfig *tls.Config) error {
	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	lis, err := createListener(u)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	ctrlpb.RegisterTunnelServiceServer(server, service)

	zap.L().Info("Ctrl server listening", zap.String("scheme", u.Scheme), zap.String("address", addr))

	if u.Scheme == "https" && tlsConfig != nil {
		return server.Serve(tls.NewListener(lis, tlsConfig))
	}

	return server.Serve(lis)
}

// createListener initializes the appropriate listener based on the URL scheme.
func createListener(u *url.URL) (net.Listener, error) {
	switch u.Scheme {
	case "unix":
		return createUnixListener(u.Path)
	case "tcp", "https":
		return createTCPListener(u.Host)
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

// createUnixListener sets up a Unix domain socket listener.
func createUnixListener(path string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("unix sockets are not supported on Windows")
	}
	_ = os.Remove(path) // Remove old socket file if exists
	return net.Listen("unix", path)
}

// createTCPListener sets up a TCP listener.
func createTCPListener(host string) (net.Listener, error) {
	if host == "" {
		return nil, fmt.Errorf("missing host in address")
	}
	return net.Listen("tcp", host)
}
