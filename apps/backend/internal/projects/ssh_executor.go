package projects

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/secrets"
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

func (h *Handler) executeSSH(ctx context.Context, serverID string, command string) ([]byte, error) {
	var host, username string
	var port int
	var encryptedKey, encodedHostKey []byte
	err := h.db.QueryRowContext(ctx, "SELECT host, ssh_port, ssh_user, encrypted_ssh_key, ssh_host_key FROM servers WHERE id = ? AND encrypted_ssh_key IS NOT NULL", serverID).Scan(&host, &port, &username, &encryptedKey, &encodedHostKey)
	if err != nil {
		return nil, err
	}
	privateKey, err := secrets.Decrypt(encryptedKey)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	hostKey, err := ssh.ParsePublicKey(encodedHostKey)
	if err != nil {
		return nil, err
	}
	connection, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), &ssh.ClientConfig{User: username, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.FixedHostKey(hostKey), HostKeyAlgorithms: []string{hostKey.Type()}, Timeout: 15 * time.Second})
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	output, err := session.CombinedOutput(command)
	if err != nil {
		return output, fmt.Errorf("remote docker command failed: %w: %s", err, output)
	}
	return output, nil
}
