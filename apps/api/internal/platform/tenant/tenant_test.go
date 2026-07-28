package tenant_test

import (
	"net/http"
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/tenant"
)

func TestParseOrganizationHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		want    string
		wantErr error
	}{
		{
			name: "valid canonical uuid",
			header: http.Header{
				"X-Organization-Id": []string{"30000000-0000-4000-8000-000000000001"},
			},
			want:    "30000000-0000-4000-8000-000000000001",
			wantErr: nil,
		},
		{
			name:    "missing header",
			header:  http.Header{},
			want:    "",
			wantErr: tenant.ErrInvalidOrganization,
		},
		{
			name: "multiple headers",
			header: http.Header{
				"X-Organization-Id": []string{"30000000-0000-4000-8000-000000000001", "30000000-0000-4000-8000-000000000002"},
			},
			want:    "",
			wantErr: tenant.ErrInvalidOrganization,
		},
		{
			name: "zero uuid",
			header: http.Header{
				"X-Organization-Id": []string{"00000000-0000-0000-0000-000000000000"},
			},
			want:    "",
			wantErr: tenant.ErrInvalidOrganization,
		},
		{
			name: "non canonical uppercase",
			header: http.Header{
				"X-Organization-Id": []string{"30000000-0000-4000-8000-00000000000A"},
			},
			want:    "",
			wantErr: tenant.ErrInvalidOrganization,
		},
		{
			name: "invalid format",
			header: http.Header{
				"X-Organization-Id": []string{"not-a-uuid"},
			},
			want:    "",
			wantErr: tenant.ErrInvalidOrganization,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tenant.ParseOrganizationHeader(tt.header)
			if err != tt.wantErr {
				t.Errorf("ParseOrganizationHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseOrganizationHeader() got = %v, want %v", got, tt.want)
			}
		})
	}
}
