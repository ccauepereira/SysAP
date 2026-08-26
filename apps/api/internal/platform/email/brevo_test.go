package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewBrevoEmailProvider(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		from     string
		fromName string
		wantErr  bool
	}{
		{"configuração válida", "valid-key", "sender@test.com", "Test Sender", false},
		{"chave ausente", "", "sender@test.com", "Test Sender", true},
		{"chave em branco", "   ", "sender@test.com", "Test Sender", true},
		{"remetente ausente", "valid-key", "", "Test Sender", true},
		{"remetente inválido (sem @)", "valid-key", "sender.test.com", "Test Sender", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewBrevoEmailProvider(tt.apiKey, tt.from, tt.fromName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBrevoEmailProvider() erro esperado, mas não ocorreu")
				}
				if provider != nil {
					t.Errorf("NewBrevoEmailProvider() provider deveria ser nil em caso de erro")
				}
			} else {
				if err != nil {
					t.Errorf("NewBrevoEmailProvider() erro inesperado: %v", err)
				}
				if provider == nil {
					t.Errorf("NewBrevoEmailProvider() provider não deveria ser nil")
				}
			}
		})
	}
}

func TestBrevoEmailProvider_Send(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		message        TransactionalEmail
		wantErr        bool
		errContains    []string
		errNotContains []string
	}{
		{
			name: "envio bem-sucedido com resposta Brevo válida",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"messageId":"<test-id>"}`))
			},
			message: TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr: false,
		},
		{
			name: "timeout em até 3 segundos",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(3100 * time.Millisecond)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"messageId":"<test-id>"}`))
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key"},
		},
		{
			name: "redirect rejeitado",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/other")
				w.WriteHeader(http.StatusFound)
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key"},
		},
		{
			name: "HTTP 4xx",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"bad request"}`))
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key", "bad request"},
		},
		{
			name: "HTTP 5xx",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal error"}`))
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key", "internal error"},
		},
		{
			name: "Content-Type inválido",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`<html><body>Success</body></html>`))
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key"},
		},
		{
			name: "JSON inválido",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{invalid-json}`))
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key", "{invalid-json}"},
		},
		{
			name: "corpo vazio",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key"},
		},
		{
			name: "corpo maior que 64 KiB",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				largeBody := make([]byte, 65*1024)
				for i := range largeBody {
					largeBody[i] = 'a'
				}
				w.Write(largeBody)
			},
			message:        TransactionalEmail{To: "dest@test.com", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"dest@test.com", "Hello", "Test", "valid-key"},
		},
		{
			name:           "destinatário inválido",
			handler:        func(w http.ResponseWriter, r *http.Request) {},
			message:        TransactionalEmail{To: "invalid-dest", Subject: "Test", Text: "Hello"},
			wantErr:        true,
			errNotContains: []string{"invalid-dest", "Hello", "Test", "valid-key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			provider, err := newBrevoEmailProvider("valid-key", "sender@test.com", "Test Sender", server.URL)
			if err != nil {
				t.Fatalf("Erro inesperado ao criar provider: %v", err)
			}

			err = provider.Send(context.Background(), tt.message)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Send() erro esperado, mas não ocorreu")
				} else {
					errStr := err.Error()
					for _, contains := range tt.errContains {
						if !strings.Contains(errStr, contains) {
							t.Errorf("Send() erro %q deveria conter %q", errStr, contains)
						}
					}
					for _, notContains := range tt.errNotContains {
						if strings.Contains(errStr, notContains) {
							t.Errorf("Send() erro %q não deveria conter %q", errStr, notContains)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Send() erro inesperado: %v", err)
				}
			}
		})
	}
}
