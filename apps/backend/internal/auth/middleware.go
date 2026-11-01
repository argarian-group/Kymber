package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
)

type contextKey struct{}

var errUnauthenticated = errors.New("authentication required")

func Authenticate(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := userFromSession(db, r)
		if errors.Is(err, errUnauthenticated) {
			writeError(w, http.StatusUnauthorized, errUnauthenticated.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not verify session")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextKey{}, user)))
	})
}

func UserFromContext(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(contextKey{}).(User)
	return user, ok
}

func userFromSession(db *sql.DB, r *http.Request) (User, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return User{}, errUnauthenticated
	}
	token, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil || len(token) != 32 {
		return User{}, errUnauthenticated
	}
	digest := sha256.Sum256(token)
	var user User
	err = db.QueryRowContext(r.Context(), `SELECT u.id, u.username, u.email, u.is_root FROM users u JOIN user_sessions s ON s.user_id = u.id WHERE s.token_hash = ? AND s.expires_at > UTC_TIMESTAMP(6)`, digest[:]).Scan(&user.ID, &user.Username, &user.Email, &user.IsRoot)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, errUnauthenticated
	}
	return user, err
}
