package tunnel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/Phillezi/tunman-remaster/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type AddressPair struct {
	LocalAddr  string
	RemoteAddr string
}

func AddrPairToProto(addrs []AddressPair) []*ctrlpb.AddrPair {
	var protoAddrs = []*ctrlpb.AddrPair{}
	for _, a := range addrs {
		protoAddrs = append(protoAddrs, utils.PtrOf(a.Proto()))
	}
	return protoAddrs
}

func (a *AddressPair) Proto() ctrlpb.AddrPair {
	return ctrlpb.AddrPair{
		LocalAddr:  a.LocalAddr,
		RemoteAddr: a.RemoteAddr,
	}
}

type Tunnel struct {
	ctx    context.Context
	cancel context.CancelFunc

	uID  *ConnOpts
	once sync.Once
	hash string

	client *ssh.Client

	activeConns []AddressPair
	connMu      sync.RWMutex
}

type TunnelOpts struct {
	ssh.ClientConfig
	ctx    context.Context
	cancel context.CancelFunc
}

type ConfigOption func(*TunnelOpts) error

// WithContext returns an option to use a specified context.
func WithContext(ctx context.Context) ConfigOption {
	return func(cfg *TunnelOpts) error {
		cfg.ctx, cfg.cancel = context.WithCancel(ctx)
		return nil
	}
}

// WithPassword returns an option to authenticate with password.
func WithPassword(password string) ConfigOption {
	return func(cfg *TunnelOpts) error {
		cfg.Auth = []ssh.AuthMethod{ssh.Password(password)}
		return nil
	}
}

// WithPrivateKey returns an option to authenticate with a private key ([]byte form).
func WithPrivateKey(key []byte) ConfigOption {
	return func(cfg *TunnelOpts) error {
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return err
		}
		cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		return nil
	}
}

func WithProtoOpts(pw string, key []byte) []ConfigOption {
	var opts = []ConfigOption{}
	if pw != "" {
		opts = append(opts, WithPassword(pw))
	}
	if len(key) > 0 {
		opts = append(opts, WithPrivateKey(key))
	}
	return opts
}

func (t *Tunnel) Proto() *ctrlpb.Tunnel {
	return &ctrlpb.Tunnel{
		Id:          t.Hash(),
		User:        t.uID.User,
		Addr:        t.uID.Addr,
		AddressPair: AddrPairToProto(t.activeConns),
	}
}

type ConnOpts struct {
	User string
	Addr string
	Opts []ConfigOption
}

// New creates a new SSH tunnel to host (user@addr).
func New(user, addr string, opts ...ConfigOption) (*Tunnel, error) {
	fallbackCtx, fallbackCancel := context.WithCancel(context.Background())
	cfg := &TunnelOpts{
		ClientConfig: ssh.ClientConfig{
			User:            user,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Insecure, replace in production
			Timeout:         10 * time.Second,
		},
		ctx:    fallbackCtx,
		cancel: fallbackCancel,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	client, err := ssh.Dial("tcp", addr, &cfg.ClientConfig)
	if err != nil {
		return nil, err
	}

	return &Tunnel{
		ctx:    cfg.ctx,
		cancel: cfg.cancel,
		client: client,
		uID: &ConnOpts{
			User: user,
			Addr: addr,
		},
		activeConns: make([]AddressPair, 0),
	}, nil
}

func (o *ConnOpts) Hash() string {
	data := []byte(o.User + o.Addr)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (t *Tunnel) Hash() string {
	t.once.Do(func() {
		t.hash = t.uID.Hash()
	})
	return t.hash
}

// Dial opens a connection through the tunnel to the target (e.g., localhost:3306).
func (t *Tunnel) Dial(network, addr string) (net.Conn, error) {
	if t.client == nil {
		return nil, errors.New("ssh client not connected")
	}
	conn, err := t.client.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	t.connMu.Lock()
	defer t.connMu.Unlock()
	t.activeConns = append(t.activeConns, AddressPair{RemoteAddr: addr})
	return conn, nil
}

// Close shuts down the SSH tunnel.
func (t *Tunnel) Close() error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.client != nil {
		return t.client.Close()
	}
	<-t.ctx.Done()
	return nil
}

// Forward listens on localAddr (e.g. "localhost:8080") and forwards all connections
// to remoteAddr (e.g. "localhost:5432") through the SSH tunnel.
func (t *Tunnel) Forward(localAddr, remoteAddr string) error {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	t.connMu.Lock()
	t.activeConns = append(t.activeConns, AddressPair{LocalAddr: localAddr, RemoteAddr: remoteAddr})
	t.connMu.Unlock()
	fmt.Printf("Forwarding %s -> %s (via SSH)\n", localAddr, remoteAddr)

	for {
		select {
		case <-t.ctx.Done():
			zap.L().Info("ctx cancelled, exitting")
			return nil
		default:
			localConn, err := listener.Accept()
			if err != nil {
				return fmt.Errorf("accept error: %w", err)
			}

			go func() {
				defer localConn.Close()

				remoteConn, err := t.Dial("tcp", remoteAddr)
				if err != nil {
					zap.L().Error("ssh dial error", zap.Error(err))
					return
				}
				defer remoteConn.Close()

				// Copy both ways
				go io.Copy(remoteConn, localConn)
				io.Copy(localConn, remoteConn)
			}()
		}
	}
}
