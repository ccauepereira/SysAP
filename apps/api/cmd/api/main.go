package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/config"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/httpserver"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/logging"
)

func main() {
	logger := logging.New(os.Stdout)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	configuration, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	var databaseChecker httpserver.DatabaseChecker = database.Unavailable{}
	if configuration.DatabaseURL == "" {
		logger.Info("database is not configured; readiness will remain unavailable")
	} else {
		pool, poolErr := database.NewPool(context.Background(), configuration.DatabaseURL)
		if poolErr != nil {
			logger.Warn("database configuration is invalid; readiness will remain unavailable")
		} else {
			databaseChecker = pool
			defer pool.Close()
		}
	}

	handler := httpserver.New(databaseChecker, logger, configuration.DatabasePingTimeout)
	server := httpserver.NewServer(configuration.HTTPAddress, handler)
	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("api started",
			"environment", configuration.Environment,
			"http_address", configuration.HTTPAddress,
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case serverErr := <-serverErrors:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", serverErr)
		}
		return nil
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), configuration.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("api stopped gracefully")
	return nil
}
