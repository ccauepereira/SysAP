package config

import (
	"fmt"
	"net"
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
)

type Config struct {
	Environment         string
	HTTPAddress         string
	DatabaseURL         string
	ShutdownTimeout     time.Duration
	DatabasePingTimeout time.Duration
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

	httpAddress := stringFromEnvironment("SYSAP_HTTP_ADDR", defaultHTTPAddress)
	if err := validateHTTPAddress(httpAddress); err != nil {
		return Config{}, fmt.Errorf("invalid SYSAP_HTTP_ADDR: %w", err)
	}

	return Config{
		Environment:         stringFromEnvironment("SYSAP_ENV", defaultEnvironment),
		HTTPAddress:         httpAddress,
		DatabaseURL:         os.Getenv("SYSAP_DATABASE_URL"),
		ShutdownTimeout:     shutdownTimeout,
		DatabasePingTimeout: databasePingTimeout,
	}, nil
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
