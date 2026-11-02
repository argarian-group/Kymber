package services

import (
	"bytes"
	"context"
	"fmt"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/sshclient"
)

func (h *Handler) executeSSH(ctx context.Context, serverID string, command string) ([]byte, error) {
	return h.executeSSHInput(ctx, serverID, command, nil)
}

func (h *Handler) connectSSH(ctx context.Context, serverID string) (*ssh.Client, error) {
	return sshclient.Connect(ctx, h.db, serverID)
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
