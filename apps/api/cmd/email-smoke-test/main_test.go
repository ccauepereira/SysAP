package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/email"
)

type mockProvider struct {
	sentMsg email.TransactionalEmail
}

func (m *mockProvider) Send(ctx context.Context, message email.TransactionalEmail) error {
	m.sentMsg = message
	return nil
}

func TestRunCommandOutput(t *testing.T) {
	envMap := map[string]string{
		"SYSAP_EMAIL_PROVIDER":      "brevo_api",
		"SYSAP_EMAIL_SMOKE_TEST_TO": "test@example.com",
		"SYSAP_BREVO_API_KEY":       "fake-key",
		"SYSAP_EMAIL_FROM":          "from@example.com",
		"SYSAP_EMAIL_FROM_NAME":     "Test Sender",
	}

	getenv := func(key string) string {
		return envMap[key]
	}

	mockP := &mockProvider{}
	newProvider := func(apiKey, from, fromName string) (email.TransactionalEmailSender, error) {
		return mockP, nil
	}

	var stdout bytes.Buffer
	err := run(getenv, &stdout, newProvider)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	outStr := stdout.String()
	expectedOutput := "E-mail de teste solicitado. Verifique sua caixa de entrada.\n"
	if outStr != expectedOutput {
		t.Errorf("expected %q, got %q", expectedOutput, outStr)
	}

	// Verify that output does not contain the generated code or recipient
	// Since the code is dynamic, we just check that the output matches EXACTLY the expected string
	// But let's be sure the generated code was in the sent message
	if !strings.Contains(mockP.sentMsg.Text, "Seu código é:") {
		t.Errorf("sent message missing code in text: %s", mockP.sentMsg.Text)
	}
	if !strings.Contains(mockP.sentMsg.HTML, "Seu código é:") {
		t.Errorf("sent message missing code in html: %s", mockP.sentMsg.HTML)
	}
	if mockP.sentMsg.To != "test@example.com" {
		t.Errorf("expected sent msg To 'test@example.com', got %q", mockP.sentMsg.To)
	}
}

func TestGenerateOTP(t *testing.T) {
	for i := 0; i < 100; i++ {
		code, err := generateOTP()
		if err != nil {
			t.Fatalf("generateOTP erro: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6 digits, got %d for code %q", len(code), code)
		}
		for _, char := range code {
			if char < '0' || char > '9' {
				t.Errorf("expected numeric characters only, got %q", code)
			}
		}
	}
}
