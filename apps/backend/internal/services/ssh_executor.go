package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

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

func (h *Handler) executeSSHStream(ctx context.Context, serverID, command string, stdin io.Reader) ([]byte, error) {
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
	stdout, stderr := &boundedSSHOutput{}, &boundedSSHOutput{}
	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr
	if err := session.Start(command); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- session.Wait() }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = session.Close()
		<-done
		return nil, ctx.Err()
	}
	output := append(stdout.Bytes(), stderr.Bytes()...)
	if err != nil {
		return output, fmt.Errorf("remote command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type boundedSSHOutput struct{ bytes.Buffer }

func (b *boundedSSHOutput) Write(data []byte) (int, error) {
	if b.Len() < 8<<20 {
		_, _ = b.Buffer.Write(data[:min(len(data), (8<<20)-b.Len())])
	}
	return len(data), nil
}

func sameJSON(left, right []byte) bool {
	decode := func(data []byte) any {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if decoder.Decode(&value) != nil {
			return nil
		}
		return value
	}
	return reflect.DeepEqual(decode(left), decode(right))
}
