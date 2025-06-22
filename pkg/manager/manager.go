package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Phillezi/tunman-remaster/internal/defaults"
	"github.com/Phillezi/tunman-remaster/interrupt"
	"github.com/Phillezi/tunman-remaster/pkg/repo"
	"github.com/Phillezi/tunman-remaster/pkg/ser"
	"github.com/Phillezi/tunman-remaster/pkg/tunnel"
	ctrlpb "github.com/Phillezi/tunman-remaster/proto"
	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Manager struct {
	ctx context.Context
	ctrlpb.UnimplementedTunnelServiceServer
	tunnels map[string]*WTunnel
	mu      sync.RWMutex

	db *repo.Repo
}

type WTunnel struct {
	*tunnel.Tunnel
	cancel context.CancelFunc
}

func New() *Manager {
	r, err := repo.OpenDB(utils.Or(viper.GetString("dbpath"), defaults.DefaultDBPath))
	if err != nil {
		zap.L().Error("failed to open db", zap.Error(err))
	}

	m := &Manager{
		ctx:     context.Background(),
		tunnels: make(map[string]*WTunnel),
		db:      r,
	}

	if r != nil {
		interrupt.GetInstance().AddShutdownHook(func() { r.Close() })

		start := time.Now()
		fwds, err := r.LoadAllFwds()
		end := time.Now()
		if err != nil {
			zap.L().Error("failed to load fwds", zap.Error(err))
			return m
		}
		zap.L().Info("loaded state in", zap.Duration("loadTime", end.Sub(start)))

		for _, fwd := range fwds {
			m.Forward(
				tunnel.ConnOpts{
					Host: fwd.Host,
					Port: uint(fwd.Port),
					User: fwd.User,
				},
				fwd.Addrs.LocalAddr, fwd.Addrs.RemoteAddr,
			)
		}

	}

	return m
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
	tun, err := tunnel.New(remote.User, remote.Host, remote.Port, append(remote.Opts, tunnel.WithContext(tunCtx))...)
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

	ap := tunnel.AddressPair{LocalAddr: localAddr, RemoteAddr: remoteAddr}
	if tun.Exists(ap.Hash()) {
		return fmt.Errorf("connection already exists")
	}

	if m.db != nil {
		if err := m.db.SaveFwd(&ctrlpb.FwdState{
			Id:    ser.Ser(tun.Hash(), ap.Hash()),
			Addrs: &ctrlpb.AddrPair{LocalAddr: ap.LocalAddr, RemoteAddr: ap.RemoteAddr},
			Host:  remote.Host,
			User:  remote.User,
			Port:  uint32(remote.Port),
		}); err != nil {
			zap.L().Warn("failed to persist fwd", zap.Error(err))
		}
	}

	go func() {
		if err := tun.Forward(ap); err != nil {
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
		for i, a := range parent.AddressPair {
			fwds = append(fwds, &ctrlpb.Fwd{Id: ser.Ser(parent.Id, i), Addrs: a, Parent: parent})
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
			Host: tf.Host,
			Port: uint(tf.Port),
			Opts: tunnel.WithProtoOpts(tf.Pw, tf.Privkey),
		}

		for _, fw := range tf.AddressPair {
			if err := m.Forward(remote, fw.LocalAddr, fw.RemoteAddr); err != nil {
				errors = append(errors, err.Error())
				zap.L().Warn("failed to forward", zap.String("remoteAddr", fw.RemoteAddr), zap.Error(err))
				continue
			}
			opened = append(opened, ser.Ser(remote.Hash(), tunnel.HashAddrPair(fw.LocalAddr, fw.RemoteAddr)))
		}
	}

	return &ctrlpb.OpenResponse{OpenedIds: opened, Errors: errors}, nil
}

func (m *Manager) CloseFwd(_ context.Context, req *ctrlpb.CloseRequest) (*ctrlpb.CloseResponse, error) {
	var closed []string
	var errors []string = make([]string, 0)

	tunConnMap := make(map[string]int)

	for _, id := range req.Ids {
		tunHash, addrHash, err := ser.DeSer(id)
		if err != nil {
			errors = append(errors, err.Error())
			zap.L().Warn("could not deserialize id into tunnel and addr hash", zap.Error(err))
			continue
		}
		if v, ok := m.tunnels[tunHash]; ok {
			if _, ok := tunConnMap[tunHash]; !ok {
				tunConnMap[tunHash] = v.FwdsCount()
			}
			closedD, errorsS := v.CloseFwd(addrHash)
			closedC := len(closedD)
			if closedC > 0 {
				closed = append(closed, closedD...)
				tunConnMap[tunHash] -= closedC
				if m.db != nil {
					if err := m.db.DeleteFwds(closedD...); err != nil {
						zap.L().Warn("failed to delete persisted fwds", zap.Error(err))
					}
				}
			}
			if len(errorsS) > 0 {
				errors = append(errors, errorsS...)
			}
			if tunConnMap[tunHash] <= 0 {
				v.Close()
				m.mu.Lock()
				delete(m.tunnels, tunHash)
				m.mu.Unlock()
				zap.L().Info("closed empty SSH tunnel")
			}
		} else {
			errors = append(errors, fmt.Sprintf("could not find tunnel by { \"id\": \"%s\"}", id))
		}
	}
	return &ctrlpb.CloseResponse{ClosedIds: closed, Errors: errors}, nil
}

func (m *Manager) CloseAllFwds(context.Context, *ctrlpb.CloseAllRequest) (*ctrlpb.CloseAllResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.tunnels) > 0 {
		for id, tun := range m.tunnels {
			tun.Close()
			zap.L().Info("closed tunnel", zap.String("id", id))
		}
		m.tunnels = make(map[string]*WTunnel)
		if m.db != nil {
			if err := m.db.NukeFwds(); err != nil {
				zap.L().Error("failed to nuke fwds bucket", zap.Error(err))
			}
		}
		return &ctrlpb.CloseAllResponse{Ok: true, Error: ""}, nil
	}
	return &ctrlpb.CloseAllResponse{Ok: false, Error: "No open tunnels"}, nil
}
