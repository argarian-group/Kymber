package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"net/http"
	"time"
)

const sessionCookie = "kymber_session"

var sessionLifetime = 7 * 24 * time.Hour

func createSession(r *http.Request, tx *sql.Tx, userID uint64) (string, time.Time, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	digest := sha256.Sum256(token)
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO user_sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)", digest[:], userID, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return base64.RawURLEncoding.EncodeToString(token), expiresAt, nil
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", Expires: expiresAt, MaxAge: int(time.Until(expiresAt).Seconds()), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
}
