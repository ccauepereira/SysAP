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

	"github.com/ccauepereira/SysAP/apps/api/internal/academy"
	"github.com/ccauepereira/SysAP/apps/api/internal/identity"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/auth"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/config"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
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

	var databasePool *database.Pool
	var databaseChecker httpserver.DatabaseChecker = database.Unavailable{}
	if configuration.DatabaseURL == "" {
		logger.Info("database is not configured; readiness will remain unavailable")
	} else {
		pool, poolErr := database.NewPool(context.Background(), configuration.DatabaseURL)
		if poolErr != nil {
			logger.Warn("database configuration is invalid; readiness will remain unavailable")
		} else {
			databaseChecker = pool
			databasePool = pool
			defer pool.Close()
		}
	}

	var authMiddleware func(http.Handler) http.Handler
	if configuration.Auth.Configured() {
		tokenVerifier, err := auth.NewTokenVerifier(configuration.Auth)
		if err != nil {
			return fmt.Errorf("configure token verifier: %w", err)
		}
		sessionResolver, err := auth.NewPostgresSessionResolver(databasePool)
		if err != nil {
			return fmt.Errorf("configure session resolver: %w", err)
		}
		authMiddleware = auth.Middleware(tokenVerifier, sessionResolver)
	} else {
		logger.Warn("authentication is not configured; protected routes will be inaccessible")
	}

	meHandler := identity.NewMeHandler(databasePool, logger)

	var deliveryProvider identity.EnrollmentDeliveryProvider = nil
	var otpProvider identity.OTPProvider = nil
	brevoAPIKey := os.Getenv("SYSAP_EMAIL_PROVIDER_BREVO_KEY")
	brevoFrom := os.Getenv("SYSAP_EMAIL_PROVIDER_FROM")
	if brevoAPIKey != "" && brevoFrom != "" {
		brevoProvider, err := email.NewBrevoEmailProvider(brevoAPIKey, brevoFrom, "SysAP")
		if err != nil {
			logger.Error("failed to configure brevo email provider", "error", err)
		} else {
			sender, err := email.NewAllowlistedSender(brevoProvider, os.Getenv("SYSAP_EMAIL_TEST_ALLOWED_RECIPIENTS"))
			if err != nil {
				logger.Error("failed to configure allowed email recipients", "error", err)
			} else {
				deliveryProvider = identity.NewEmailEnrollmentDeliveryProvider(sender)
				otpProvider = identity.NewEmailOTPProvider(sender)
			}
		}
	} else if configuration.Environment == "development" || configuration.Environment == "local" || os.Getenv("SYSAP_SMTP_HOST") != "" {
		localSMTP, err := email.NewLocalSMTPEmailProvider(email.LocalSMTPConfig{
			Host:        os.Getenv("SYSAP_SMTP_HOST"),
			Port:        os.Getenv("SYSAP_SMTP_PORT"),
			SenderEmail: brevoFrom,
			SenderName:  "SysAP Artur Performance",
		})
		if err != nil {
			logger.Error("failed to configure local smtp email provider", "error", err)
		} else {
			deliveryProvider = identity.NewEmailEnrollmentDeliveryProvider(localSMTP)
			otpProvider = identity.NewEmailOTPProvider(localSMTP)
		}
	}

	invitationHandler := identity.NewInvitationHandlerWithDelivery(databasePool, logger, nil, nil, deliveryProvider)

	otpPepper := os.Getenv("SYSAP_OTP_PEPPER")
	loginRateSecret := os.Getenv("SYSAP_LOGIN_RATE_LIMIT_SECRET")
	recoveryPepper := os.Getenv("SYSAP_PASSWORD_RECOVERY_PEPPER")
	if configuration.Environment == "development" || configuration.Environment == "local" {
		if len(otpPepper) < 16 {
			otpPepper = "local-dev-only-otp-pepper-32bytes"
		}
		if len(loginRateSecret) < 16 {
			loginRateSecret = "local-dev-only-login-rate-32bytes"
		}
		if len(recoveryPepper) < 16 {
			recoveryPepper = "local-dev-only-recovery-pepper32"
		}
	}

	activationHandler := identity.NewActivationHandlerWithOTP(databasePool, otpPepper, otpProvider, logger)
	loginHandler := identity.NewLoginHandler(databasePool, loginRateSecret)
	sessionHandler := identity.NewSessionLifecycleHandler(databasePool)
	mfaHandler := identity.NewMFAHandler(databasePool)
	recoveryHandler := identity.NewPasswordRecoveryHandler(databasePool, recoveryPepper)

	trainingGroupHandler := academy.NewCreateTrainingGroupHandler(databasePool)
	handler := httpserver.New(databaseChecker, logger, configuration.DatabasePingTimeout, authMiddleware, meHandler, invitationHandler, activationHandler, loginHandler, sessionHandler, mfaHandler, recoveryHandler, trainingGroupHandler)
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
