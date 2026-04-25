package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Config struct {
	DatabaseURL           string
	IssuerURL             string
	FrontendURL           string
	WebAuthnRPID          string
	GoogleClientID        string
	GoogleClientSecret    string
	GoogleRedirectURL     string
	BootstrapAdminSub     string
	BootstrapAdminEmail   string
	AccessTokenTTLSeconds int
	RefreshTokenTTLHours  int
	SigningKeyID          string
	PrivateKeyPEM         string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		DatabaseURL:           envOr("ORIVIS_DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/orivis?sslmode=disable"),
		IssuerURL:             envOr("ORIVIS_ISSUER_URL", "http://localhost:8080"),
		FrontendURL:           envOr("ORIVIS_FRONTEND_URL", "http://localhost:5173"),
		WebAuthnRPID:          envOr("ORIVIS_WEBAUTHN_RPID", "localhost"),
		GoogleClientID:        strings.TrimSpace(os.Getenv("ORIVIS_GOOGLE_CLIENT_ID")),
		GoogleClientSecret:    strings.TrimSpace(os.Getenv("ORIVIS_GOOGLE_CLIENT_SECRET")),
		GoogleRedirectURL:     envOr("ORIVIS_GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/providers/google/callback"),
		BootstrapAdminSub:     strings.TrimSpace(os.Getenv("ORIVIS_BOOTSTRAP_ADMIN_SUB")),
		BootstrapAdminEmail:   strings.TrimSpace(os.Getenv("ORIVIS_BOOTSTRAP_ADMIN_EMAIL")),
		AccessTokenTTLSeconds: envOrInt("ORIVIS_ACCESS_TOKEN_TTL_SECONDS", 900),
		RefreshTokenTTLHours:  envOrInt("ORIVIS_REFRESH_TOKEN_TTL_HOURS", 720),
		SigningKeyID:          envOr("ORIVIS_SIGNING_KID", "orivis-dev-key"),
		PrivateKeyPEM:         strings.TrimSpace(os.Getenv("ORIVIS_JWT_PRIVATE_KEY_PEM")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("ORIVIS_DATABASE_URL must be set")
	}

	if cfg.AccessTokenTTLSeconds < 60 {
		return Config{}, errors.New("ORIVIS_ACCESS_TOKEN_TTL_SECONDS must be >= 60")
	}

	if cfg.RefreshTokenTTLHours < 1 {
		return Config{}, errors.New("ORIVIS_REFRESH_TOKEN_TTL_HOURS must be >= 1")
	}

	return cfg, nil
}

func (c Config) GoogleOAuthConfig() *oauth2.Config {
	if c.GoogleClientID == "" || c.GoogleClientSecret == "" {
		return nil
	}

	return &oauth2.Config{
		ClientID:     c.GoogleClientID,
		ClientSecret: c.GoogleClientSecret,
		RedirectURL:  c.GoogleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func (c Config) LoadOrCreatePrivateKey() (*rsa.PrivateKey, error) {
	if c.PrivateKeyPEM == "" {
		return rsa.GenerateKey(rand.Reader, 2048)
	}

	block, _ := pem.Decode([]byte(c.PrivateKeyPEM))
	if block == nil {
		return nil, errors.New("invalid ORIVIS_JWT_PRIVATE_KEY_PEM")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	key, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("ORIVIS_JWT_PRIVATE_KEY_PEM must be RSA")
	}

	return key, nil
}

func envOr(k, fallback string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	return v
}

func envOrInt(k string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return fallback
	}
	var n int
	_, _ = fmt.Sscanf(v, "%d", &n)
	if n == 0 {
		return fallback
	}
	return n
}
