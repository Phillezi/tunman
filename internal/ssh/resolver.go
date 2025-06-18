package ssh

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/kevinburke/ssh_config"
	"go.uber.org/zap"
)

type Config struct {
	User         string
	Host         string
	Port         string
	IdentityFile string
	PrivateKey   []byte
	UseAgent     bool
}

func Resolve(host string) Config {
	cfgPath := filepath.Join(getHomeDir(), ".ssh", "config")
	f, err := os.Open(cfgPath)
	if err != nil {
		return Config{}
	}
	defer f.Close()

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		return Config{}
	}

	user, err := cfg.Get(host, "User")
	if err != nil {
		zap.L().Warn("failed to get User", zap.Error(err))
	}
	hostname, err := cfg.Get(host, "Hostname")
	if err != nil {
		zap.L().Warn("failed to get Hostname", zap.Error(err))
	}
	port, err := cfg.Get(host, "Port")
	if err != nil {
		zap.L().Warn("failed to get Port", zap.Error(err))
	}
	identity, err := cfg.Get(host, "IdentityFile")
	if err != nil {
		zap.L().Warn("failed to get IdentityFile", zap.Error(err))
	}

	// Expand ~ in IdentityFile
	if len(identity) > 1 && identity[:2] == "~/" {
		identity = filepath.Join(getHomeDir(), identity[2:])
	}

	useAgent := os.Getenv("SSH_AUTH_SOCK") != "" // detect ssh-agent

	// If the IdentityFile doesn't exist, ignore it
	var privateKey []byte
	if identity != "" {
		if data, err := os.ReadFile(identity); err == nil {
			privateKey = data
		} else {
			identity = "" // fallback to empty if file is unreadable
		}
	}

	return Config{
		User:         user,
		Host:         hostname,
		Port:         port,
		IdentityFile: identity,
		PrivateKey:   privateKey,
		UseAgent:     useAgent,
	}
}

func getHomeDir() string {
	u, err := user.Current()
	if err != nil {
		return utils.Or(os.Getenv("XDG_HOME"), os.Getenv("HOME"))
	}
	return u.HomeDir
}
