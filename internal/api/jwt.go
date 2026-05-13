package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const userIDKey contextKey = "user_id"

type Claims struct {
	UserID string   `json:"user_id"`
	Scopes []string `json:"scopes"`
	Exp    int64    `json:"exp"`
}

func SignHS256(secret string, claims Claims) (string, error) {
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(time.Hour).Unix()
	}
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	h, _ := json.Marshal(header)
	p, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := b64(h) + "." + b64(p)
	sig := mac(secret, unsigned)
	return unsigned + "." + b64(sig), nil
}

func VerifyHS256(secret, token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	unsigned := parts[0] + "." + parts[1]
	expected := b64(mac(secret, unsigned))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, err
	}
	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return Claims{}, errors.New("token expired")
	}
	if claims.UserID == "" {
		return Claims{}, errors.New("missing user_id")
	}
	return claims, nil
}

func mac(secret, msg string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write([]byte(msg))
	return h.Sum(nil)
}
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func UserID(ctx context.Context) string { v, _ := ctx.Value(userIDKey).(string); return v }

func AuthMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		claims, err := VerifyHS256(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			http.Error(w, "invalid bearer token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, claims.UserID)))
	})
}
