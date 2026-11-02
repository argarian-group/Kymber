package servers

import (
	"context"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

func validateSSHAccess(ctx context.Context, host string, port int, username string, privateKey []byte, hostKey ssh.PublicKey) error {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return err
	}
	connection, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), &ssh.ClientConfig{User: username, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.FixedHostKey(hostKey), HostKeyAlgorithms: []string{hostKey.Type()}, Timeout: 10 * time.Second})
	if err != nil {
		return err
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run("docker version --format '{{.Server.Version}}'")
}
