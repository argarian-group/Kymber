package monitoring

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/sshclient"
)

type sshPool struct {
	mu      sync.Mutex
	clients map[string]*ssh.Client
}

var connections = &sshPool{clients: make(map[string]*ssh.Client)}

func (p *sshPool) dial(ctx context.Context, db *sql.DB, serverID, address string) (net.Conn, error) {
	p.mu.Lock()
	client := p.clients[serverID]
	p.mu.Unlock()
	if client != nil {
		conn, err := client.DialContext(ctx, "tcp", address)
		var refused *ssh.OpenChannelError
		if err == nil || errors.As(err, &refused) || ctx.Err() != nil {
			return conn, err
		}
		p.forget(serverID, client)
	}
	client, err := sshclient.Connect(ctx, db, serverID)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	if current := p.clients[serverID]; current != nil {
		p.mu.Unlock()
		client.Close()
		return current.DialContext(ctx, "tcp", address)
	}
	p.clients[serverID] = client
	p.mu.Unlock()
	return client.DialContext(ctx, "tcp", address)
}

func (p *sshPool) forget(serverID string, only *ssh.Client) {
	p.mu.Lock()
	client := p.clients[serverID]
	if client != nil && (only == nil || client == only) {
		delete(p.clients, serverID)
	}
	p.mu.Unlock()
	if client != nil && (only == nil || client == only) {
		client.Close()
	}
}
