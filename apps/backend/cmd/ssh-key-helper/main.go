package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func main() {
	secret, err := os.ReadFile(os.Getenv("SSH_HELPER_TOKEN_FILE"))
	if err != nil || len(strings.TrimSpace(string(secret))) < 32 {
		log.Fatal("SSH helper token is unavailable")
	}
	secret = []byte(strings.TrimSpace(string(secret)))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /install", func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), secret) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input struct {
			PublicKey string `json:"public_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		key, _, _, _, err := ssh.ParseAuthorizedKey([]byte(input.PublicKey))
		if err != nil || key.Type() != ssh.KeyAlgoED25519 {
			http.Error(w, "only an Ed25519 key is accepted", http.StatusBadRequest)
			return
		}
		hostKey, err := os.ReadFile("/host-ssh-hostkey.pub")
		if err != nil {
			http.Error(w, "host SSH public key is unavailable", http.StatusInternalServerError)
			return
		}
		if _, _, _, _, err := ssh.ParseAuthorizedKey(hostKey); err != nil {
			http.Error(w, "invalid host SSH public key", http.StatusInternalServerError)
			return
		}
		if err := installKey(ssh.MarshalAuthorizedKey(key)); err != nil {
			http.Error(w, "could not install SSH key", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"host_key": strings.TrimSpace(string(hostKey))})
	})
	log.Fatal(http.ListenAndServe(":8071", mux))
}

func installKey(key []byte) error {
	wanted, _, _, _, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return err
	}
	path := "/host-authorized-keys"
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("invalid authorized_keys file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for len(data) > 0 {
		parsed, _, _, rest, parseErr := ssh.ParseAuthorizedKey(data)
		if parseErr != nil {
			break
		}
		if bytes.Equal(parsed.Marshal(), wanted.Marshal()) {
			return nil
		}
		data = rest
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return err
	}
	if _, err := file.Write(key); err != nil {
		return err
	}
	if _, err := io.WriteString(file, "\n"); err != nil {
		return err
	}
	return file.Sync()
}
