package services

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/secrets"
)

func (h *Handler) executeSSH(ctx context.Context, serverID string, command string) ([]byte, error) {
	return h.executeSSHInput(ctx, serverID, command, nil)
}

func (h *Handler) connectSSH(ctx context.Context, serverID string) (*ssh.Client, error) {
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
	return ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), &ssh.ClientConfig{User: username, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.FixedHostKey(hostKey), HostKeyAlgorithms: []string{hostKey.Type()}, Timeout: 15 * time.Second})
}

func (h *Handler) executeSSHInput(ctx context.Context, serverID string, command string, stdin []byte) ([]byte, error) {
	connection, err := h.connectSSH(ctx, serverID)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	if stdin != nil {
		session.Stdin = bytes.NewReader(stdin)
	}
	output, err := session.CombinedOutput(command)
	if err != nil {
		return output, fmt.Errorf("remote docker command failed: %w: %s", err, output)
	}
	return output, nil
}
