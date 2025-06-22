package ssh

import "golang.org/x/crypto/ssh"

func createSSHClient(from *ssh.Client, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	if from == nil {
		return ssh.Dial("tcp", addr, cfg)
	}
	conn, err := from.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	ncc, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		return nil, err
	}
	return ssh.NewClient(ncc, chans, reqs), nil
}
