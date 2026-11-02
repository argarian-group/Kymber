package sshclient

import (
	"context"
	"database/sql"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"kymber/backend/internal/secrets"
)

func Connect(ctx context.Context, db *sql.DB, serverID string) (*ssh.Client, error) {
	var host, username string
	var port int
	var encryptedKey, encodedHostKey []byte
	err := db.QueryRowContext(ctx, "SELECT host, ssh_port, ssh_user, encrypted_ssh_key, ssh_host_key FROM servers WHERE id = ? AND encrypted_ssh_key IS NOT NULL", serverID).Scan(&host, &port, &username, &encryptedKey, &encodedHostKey)
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

func Run(ctx context.Context, db *sql.DB, serverID, command string) ([]byte, error) {
	connection, err := Connect(ctx, db, serverID)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	session, err := connection.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	return session.CombinedOutput(command)
}
