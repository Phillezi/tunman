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

	hash string
}

func AddrPairToProto(addrs map[string]*FwdConn) map[string]*ctrlpb.AddrPair {
	protoAddrs := make(map[string]*ctrlpb.AddrPair, len(addrs))
	for hash, a := range addrs {
		protoAddrs[hash] = utils.PtrOf(a.AddrPair.Proto())
	}
	return protoAddrs
}

func (a *AddressPair) Proto() ctrlpb.AddrPair {
	return ctrlpb.AddrPair{
		LocalAddr:  a.LocalAddr,
		RemoteAddr: a.RemoteAddr,
	}
}

func HashAddrPair(localAddr, remoteAddr string) string {
	data := []byte(localAddr + remoteAddr)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (a *AddressPair) Hash() string {
	if a.hash != "" {
		return a.hash
	}
	a.hash = HashAddrPair(a.LocalAddr, a.RemoteAddr)
	return a.hash
}

type FwdConn struct {
	AddrPair  AddressPair
	CloseChan chan struct{}
}

type Tunnel struct {
	ctx    context.Context
	cancel context.CancelFunc

	uID  *ConnOpts
	once sync.Once
	hash string

	client *ssh.Client

	conns  map[string]*FwdConn
	connMu sync.RWMutex
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
		AddressPair: AddrPairToProto(t.conns),
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
		conns: make(map[string]*FwdConn),
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

func (t *Tunnel) CloseFwd(ids ...string) ([]string, []string) {
	var closed []string
	var errors []string
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	for _, id := range ids {
		if v, ok := t.conns[id]; ok {
			go func() {
				select {
				case v.CloseChan <- struct{}{}:
					zap.L().Info("closed forward", zap.String("id", id))
				default:
					zap.L().Warn("could not close forward, channel full!", zap.String("id", id))
				}
			}()
			closed = append(closed, id)
		} else {
			errors = append(errors, fmt.Sprintf("fwd with id %s not found", id))
		}
	}
	return closed, errors
}

// Forward listens on localAddr (e.g. "localhost:8080") and forwards all connections
// to remoteAddr (e.g. "localhost:5432") through the SSH tunnel.
func (t *Tunnel) Forward(localAddr, remoteAddr string) error {
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	closeMe := make(chan struct{}, 1)
	defer close(closeMe)
	ap := AddressPair{LocalAddr: localAddr, RemoteAddr: remoteAddr}
	id := ap.Hash() // calc hash before locking (gets saved in the struct)
	t.connMu.Lock()
	t.conns[id] = &FwdConn{AddrPair: ap, CloseChan: closeMe}
	t.connMu.Unlock()
	fmt.Printf("[%s] Forwarding %s -> %s (via SSH)\n", ap.Hash(), localAddr, remoteAddr)

	rm := func() {
		t.connMu.Lock()
		defer t.connMu.Unlock()
		delete(t.conns, id)
	}

	for {
		select {
		case <-t.ctx.Done():
			zap.L().Info("ctx cancelled, exitting")
			return nil
		case <-closeMe:
			zap.L().Info("Fwd handler recv close req, closing...", zap.String("id", id))
			rm()
			return nil
		default:
			localConn, err := listener.Accept()
			if err != nil {
				rm()
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
