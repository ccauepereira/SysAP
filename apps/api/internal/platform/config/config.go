package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEnvironment         = "development"
	defaultHTTPAddress         = ":8080"
	defaultShutdownTimeout     = 10 * time.Second
	defaultDatabasePingTimeout = 2 * time.Second
	defaultJWKSQueryTimeout    = 2 * time.Second
	defaultJWKSCacheTTL        = 5 * time.Minute
	defaultJWKSMaxBodyLength   = 64 * 1024
	defaultJWKSMaxKeys         = 16
)

type Config struct {
	Environment         string
	HTTPAddress         string
	DatabaseURL         string
	ShutdownTimeout     time.Duration
	DatabasePingTimeout time.Duration
	Auth                AuthConfig
}

// AuthConfig is deliberately server-side only. It remains optional until the
// first protected route is registered, but a partially supplied configuration
// is rejected during process startup.
type AuthConfig struct {
	Issuer            string
	Audience          string
	JWKSURL           string
	JWKSQueryTimeout  time.Duration
	JWKSCacheTTL      time.Duration
	JWKSMaxBodyLength int64
	JWKSMaxKeys       int
}

func (c AuthConfig) Configured() bool {
	return c.Issuer != "" || c.Audience != "" || c.JWKSURL != ""
}

func Load() (Config, error) {
	shutdownTimeout, err := durationFromEnvironment(
		"SYSAP_SHUTDOWN_TIMEOUT",
		defaultShutdownTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	databasePingTimeout, err := durationFromEnvironment(
		"SYSAP_DATABASE_PING_TIMEOUT",
		defaultDatabasePingTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	environment := stringFromEnvironment("SYSAP_ENV", defaultEnvironment)
	auth, err := authConfigFromEnvironment(environment)
	if err != nil {
		return Config{}, err
	}

	httpAddress := stringFromEnvironment("SYSAP_HTTP_ADDR", defaultHTTPAddress)
	if err := validateHTTPAddress(httpAddress); err != nil {
		return Config{}, fmt.Errorf("invalid SYSAP_HTTP_ADDR: %w", err)
	}

	return Config{
		Environment:         environment,
		HTTPAddress:         httpAddress,
		DatabaseURL:         os.Getenv("SYSAP_DATABASE_URL"),
		ShutdownTimeout:     shutdownTimeout,
		DatabasePingTimeout: databasePingTimeout,
		Auth:                auth,
	}, nil
}

func authConfigFromEnvironment(environment string) (AuthConfig, error) {
	issuer := os.Getenv("SYSAP_AUTH_JWT_ISSUER")
	audience := os.Getenv("SYSAP_AUTH_JWT_AUDIENCE")
	jwksURL := os.Getenv("SYSAP_AUTH_JWKS_URL")

	if issuer == "" && audience == "" && jwksURL == "" {
		return AuthConfig{}, nil
	}
	if issuer == "" || audience == "" || jwksURL == "" {
		return AuthConfig{}, fmt.Errorf("SYSAP_AUTH_JWT_ISSUER, SYSAP_AUTH_JWT_AUDIENCE, and SYSAP_AUTH_JWKS_URL must be configured together")
	}
	if issuer != strings.TrimSpace(issuer) || audience != strings.TrimSpace(audience) || jwksURL != strings.TrimSpace(jwksURL) {
		return AuthConfig{}, fmt.Errorf("authentication configuration must not contain surrounding whitespace")
	}

	timeout, err := durationFromEnvironment("SYSAP_AUTH_JWKS_TIMEOUT", defaultJWKSQueryTimeout)
	if err != nil {
		return AuthConfig{}, err
	}

	cacheTTL, err := durationFromEnvironment("SYSAP_AUTH_JWKS_CACHE_TTL", defaultJWKSCacheTTL)
	if err != nil {
		return AuthConfig{}, err
	}

	maxBodyLength, err := intFromEnvironment("SYSAP_AUTH_JWKS_MAX_BODY_LENGTH", defaultJWKSMaxBodyLength)
	if err != nil || maxBodyLength <= 0 {
		return AuthConfig{}, fmt.Errorf("SYSAP_AUTH_JWKS_MAX_BODY_LENGTH must be greater than zero")
	}

	maxKeys, err := intFromEnvironment("SYSAP_AUTH_JWKS_MAX_KEYS", defaultJWKSMaxKeys)
	if err != nil || maxKeys <= 0 {
		return AuthConfig{}, fmt.Errorf("SYSAP_AUTH_JWKS_MAX_KEYS must be greater than zero")
	}

	parsedURL, err := url.Parse(jwksURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return AuthConfig{}, fmt.Errorf("SYSAP_AUTH_JWKS_URL must be an absolute URL without credentials, query, or fragment")
	}
	if parsedURL.Scheme != "https" && !(isLocalEnvironment(environment) && parsedURL.Scheme == "http" && isLoopbackHost(parsedURL.Hostname())) {
		return AuthConfig{}, fmt.Errorf("SYSAP_AUTH_JWKS_URL must use HTTPS outside local loopback development")
	}

	return AuthConfig{
		Issuer:            issuer,
		Audience:          audience,
		JWKSURL:           parsedURL.String(),
		JWKSQueryTimeout:  timeout,
		JWKSCacheTTL:      cacheTTL,
		JWKSMaxBodyLength: int64(maxBodyLength),
		JWKSMaxKeys:       maxKeys,
	}, nil
}

func isLocalEnvironment(environment string) bool {
	switch environment {
	case "development", "local", "test":
		return true
	default:
		return false
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func durationFromEnvironment(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration", name)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return value, nil
}

func intFromEnvironment(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", name)
	}

	return value, nil
}

func validateHTTPAddress(address string) error {
	if address != strings.TrimSpace(address) {
		return fmt.Errorf("must not contain surrounding whitespace")
	}

	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("must use host:port format")
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func stringFromEnvironment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
