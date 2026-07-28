package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	authAdminTimeout  = 3 * time.Second
	authAdminMaxBody  = 64 * 1024
	authAdminEndpoint = "admin/users"
)

var errAuthUnavailable = errors.New("auth admin unavailable")

type SupabaseAuthAdmin interface {
	CreateUser(ctx context.Context, phone, password string) (uuid.UUID, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type unavailableAuthAdmin struct{}

func (unavailableAuthAdmin) CreateUser(context.Context, string, string) (uuid.UUID, error) {
	return uuid.Nil, errAuthUnavailable
}
func (unavailableAuthAdmin) DeleteUser(context.Context, uuid.UUID) error { return errAuthUnavailable }

type supabaseAuthAdmin struct {
	baseURL *url.URL
	roleKey string
	client  *http.Client
}

// NewSupabaseAuthAdmin intentionally accepts no URL from an HTTP request. The
// Auth endpoint is server configuration and remains unavailable when omitted.
func NewSupabaseAuthAdmin() SupabaseAuthAdmin {
	baseURL, roleKey := os.Getenv("SYSAP_SUPABASE_AUTH_URL"), os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY")
	if baseURL == "" || roleKey == "" {
		return unavailableAuthAdmin{}
	}
	admin, err := newSupabaseAuthAdmin(baseURL, roleKey, os.Getenv("SYSAP_ENV"), nil)
	if err != nil {
		return unavailableAuthAdmin{}
	}
	return admin
}

func newSupabaseAuthAdmin(baseURL, roleKey, environment string, client *http.Client) (*supabaseAuthAdmin, error) {
	if roleKey == "" {
		return nil, errAuthUnavailable
	}
	parsed, err := validatedAuthURL(baseURL, environment)
	if err != nil {
		return nil, errAuthUnavailable
	}
	if client == nil {
		client = &http.Client{Timeout: authAdminTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &supabaseAuthAdmin{baseURL: parsed, roleKey: roleKey, client: client}, nil
}

func validatedAuthURL(rawURL, environment string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errAuthUnavailable
	}
	if parsed.Scheme == "https" {
		return parsed, nil
	}
	if parsed.Scheme != "http" || !isLocalEnvironment(environment) || !isLoopbackHost(parsed.Hostname()) {
		return nil, errAuthUnavailable
	}
	return parsed, nil
}

func isLocalEnvironment(environment string) bool {
	return environment == "development" || environment == "test" || environment == "local"
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *supabaseAuthAdmin) CreateUser(ctx context.Context, phone, password string) (uuid.UUID, error) {
	payload, err := json.Marshal(struct {
		Phone        string `json:"phone"`
		Password     string `json:"password"`
		PhoneConfirm bool   `json:"phone_confirm"`
	}{Phone: phone, Password: password, PhoneConfirm: true})
	if err != nil {
		return uuid.Nil, errAuthUnavailable
	}
	var response struct {
		ID uuid.UUID `json:"id"`
	}
	if err := s.doJSON(ctx, http.MethodPost, authAdminEndpoint, bytes.NewReader(payload), &response); err != nil || response.ID == uuid.Nil {
		return uuid.Nil, errAuthUnavailable
	}
	return response.ID, nil
}

func (s *supabaseAuthAdmin) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errAuthUnavailable
	}
	return s.doJSON(ctx, http.MethodDelete, authAdminEndpoint+"/"+id.String(), nil, nil)
}

func (s *supabaseAuthAdmin) doJSON(ctx context.Context, method, path string, body io.Reader, target any) error {
	endpoint := s.baseURL.JoinPath(path)
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return errAuthUnavailable
	}
	req.Header.Set("apikey", s.roleKey)
	req.Header.Set("Authorization", "Bearer "+s.roleKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return errAuthUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errAuthUnavailable
	}
	if target == nil {
		return discardBounded(resp.Body)
	}
	bodyBytes, err := readBounded(resp.Body)
	if err != nil {
		return errAuthUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errAuthUnavailable
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errAuthUnavailable
	}
	return nil
}

func readBounded(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, authAdminMaxBody+1))
	if err != nil || len(body) > authAdminMaxBody {
		return nil, errAuthUnavailable
	}
	return body, nil
}

func discardBounded(reader io.Reader) error {
	_, err := readBounded(reader)
	return err
}
