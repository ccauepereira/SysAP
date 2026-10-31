package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxResponseSize = 64 * 1024
	brevoEndpoint   = "https://api.brevo.com/v3/smtp/email"
)

// ClassifiedSendError exposes only whether delivery is definitely rejected or
// ambiguous; provider response bodies and credentials are never retained.
type ClassifiedSendError struct {
	message string
	outcome string
}

func (e ClassifiedSendError) Error() string           { return e.message }
func (e ClassifiedSendError) DeliveryOutcome() string { return e.outcome }

func sendFailure(message, outcome string) error {
	return ClassifiedSendError{message: message, outcome: outcome}
}

type BrevoEmailProvider struct {
	apiKey   string
	from     string
	fromName string
	endpoint string
	client   *http.Client
}

type BrevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BrevoRecipient struct {
	Email string `json:"email"`
}

type BrevoPayload struct {
	Sender      BrevoSender      `json:"sender"`
	To          []BrevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	TextContent string           `json:"textContent"`
	HtmlContent string           `json:"htmlContent,omitempty"`
}

func NewBrevoEmailProvider(apiKey, from, fromName string) (*BrevoEmailProvider, error) {
	return newBrevoEmailProvider(apiKey, from, fromName, brevoEndpoint)
}

func newBrevoEmailProvider(apiKey, from, fromName, endpoint string) (*BrevoEmailProvider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("email: provider requer chave de API")
	}
	if strings.TrimSpace(from) == "" || !strings.Contains(from, "@") {
		return nil, errors.New("email: provider requer remetente valido")
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &BrevoEmailProvider{
		apiKey:   apiKey,
		from:     from,
		fromName: fromName,
		endpoint: endpoint,
		client:   client,
	}, nil
}

func (p *BrevoEmailProvider) Send(ctx context.Context, message TransactionalEmail) error {
	if strings.TrimSpace(message.To) == "" || !strings.Contains(message.To, "@") {
		return sendFailure("email: destinatario invalido", "failed")
	}

	payload := BrevoPayload{
		Sender: BrevoSender{
			Name:  p.fromName,
			Email: p.from,
		},
		To: []BrevoRecipient{
			{Email: message.To},
		},
		Subject:     message.Subject,
		TextContent: message.Text,
		HtmlContent: message.HTML,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return sendFailure("email: erro interno ao preparar mensagem", "failed")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return sendFailure("email: erro interno ao criar requisicao", "failed")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return sendFailure("email: falha na comunicacao com provedor", "unknown")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode <= 399 {
		return sendFailure("email: provedor retornou redirecionamento inesperado", "unknown")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusRequestTimeout || resp.StatusCode == http.StatusConflict {
			return sendFailure("email: provedor retornou resultado ambiguo", "unknown")
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return sendFailure("email: provedor rejeitou o envio", "failed")
		}
		return sendFailure("email: provedor rejeitou o envio", "unknown")
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return sendFailure("email: provedor retornou tipo de conteudo inesperado", "unknown")
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return sendFailure("email: falha ao ler resposta do provedor", "unknown")
	}

	if len(respBody) > maxResponseSize {
		return sendFailure("email: resposta do provedor excede limite de tamanho", "unknown")
	}

	if len(respBody) == 0 {
		return sendFailure("email: provedor retornou resposta vazia", "unknown")
	}

	var jsonResp map[string]interface{}
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return sendFailure("email: provedor retornou json invalido", "unknown")
	}

	return nil
}
