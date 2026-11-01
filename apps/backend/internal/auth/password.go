package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	passwordMemory      = 64 * 1024
	passwordIterations  = 3
	passwordParallelism = 4
	passwordKeyLength   = 32
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemory, passwordParallelism, passwordKeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", passwordMemory, passwordIterations, passwordParallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=3,p=4" {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) != passwordKeyLength {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemory, passwordParallelism, passwordKeyLength)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func validatePassword(password string) error {
	length := utf8.RuneCountInString(password)
	if length < 8 || length > 128 || len(password) > 256 {
		return fmt.Errorf("password must be 8 to 128 characters")
	}
	return nil
}
