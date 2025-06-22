package tunnel

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Phillezi/tunman-remaster/interrupt"
	"github.com/Phillezi/tunman-remaster/pkg/ser"
	sshutils "github.com/Phillezi/tunman-remaster/pkg/ssh"
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
	h := fnv.New64a()
	h.Write([]byte(localAddr + remoteAddr))
	return strconv.FormatUint(h.Sum64(), 16) // 16 hex chars
}

func (a *AddressPair) Hash() string {
	if a.hash != "" {
		return a.hash
	}
	a.hash = HashAddrPair(a.LocalAddr, a.RemoteAddr)
	return a.hash
}

type FwdConn struct {
	AddrPair AddressPair
	Cancel   context.CancelFunc
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
		if len(cfg.Auth) == 0 {
			cfg.Auth = []ssh.AuthMethod{ssh.Password(password)}
		} else {
			cfg.Auth = utils.Prepend(cfg.Auth, ssh.Password(password))
		}
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
		if len(cfg.Auth) == 0 {
			cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
		} else {
			cfg.Auth = utils.Prepend(cfg.Auth, ssh.PublicKeys(signer))
		}
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
		Host:        t.uID.Host,
		Port:        uint32(t.uID.Port),
		AddressPair: AddrPairToProto(t.conns),
	}
}

type ConnOpts struct {
	User string
	Host string
	Port uint
	addr string
	Opts []ConfigOption
}

// New creates a new SSH tunnel to host (user@addr).
func New(user, host string, port uint, opts ...ConfigOption) (*Tunnel, error) {

	fallbackCtx, fallbackCancel := context.WithCancel(context.Background())
	cfg := &TunnelOpts{
		ClientConfig: ssh.ClientConfig{
			User: user,
		},
		ctx:    fallbackCtx,
		cancel: fallbackCancel,
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	client, err := sshutils.DialWithJumpChain(&sshutils.Target{
		User: user,
		Host: host,
		Port: port,
	}, &cfg.ClientConfig)
	if err != nil {
		return nil, err
	}

	return &Tunnel{
		ctx:    cfg.ctx,
		cancel: cfg.cancel,
		client: client,
		uID: &ConnOpts{
			User: user,
			Host: host,
			Port: port,
			addr: "",
		},
		conns: make(map[string]*FwdConn),
	}, nil
}

func (o *ConnOpts) Hash() string {
	if o.addr == "" {
		addr, err := sshutils.Resolve(&sshutils.Target{User: o.User, Host: o.Host, Port: o.Port})
		if err != nil {
			zap.L().Error("failed to resolve addr for hashing", zap.Error(err))
			// fallback
			o.addr = fmt.Sprintf("%s:%d", utils.Or(o.Host, "0.0.0.0"), utils.Or(o.Port, 22))
		} else {
			o.addr = addr
		}
	}
	h := fnv.New64a()
	h.Write([]byte(o.User + o.addr))
	return strconv.FormatUint(h.Sum64(), 16) // 16 hex chars
}

func (t *Tunnel) Hash() string {
	t.once.Do(func() {
		t.hash = t.uID.Hash()
	})
	return t.hash
}

func (t *Tunnel) Exists(id string) bool {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	_, found := t.conns[id]
	return found
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

// DialWCtx opens a connection through the tunnel to the target (e.g., localhost:3306).
func (t *Tunnel) DialWCtx(ctx context.Context, network, addr string) (net.Conn, error) {
	if t.client == nil {
		return nil, errors.New("ssh client not connected")
	}
	conn, err := t.client.DialContext(ctx, network, addr)
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

	t.connMu.RLock()
	for id, c := range t.conns {
		// we need to call cancel to close all conns,
		// since this cancel func closes the listener
		// otherwise the go func is blocked listening
		// and will not recv the ctx cancel : /
		c.Cancel()
		zap.L().Debug("cancelled fwd", zap.String("id", id))
	}
	t.connMu.RUnlock()

	if t.client != nil {
		defer func() { t.client.Wait(); zap.L().Debug("ssh client closed"); t.client = nil }()
		return t.client.Close()
	}

	return nil
}

func (t *Tunnel) FwdsCount() int {
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	return len(t.conns)
}

func (t *Tunnel) CloseFwd(ids ...string) ([]string, []string) {
	var closed []string
	var errors []string
	t.connMu.RLock()
	defer t.connMu.RUnlock()
	for _, id := range ids {
		if v, ok := t.conns[id]; ok {
			go func() {
				if v.Cancel != nil {
					v.Cancel()
					zap.L().Info("closed forward", zap.String("id", id))
				}
			}()
			closed = append(closed, ser.Ser(t.Hash(), id))
		} else {
			errors = append(errors, fmt.Sprintf("fwd with id %s not found", id))
		}
	}
	return closed, errors
}

// Forward listens on localAddr (e.g. "localhost:8080") and forwards all connections
// to remoteAddr (e.g. "localhost:5432") through the SSH tunnel.
func (t *Tunnel) Forward(ap AddressPair) error {
	id := ap.Hash()
	defer zap.L().Debug("Forward exited", zap.String("id", id))
	var once sync.Once

	listener, err := net.Listen("tcp", ap.LocalAddr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	ctx, cancel := context.WithCancel(t.ctx)
	defer once.Do(func() {
		listener.Close()
		cancel()
	})

	t.connMu.Lock()
	t.conns[id] = &FwdConn{AddrPair: ap, Cancel: func() {
		once.Do(func() {
			listener.Close()
			cancel()
		})
	}}
	t.connMu.Unlock()
	defer func() {
		go func() {
			ctxx, ccancel := context.WithTimeout(interrupt.GetInstance().Context(), 10*time.Second)
			defer ccancel()
			select {
			case <-ctxx.Done():
				zap.L().Warn("timed out when trying to remove fwd on exit from fwds in tunnel", zap.String("id", id))
			default:
				t.connMu.Lock()
				defer t.connMu.Unlock()
				delete(t.conns, id)
				zap.L().Debug("succesfully removed fwd from fwds in tunnel", zap.String("id", id))
			}
		}()
	}()

	zap.L().Info("Forwarding", zap.String("id", id), zap.String("local", ap.LocalAddr), zap.String("remote", ap.RemoteAddr))

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("context cancelled, exiting", zap.String("id", id))
			return nil
		default:
			localConn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok {
					// Handle "use of closed network connection"
					// the actual error is poll.errNetClosing, but it is private so i cant check if it is that
					if strings.Contains(opErr.Err.Error(), "use of closed network connection") {
						return nil
					}
				}
				return fmt.Errorf("accept error: %w", err)
			}

			go t.handleForwardConn(ctx, localConn, ap.RemoteAddr)
		}
	}
}

func (t *Tunnel) handleForwardConn(ctx context.Context, localConn net.Conn, remoteAddr string) {
	//defer zap.L().Debug("handleForwardConn exited")
	defer localConn.Close()

	remoteConn, err := t.DialWCtx(ctx, "tcp", remoteAddr)
	if err != nil {
		zap.L().Error("SSH dial failed", zap.Error(err))
		return
	}
	defer remoteConn.Close()

	go func() {
		<-ctx.Done()
		zap.L().Debug("handleForwardConn recv ctx cancelled")
		localConn.Close()
		remoteConn.Close()
	}()

	go io.Copy(remoteConn, localConn)
	io.Copy(localConn, remoteConn)
}
