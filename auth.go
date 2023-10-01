package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type AuthConfig struct {
	JwtSecretKey    string   `json:"jwtSecretKey,omitempty"`
	JwtSecretKeyEnv string   `json:"jwtSecretKeyEnv,omitempty"`
	RolePath        []string `json:"rolePath,omitempty"`
}

func (c *AuthConfig) defaults() {
	if c.JwtSecretKey == "" {
		c.JwtSecretKey = os.Getenv(c.JwtSecretKeyEnv)
	}
}

type Auth struct {
	jwtSecretKey []byte
	rolePath     []string
}

func NewAuth(config AuthConfig) *Auth {
	config.defaults()
	return &Auth{
		jwtSecretKey: []byte(config.JwtSecretKey),
		rolePath:     config.RolePath,
	}
}

func (a *Auth) Middleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		token := r.Header.Get("Authorization")

		claims, err := a.getClaims(token)
		if err != nil || (a.rolePath != nil && !IsJsonPath(claims, a.rolePath)) {
			LogHttpErr(w, id, "Unauthorized", err, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Auth) getClaims(token string) (map[string]any, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(a.jwtSecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if !t.Valid {
		return nil, fmt.Errorf("invalid jwt claim")
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid jwt claim")
	}
	return claims, nil
}
