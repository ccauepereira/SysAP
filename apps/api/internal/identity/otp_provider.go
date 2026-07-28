package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type OTPProvider interface {
	IsExternal() bool
	NewCode() (string, error)
	Start(ctx context.Context, phone string) error
	Verify(ctx context.Context, phone, code string) error
}

type localOTPProvider struct {
	hook func() (string, error)
}

func (p localOTPProvider) IsExternal() bool { return false }
func (p localOTPProvider) Start(ctx context.Context, phone string) error { return nil }
func (p localOTPProvider) Verify(ctx context.Context, phone, code string) error { return nil }
func (p localOTPProvider) NewCode() (string, error) {
	if p.hook != nil {
		return p.hook()
	}
	var b [6]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b[:]), nil
}

type twilioOTPProvider struct {
	accountSID  string
	authToken   string
	serviceSID  string
	httpClient  *http.Client
}

func NewTwilioOTPProvider() (OTPProvider, error) {
	sid := os.Getenv("SYSAP_TWILIO_ACCOUNT_SID")
	token := os.Getenv("SYSAP_TWILIO_AUTH_TOKEN")
	service := os.Getenv("SYSAP_TWILIO_VERIFY_SERVICE_SID")
	
	if sid == "" || token == "" || service == "" {
		return nil, errors.New("missing twilio credentials")
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &twilioOTPProvider{
		accountSID: sid,
		authToken:  token,
		serviceSID: service,
		httpClient: client,
	}, nil
}

func (p *twilioOTPProvider) IsExternal() bool { return true }
func (p *twilioOTPProvider) NewCode() (string, error) {
	// Dummy code to satisfy DB not-null constraint on otp_hmac
	var b [6]byte
	rand.Read(b[:])
	return string(b[:]), nil
}

func (p *twilioOTPProvider) Start(ctx context.Context, phone string) error {
	url := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/Verifications", p.serviceSID)
	
	data := "To=" + phone + "&Channel=sms"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return errors.New("provider_error")
	}
	defer resp.Body.Close()
	io.CopyN(io.Discard, resp.Body, 1024)

	if resp.StatusCode != http.StatusCreated {
		return errors.New("provider_error")
	}
	return nil
}

func (p *twilioOTPProvider) Verify(ctx context.Context, phone, code string) error {
	url := fmt.Sprintf("https://verify.twilio.com/v2/Services/%s/VerificationCheck", p.serviceSID)
	
	data := "To=" + phone + "&Code=" + code
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.accountSID, p.authToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return errors.New("provider_error")
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode != http.StatusOK {
		return errors.New("invalid_code")
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.New("provider_error")
	}

	if result.Status != "approved" {
		return errors.New("invalid_code")
	}

	return nil
}
