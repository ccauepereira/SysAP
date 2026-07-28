package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

type SupabaseAuthAdmin interface {
	CreateUser(ctx context.Context, phone, password string) (uuid.UUID, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type supabaseAuthAdmin struct {
	baseURL string
	roleKey string
	client  *http.Client
}

func NewSupabaseAuthAdmin() SupabaseAuthAdmin {
	return &supabaseAuthAdmin{
		baseURL: os.Getenv("SYSAP_SUPABASE_URL"),
		roleKey: os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY"),
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (s *supabaseAuthAdmin) CreateUser(ctx context.Context, phone, password string) (uuid.UUID, error) {
	if s.baseURL == "" || s.roleKey == "" {
		return uuid.Nil, errors.New("supabase admin not configured")
	}

	payload := map[string]interface{}{
		"phone":        phone,
		"password":     password,
		"phone_confirm": true,
	}
	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/auth/v1/admin/users", bytes.NewReader(b))
	if err != nil {
		return uuid.Nil, err
	}

	req.Header.Set("apikey", s.roleKey)
	req.Header.Set("Authorization", "Bearer "+s.roleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return uuid.Nil, fmt.Errorf("auth error: %d", resp.StatusCode)
	}

	var result struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return uuid.Nil, err
	}

	return result.ID, nil
}

func (s *supabaseAuthAdmin) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if s.baseURL == "" || s.roleKey == "" {
		return errors.New("supabase admin not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.baseURL+"/auth/v1/admin/users/"+id.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Set("apikey", s.roleKey)
	req.Header.Set("Authorization", "Bearer "+s.roleKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("auth delete error: %d", resp.StatusCode)
	}
	return nil
}
