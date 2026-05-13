package daemon

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const tokenBytes = 32

func GenerateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func AuthMiddleware(token string, next http.Handler) http.Handler {
	expected := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBearer(r, expected) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="remote-shell-mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkBearer(r *http.Request, expected []byte) bool {
	h := r.Header.Get("Authorization")
	const prefix = "bearer "
	if len(h) < len(prefix) {
		return false
	}
	// RFC 7235 § 2.1: auth-scheme matching is case-insensitive.
	if !strings.EqualFold(h[:len(prefix)], prefix) {
		return false
	}
	got := []byte(h[len(prefix):])
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(got, expected) == 1
}
