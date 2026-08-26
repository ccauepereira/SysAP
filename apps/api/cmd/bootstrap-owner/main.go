package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ccauepereira/SysAP/apps/api/internal/identity"
	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
)

func readInput(reader *bufio.Reader, prompt string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return text, nil
}

func readSecureInput(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	// Disable terminal echo
	cmd := exec.Command("stty", "-echo")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return "", err
	}
	defer func() {
		// Enable terminal echo
		cmd := exec.Command("stty", "echo")
		cmd.Stdin = os.Stdin
		_ = cmd.Run()
		fmt.Println()
	}()

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateEnvironment() error {
	env := os.Getenv("SYSAP_ENV")
	if env != "development" && env != "local" && env != "test" {
		return fmt.Errorf("invalid environment: must be development, local or test (got: %s)", env)
	}

	dbURL := os.Getenv("SYSAP_DATABASE_URL")
	if dbURL == "" {
		return errors.New("SYSAP_DATABASE_URL environment variable is required")
	}
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}
	if host := u.Hostname(); host != "" && !isLoopbackHost(host) {
		return fmt.Errorf("database host must be a loopback host (got: %s)", host)
	}

	authURL := os.Getenv("SYSAP_SUPABASE_AUTH_URL")
	if authURL == "" {
		return errors.New("SYSAP_SUPABASE_AUTH_URL environment variable is required")
	}
	uAuth, err := url.Parse(authURL)
	if err != nil {
		return fmt.Errorf("invalid Auth URL: %w", err)
	}
	if uAuth.Scheme != "https" {
		if uAuth.Scheme != "http" || !isLoopbackHost(uAuth.Hostname()) {
			return errors.New("Auth URL must use HTTPS outside local loopback development")
		}
	}

	roleKey := os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY")
	if roleKey == "" {
		return errors.New("SYSAP_SUPABASE_SERVICE_ROLE_KEY environment variable is required")
	}

	return nil
}

func testHTTPConnectivity(ctx context.Context, authURL string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		// Fallback to checking the root endpoint
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodGet, authURL, nil)
		if err2 != nil {
			return err2
		}
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return fmt.Errorf("failed HTTP connectivity check to %s: %w", authURL, err2)
		}
		defer resp2.Body.Close()
		return nil
	}
	defer resp.Body.Close()
	return nil
}

func main() {
	if err := validateEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "Security constraint failed: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialise DB pool
	pool, err := database.NewPool(ctx, os.Getenv("SYSAP_DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Check DB connectivity
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Database ping failed: %v\n", err)
		os.Exit(1)
	}

	// Check Auth URL connectivity
	if err := testHTTPConnectivity(ctx, os.Getenv("SYSAP_SUPABASE_AUTH_URL")); err != nil {
		fmt.Fprintf(os.Stderr, "Supabase local auth check failed: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("--- SysAP Owner Bootstrap ---")

	orgName, err := readInput(reader, "Nome da Organização", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	timezone, err := readInput(reader, "Timezone", "America/Fortaleza")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	fullName, err := readInput(reader, "Nome completo do Owner", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	email, err := readInput(reader, "E-mail", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	phone, err := readInput(reader, "Telefone E.164 (ex. +5585999999999)", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	password, err := readSecureInput("Senha")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	confirmPassword, err := readSecureInput("Confirmação de Senha")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read input: %v\n", err)
		os.Exit(1)
	}

	if password != confirmPassword {
		fmt.Fprintln(os.Stderr, "Validation failed: passwords do not match")
		os.Exit(1)
	}

	params := identity.BootstrapOwnerParams{
		OrgName:  orgName,
		Timezone: timezone,
		FullName: fullName,
		Email:    email,
		Phone:    phone,
		Password: password,
	}

	authAdmin := identity.NewSupabaseAuthAdmin()
	generator := identity.NewEnrollmentGenerator(time.Now)

	enrollment, err := identity.BootstrapOwner(ctx, pool, authAdmin, generator, params, time.Now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("Owner Bootstrap concluído com sucesso!")
	fmt.Printf("Matrícula do Owner: %s\n", enrollment)
	fmt.Printf("Organização criada:  %s\n", orgName)
	fmt.Println("Instrução: abra /login no painel para efetuar o login.")
	fmt.Println("==================================================")
}
