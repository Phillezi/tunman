package ssh

import (
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func GetHostKeyCallback() (ssh.HostKeyCallback, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	knownHostsPath := filepath.Join(userHomeDir, ".ssh", "known_hosts")
	return knownhosts.New(knownHostsPath)
}
