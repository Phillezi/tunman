package manager

import (
	"context"
	"sync"

	"github.com/Phillezi/tunman-remaster/pkg/tunnel"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"go.uber.org/zap"
)

type Manager struct {
	ctx context.Context
	ctrlpb.UnimplementedTunnelServiceServer
	tunnels map[string]*WTunnel
	mu      sync.RWMutex
}

type WTunnel struct {
	*tunnel.Tunnel
	cancel context.CancelFunc
}

func New() *Manager {
	return &Manager{
		ctx:     context.Background(),
		tunnels: make(map[string]*WTunnel),
	}
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tunnels {
		t.Close()
		t.cancel()
	}
	m.tunnels = make(map[string]*WTunnel)
}

func (m *Manager) findOrCreate(remote tunnel.ConnOpts) (*WTunnel, error) {
	hash := remote.Hash()
	m.mu.RLock()
	if t, found := m.tunnels[hash]; found {
		defer m.mu.RUnlock()
		return t, nil
	}
	m.mu.RUnlock()

	tunCtx, tunCan := context.WithCancel(m.ctx)
	tun, err := tunnel.New(remote.User, remote.Addr, append(remote.Opts, tunnel.WithContext(tunCtx))...)
	if err != nil {
		tunCan()
		return nil, err
	}
	wtun := &WTunnel{
		Tunnel: tun,
		cancel: tunCan,
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.tunnels[hash] = wtun

	return wtun, nil
}

func (m *Manager) Forward(remote tunnel.ConnOpts, localAddr string, remoteAddr string) error {
	tun, err := m.findOrCreate(remote)
	if err != nil {
		return err
	}

	go func() {
		if err := tun.Forward(localAddr, remoteAddr); err != nil {
			zap.L().Error("error on fwd", zap.String("localAddr", localAddr), zap.String("remoteAddr", remoteAddr), zap.Error(err))
		}
	}()
	return nil
}

func (m *Manager) Ps(_ context.Context, _ *ctrlpb.PsRequest) (*ctrlpb.PsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var fwds []*ctrlpb.Fwd
	for _, t := range m.tunnels {
		parent := t.Proto()
		for _, a := range parent.AddressPair {
			// TODO: proper ids
			fwds = append(fwds, &ctrlpb.Fwd{Id: "TODO", Addrs: a, Parent: parent})
		}
	}

	return &ctrlpb.PsResponse{Fwds: fwds}, nil
}

func (m *Manager) OpenFwd(_ context.Context, req *ctrlpb.OpenRequest) (*ctrlpb.OpenResponse, error) {
	var opened []string
	var errors []string = make([]string, 0)

	for _, tf := range req.Tunnels {

		remote := tunnel.ConnOpts{
			User: tf.User,
			Addr: tf.Addr,
			Opts: tunnel.WithProtoOpts(tf.Pw, tf.Privkey),
		}

		for _, fw := range tf.AddressPair {
			if err := m.Forward(remote, fw.LocalAddr, fw.RemoteAddr); err != nil {
				errors = append(errors, err.Error())
				zap.L().Warn("failed to forward", zap.String("remoteAddr", fw.RemoteAddr), zap.Error(err))
				continue
			}
		}

		// TODO: proper ids
		opened = append(opened, remote.Hash())
	}

	return &ctrlpb.OpenResponse{OpenedIds: opened, Errors: errors}, nil
}
