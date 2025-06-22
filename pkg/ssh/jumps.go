package ssh

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"

	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/kevinburke/ssh_config"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func Resolve(target *Target) (string, error) {
	jumpChain, err := ssh_config.GetStrict(target.Host, "ProxyJump")
	if err != nil {
		zap.L().Debug("no ProxyJump entry", zap.String("host", target.Host), zap.Error(err))
	}

	var lastTarget = &Target{
		User: target.User,
		Host: target.Host,
		Port: target.Port,
	}

	if jumpChain != "" {
		jumps := strings.SplitSeq(jumpChain, ",")
		for jump := range jumps {
			lastTarget = &Target{
				Host: strings.TrimSpace(jump),
			}
			resolveTargetFields(lastTarget)
		}
	}

	resolveTargetFields(lastTarget)

	host := ssh_config.Get(lastTarget.Host, "HostName")
	if host == "" {
		host = lastTarget.Host
	}

	port := func() string {
		if lastTarget.Port != 0 {
			return fmt.Sprintf("%d", lastTarget.Port)
		}
		p := ssh_config.Get(lastTarget.Host, "Port")
		if p == "" {
			return "22"
		}
		return p
	}()

	addr := fmt.Sprintf("%s:%s", host, port)
	return addr, nil
}

func resolveTargetFields(t *Target) {
	if t.User == "" {
		if u, err := ssh_config.GetStrict(t.Host, "User"); err == nil && u != "" {
			t.User = u
		} else if u2, err := user.Current(); err == nil {
			t.User = u2.Username
		} else {
			t.User = "user"
		}
	}

	if t.Port == 0 {
		if p := ssh_config.Get(t.Host, "Port"); p != "" {
			if pint, err := strconv.Atoi(p); err == nil {
				t.Port = uint(pint)
			}
		}
	}
}

func DialWithJumpChain(target *Target, cfgs ...*ssh.ClientConfig) (*ssh.Client, error) {
	jumpChain, err := ssh_config.GetStrict(target.Host, "ProxyJump")
	if err != nil || jumpChain == "" {
		return DialDirect(target, nil, cfgs...)
	}

	jumps := strings.Split(jumpChain, ",")
	var client *ssh.Client

	for _, jump := range jumps {
		jumpTarget := &Target{Host: jump}
		cfg, err := getSSHClientConfig(jumpTarget, cfgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH config for jump %s: %w", jump, err)
		}

		host := utils.Or(ssh_config.Get(jump, "HostName"), jump, "0.0.0.0")
		port := utils.Or(ssh_config.Get(jump, "Port"), "22")
		addr := fmt.Sprintf("%s:%s", host, port)

		client, err = createSSHClient(client, addr, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to jump %s: %w", jump, err)
		}
	}

	// Final target
	return DialDirect(target, client)
}

func DialDirect(target *Target, through *ssh.Client, cfgs ...*ssh.ClientConfig) (*ssh.Client, error) {
	cfg, err := getSSHClientConfig(target, cfgs...)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%s", utils.Or(ssh_config.Get(target.Host, "HostName"), target.Host, "0.0.0.0"), utils.Or(func() string {
		if target.Port == 0 {
			return ""
		}
		return fmt.Sprintf("%d", target.Port)
	}(), ssh_config.Get(target.Host, "Port"), "22"))

	if through == nil {
		return ssh.Dial("tcp", addr, cfg)
	}

	return createSSHClient(through, addr, cfg)
}
