package ssh

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/Phillezi/tunman-remaster/utils"
	"github.com/kevinburke/ssh_config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type Target struct {
	User string
	Host string
	Port uint
}

func getSSHClientConfig(target *Target, cfgs ...*ssh.ClientConfig) (*ssh.ClientConfig, error) {
	var cfg *ssh.ClientConfig
	if len(cfgs) > 0 && cfgs[0] != nil {
		cfg = cfgs[0]
	} else {
		cfg = &ssh.ClientConfig{}
	}

	// Set user if not already set
	if cfg.User == "" && target.User == "" {
		var err error
		target.User, err = ssh_config.GetStrict(target.Host, "User")
		if err != nil || target.User == "" {
			zap.L().Warn("failed to get user from ssh config", zap.Error(err))
			if viper.GetBool("default-to-user") {
				zap.L().Info("defaulting to user running daemon")
				if usr, err := user.Current(); err == nil {
					target.User = usr.Username
				} else {
					zap.L().Warn("failed to get current user", zap.Error(err))
					target.User = "root"
				}
			}
		}
	}
	if cfg.User == "" {
		cfg.User = target.User
	}

	// Set port if not set
	if target.Port == 0 {
		portStr, err := ssh_config.GetStrict(target.Host, "Port")
		if err != nil {
			zap.L().Error("error retrieving port from ssh config", zap.Error(err))
			target.Port = 22
		} else {
			if p, err := strconv.Atoi(portStr); err == nil {
				target.Port = uint(p)
			} else {
				zap.L().Error("error parsing port", zap.Error(err))
				target.Port = 22
			}
		}
	}

	// Start building auth methods
	var authOpts []ssh.AuthMethod

	// Add SSH agent auth if available
	if agentAuth, err := GetSSHAgentAuth(); err == nil {
		authOpts = append(authOpts, agentAuth)
	} else {
		zap.L().Warn("could not get ssh agent signers", zap.Error(err))
	}

	// Use IdentityFile from ssh config
	keyFile, err := ssh_config.GetStrict(target.Host, "IdentityFile")
	if err != nil {
		zap.L().Error("error retrieving IdentityFile", zap.Error(err))
	}
	if keyFile != "" {
		keyFile = utils.EvalPath(keyFile)
		if _, err := os.Stat(keyFile); err == nil {
			if pk, err := loadPrivateKey(keyFile); err == nil {
				authOpts = append(authOpts, ssh.PublicKeys(pk))
			} else {
				return nil, fmt.Errorf("failed to load private key: %w", err)
			}
		} else if keyFile != utils.EvalPath(ssh_config.Default("IdentityFile")) {
			zap.L().Warn("specified IdentityFile does not exist", zap.String("keyFile", keyFile))
		}
	}

	if len(cfg.Auth) == 0 {
		cfg.Auth = authOpts
	} else {
		cfg.Auth = append(cfg.Auth, authOpts...)
	}

	// HostKeyCallback
	if !viper.GetBool("insecure") && !viper.GetBool("insecure-skip-hostkey-callback") {
		hostKeyCallback, err := GetHostKeyCallback()
		if err != nil {
			zap.L().Warn("could not get host key callback", zap.Error(err))
			if !viper.GetBool("insecure") && !viper.GetBool("insecure-skip-hostkey-callback") {
				return nil, err
			}
			hostKeyCallback = ssh.InsecureIgnoreHostKey()
		}
		cfg.HostKeyCallback = hostKeyCallback
	} else {
		cfg.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	// Set timeout
	cfg.Timeout = 10 * time.Second

	return cfg, nil
}

// loadPrivateKey loads the SSH private key from the user's .ssh directory.
func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	privateKey, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return privateKey, nil
}

func loadPublicKey(keyPath string) (ssh.PublicKey, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	publicKey, err := ssh.ParsePublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return publicKey, nil
}
