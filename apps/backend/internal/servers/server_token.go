package servers

import (
	"context"
	"crypto/sha256"
	"os"
	"strings"
)

func writeTokenFile(path, token string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token), 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func tokenFileDigest(path string) ([32]byte, bool) {
	data, err := os.ReadFile(path)
	token := strings.TrimSpace(string(data))
	if err != nil || token == "" {
		return [32]byte{}, false
	}
	return sha256.Sum256([]byte(token)), true
}

func isLocalServer(tokenHash []byte) bool {
	digest, ok := tokenFileDigest(os.Getenv("KYMBER_SERVER_TOKEN_FILE"))
	return ok && string(digest[:]) == string(tokenHash)
}

func (h *Handler) claimLocalToken(ctx context.Context, token string) error {
	path := os.Getenv("KYMBER_SERVER_TOKEN_FILE")
	if path == "" {
		return nil
	}
	if digest, ok := tokenFileDigest(path); ok {
		var taken bool
		if err := h.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM servers WHERE token_hash = ?)", digest[:]).Scan(&taken); err != nil {
			return err
		}
		if taken {
			return nil
		}
	}
	return writeTokenFile(path, token)
}
