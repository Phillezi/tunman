package ssh

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func GetSSHAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not found")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %v", err)
	}

	agentClient := agent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		zap.L().Error("error getting signers from ssh agent", zap.Error(err))
	} else if len(signers) == 0 {
		zap.L().Warn("no signers available from the ssh agent, make sure you add your signers to the ssh agent")
	}

	return ssh.PublicKeysCallback(agentClient.Signers), nil
}
