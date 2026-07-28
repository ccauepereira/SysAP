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

type supabasePasswordIdentityProvider struct {
	auth *supabaseAuthAdmin
}

// NewSupabasePasswordIdentityProvider keeps password authentication behind the
// API. Neither browser nor mobile code calls Supabase Auth directly.
func NewSupabasePasswordIdentityProvider() PasswordIdentityProvider {
	baseURL, roleKey := os.Getenv("SYSAP_SUPABASE_AUTH_URL"), os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY")
	auth, err := newSupabaseAuthAdmin(baseURL, roleKey, os.Getenv("SYSAP_ENV"), nil)
	if err != nil {
		return unavailablePasswordIdentityProvider{}
	}
	return &supabasePasswordIdentityProvider{auth: auth}
}

func (p *supabasePasswordIdentityProvider) Authenticate(ctx context.Context, expectedSubject uuid.UUID, password string) (ProviderSession, error) {
	if p == nil || p.auth == nil || expectedSubject == uuid.Nil || password == "" {
		return ProviderSession{}, errLoginDenied
	}
	var account struct {
		ID    uuid.UUID `json:"id"`
		Phone string    `json:"phone"`
		Email string    `json:"email"`
	}
	if err := p.auth.getUser(ctx, expectedSubject, &account); err != nil || account.ID != expectedSubject {
		return ProviderSession{}, errLoginDenied
	}

	payload := struct {
		Phone    string `json:"phone,omitempty"`
		Email    string `json:"email,omitempty"`
		Password string `json:"password"`
	}{Password: password}
	if account.Phone != "" {
		payload.Phone = account.Phone
	} else if account.Email != "" {
		payload.Email = account.Email
	} else {
		return ProviderSession{}, errLoginDenied
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}

	endpoint := p.auth.baseURL.JoinPath("token")
	query := endpoint.Query()
	query.Set("grant_type", "password")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}
	req.Header.Set("apikey", p.auth.roleKey)
	req.Header.Set("Authorization", "Bearer "+p.auth.roleKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := p.auth.client.Do(req)
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ProviderSession{}, errLoginDenied
	}
	responseBody, err := readBounded(response.Body)
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		User         struct {
			ID uuid.UUID `json:"id"`
		} `json:"user"`
	}
	if json.Unmarshal(responseBody, &result) != nil || result.User.ID != expectedSubject || result.RefreshToken == "" || result.ExpiresIn <= 0 {
		return ProviderSession{}, errLoginDenied
	}
	sessionID, aal, err := parseProviderSession(result.AccessToken, expectedSubject)
	if err != nil {
		return ProviderSession{}, errLoginDenied
	}
	return ProviderSession{
		SubjectID: expectedSubject, SessionID: sessionID, AAL: aal,
		AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, ExpiresIn: result.ExpiresIn,
	}, nil
}

func (s *supabaseAuthAdmin) getUser(ctx context.Context, id uuid.UUID, target any) error {
	endpoint := s.baseURL.JoinPath(authAdminEndpoint, id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return errAuthUnavailable
	}
	req.Header.Set("apikey", s.roleKey)
	req.Header.Set("Authorization", "Bearer "+s.roleKey)
	response, err := s.client.Do(req)
	if err != nil {
		return errAuthUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return errAuthUnavailable
	}
	body, err := readBounded(response.Body)
	if err != nil || json.Unmarshal(body, target) != nil {
		return errAuthUnavailable
	}
	return nil
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
