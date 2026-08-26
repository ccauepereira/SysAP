package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"html"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
)

func main() {
	if err := run(os.Getenv, os.Stdout, func(apiKey, from, fromName string) (email.TransactionalEmailSender, error) {
		return email.NewBrevoEmailProvider(apiKey, from, fromName)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Falha segura: %v\n", err)
		os.Exit(1)
	}
}

func run(getenv func(string) string, stdout io.Writer, newProvider func(string, string, string) (email.TransactionalEmailSender, error)) error {
	providerEnv := strings.TrimSpace(getenv("SYSAP_EMAIL_PROVIDER"))
	if providerEnv != "brevo_api" {
		return fmt.Errorf("provedor invalido ou nao configurado para teste local")
	}

	to := strings.TrimSpace(getenv("SYSAP_EMAIL_SMOKE_TEST_TO"))
	if to == "" || !strings.Contains(to, "@") {
		return fmt.Errorf("destinatario de teste ausente ou invalido")
	}

	apiKey := strings.TrimSpace(getenv("SYSAP_BREVO_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("chave de api ausente")
	}

	from := strings.TrimSpace(getenv("SYSAP_EMAIL_FROM"))
	if from == "" || !strings.Contains(from, "@") {
		return fmt.Errorf("remetente ausente")
	}

	fromName := strings.TrimSpace(getenv("SYSAP_EMAIL_FROM_NAME"))
	if fromName == "" {
		fromName = "Artur Performance"
	}

	provider, err := newProvider(apiKey, from, fromName)
	if err != nil {
		return fmt.Errorf("erro ao configurar provedor")
	}

	code, err := generateOTP()
	if err != nil {
		return fmt.Errorf("falha ao gerar codigo seguro")
	}

	htmlContent := fmt.Sprintf(`
		<h1>Teste Local SysAP</h1>
		<p>Este é um teste local gerado pelo comando de smoke test.</p>
		<p>Seu código é: <strong>%s</strong></p>
	`, html.EscapeString(code))

	msg := email.TransactionalEmail{
		To:      to,
		Subject: "SysAP — teste de entrega",
		Text:    fmt.Sprintf("Teste Local SysAP\n\nEste é um teste local gerado pelo comando de smoke test.\nSeu código é: %s", code),
		HTML:    htmlContent,
	}

	ctx := context.Background()
	if err := provider.Send(ctx, msg); err != nil {
		return fmt.Errorf("falha ao enviar mensagem de teste")
	}

	fmt.Fprintln(stdout, "E-mail de teste solicitado. Verifique sua caixa de entrada.")
	return nil
}

func generateOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
