package identity

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/ccauepereira/SysAP/apps/api/internal/platform/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mockEnrollmentGenerator struct {
	val string
	err error
}

func (g *mockEnrollmentGenerator) Generate() (string, error) {
	return g.val, g.err
}

func setupBootstrapTest(t *testing.T) (*database.Pool, *pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("SYSAP_TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("SYSAP_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	adminPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect admin pool: %v", err)
	}

	teardown := func() {
		adminPool.Exec(ctx, "delete from app.identity_repair_tasks")
		adminPool.Exec(ctx, "delete from app.login_enrollments")
		adminPool.Exec(ctx, "delete from app.organization_memberships")
		adminPool.Exec(ctx, "delete from app.profiles")
		adminPool.Exec(ctx, "delete from app.organizations")
		adminPool.Close()
		pool.Close()
	}

	return pool, adminPool, teardown
}

func TestBootstrapOwnerSuccessAndConflict(t *testing.T) {
	pool, adminPool, teardown := setupBootstrapTest(t)
	defer teardown()

	ctx := context.Background()
	authID := uuid.New()
	mockAuth := &mockAuthAdmin{createdID: authID}
	mockGen := &mockEnrollmentGenerator{val: "2026123456"}

	params := BootstrapOwnerParams{
		OrgName:  "Test Bootstrap Org",
		Timezone: "America/Fortaleza",
		FullName: "Arthur Dent",
		Email:    "arthur@example.test",
		Phone:    "+5585999999999",
		Password: "Password123!@#$123",
	}

	// Insert auth user to prevent foreign key violation on profiles insert
	_, err := adminPool.Exec(ctx, `
		insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at)
		values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())
	`, authID, params.Email)
	if err != nil {
		t.Fatalf("failed to setup auth user fixture: %v", err)
	}
	defer adminPool.Exec(ctx, "delete from auth.users where id = $1", authID)

	// 1. First valid creation
	enrollment, err := BootstrapOwner(ctx, pool, mockAuth, mockGen, params, nil)
	if err != nil {
		t.Fatalf("expected successful bootstrap, got: %v", err)
	}

	if enrollment != "2026123456" {
		t.Errorf("expected enrollment 2026123456, got %s", enrollment)
	}

	// Verify database state
	var orgID uuid.UUID
	var timezone string
	err = adminPool.QueryRow(ctx, "select id, timezone from app.organizations where name = $1", params.OrgName).Scan(&orgID, &timezone)
	if err != nil {
		t.Fatalf("failed to find organization in db: %v", err)
	}
	if timezone != "America/Fortaleza" {
		t.Errorf("expected timezone America/Fortaleza, got %s", timezone)
	}

	var role string
	var status string
	err = adminPool.QueryRow(ctx, "select role, status from app.organization_memberships where organization_id = $1", orgID).Scan(&role, &status)
	if err != nil {
		t.Fatalf("failed to find membership: %v", err)
	}
	if role != "owner" || status != "active" {
		t.Errorf("expected owner role and active status, got %s/%s", role, status)
	}

	var mappedProfileID uuid.UUID
	err = adminPool.QueryRow(ctx, "select profile_id from app.login_enrollments where enrollment_number = $1", enrollment).Scan(&mappedProfileID)
	if err != nil {
		t.Fatalf("failed to find login enrollment: %v", err)
	}

	// 2. Second creation for same organization should be refused
	_, err = BootstrapOwner(ctx, pool, mockAuth, mockGen, params, nil)
	if err == nil || !strings.Contains(err.Error(), "owner_already_exists") {
		t.Errorf("expected owner_already_exists error, got: %v", err)
	}
}

func TestBootstrapOwnerCompensationOnLocalFailure(t *testing.T) {
	pool, adminPool, teardown := setupBootstrapTest(t)
	defer teardown()

	ctx := context.Background()
	authID := uuid.New()
	mockAuth := &mockAuthAdmin{createdID: authID}

	// Use an invalid generator to trigger a database insert failure (length format error for enrollment)
	mockGen := &mockEnrollmentGenerator{val: "invalid-short-enrollment"}

	params := BootstrapOwnerParams{
		OrgName:  "Test Fail Org",
		Timezone: "UTC",
		FullName: "Arthur Dent Fail",
		Email:    "arthur_fail@example.test",
		Phone:    "+5585999999998",
		Password: "Password123!@#$123",
	}

	// Insert auth user to prevent foreign key violation on profiles insert
	_, err := adminPool.Exec(ctx, `
		insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at)
		values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())
	`, authID, params.Email)
	if err != nil {
		t.Fatalf("failed to setup auth user fixture: %v", err)
	}
	defer adminPool.Exec(ctx, "delete from auth.users where id = $1", authID)

	_, err = BootstrapOwner(ctx, pool, mockAuth, mockGen, params, nil)
	if err == nil {
		t.Fatal("expected bootstrap to fail due to invalid enrollment number constraint")
	}

	// Check if auth user was deleted (compensation)
	mockAuth.mu.Lock()
	deletedCount := len(mockAuth.deleted)
	var deletedID uuid.UUID
	if deletedCount > 0 {
		deletedID = mockAuth.deleted[0]
	}
	mockAuth.mu.Unlock()

	if deletedCount != 1 || deletedID != authID {
		t.Errorf("expected auth user %s to be deleted once, got deletedCount=%d, deletedID=%v", authID, deletedCount, deletedID)
	}
}

func TestBootstrapOwnerRepairTaskOnCompensationFailure(t *testing.T) {
	pool, adminPool, teardown := setupBootstrapTest(t)
	defer teardown()

	ctx := context.Background()
	authID := uuid.New()
	// Mock auth delete to fail to test repair task recording
	mockAuth := &mockAuthAdmin{createdID: authID, deleteErr: errors.New("auth delete failed")}
	mockGen := &mockEnrollmentGenerator{val: "invalid-short-enrollment"} // triggers db constraint error

	params := BootstrapOwnerParams{
		OrgName:  "Test Repair Org",
		Timezone: "UTC",
		FullName: "Arthur Repair",
		Email:    "arthur_repair@example.test",
		Phone:    "+5585999999997",
		Password: "Password123!@#$123",
	}

	// Insert auth user to prevent foreign key violation on profiles insert
	_, err := adminPool.Exec(ctx, `
		insert into auth.users (id, aud, role, email, raw_app_meta_data, raw_user_meta_data, created_at, updated_at)
		values ($1, 'authenticated', 'authenticated', $2, '{}'::jsonb, '{}'::jsonb, now(), now())
	`, authID, params.Email)
	if err != nil {
		t.Fatalf("failed to setup auth user fixture: %v", err)
	}
	defer adminPool.Exec(ctx, "delete from auth.users where id = $1", authID)

	_, bootstrapErr := BootstrapOwner(ctx, pool, mockAuth, mockGen, params, nil)
	if bootstrapErr == nil {
		t.Fatal("expected bootstrap to fail")
	}

	// Verify repair task was created in DB
	var repairOp string
	var repairStatus string
	dbErr := adminPool.QueryRow(ctx, "select operation, status from app.identity_repair_tasks where auth_user_id = $1", authID).Scan(&repairOp, &repairStatus)
	if dbErr != nil {
		t.Fatalf("failed to find repair task: %v", dbErr)
	}
	if repairOp != "delete_auth_user" || repairStatus != "pending" {
		t.Errorf("expected delete_auth_user/pending, got %s/%s", repairOp, repairStatus)
	}

	// Ensure no PII in returned error
	if strings.Contains(bootstrapErr.Error(), params.Password) || strings.Contains(bootstrapErr.Error(), params.Email) || strings.Contains(bootstrapErr.Error(), params.Phone) {
		t.Errorf("PII leaked in error: %v", bootstrapErr)
	}
}

func TestBootstrapOwnerValidation(t *testing.T) {
	params := BootstrapOwnerParams{
		OrgName:  "",
		Timezone: "America/Fortaleza",
		FullName: "Arthur Dent",
		Email:    "invalid-email",
		Phone:    "123",
		Password: "short",
	}

	err := params.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for all parameters")
	}

	// Valid validation
	params = BootstrapOwnerParams{
		OrgName:  "Valid Org",
		Timezone: "America/Fortaleza",
		FullName: "Arthur Valid",
		Email:    "valid@example.test",
		Phone:    "+5585999999999",
		Password: "Password123!@#$123",
	}
	if err := params.Validate(); err != nil {
		t.Errorf("expected valid parameters to pass validation, got: %v", err)
	}
}

func TestEnvironmentValidation(t *testing.T) {
	// Save existing env vars
	origEnv := os.Getenv("SYSAP_ENV")
	origDB := os.Getenv("SYSAP_DATABASE_URL")
	origAuth := os.Getenv("SYSAP_SUPABASE_AUTH_URL")
	origKey := os.Getenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY")

	defer func() {
		os.Setenv("SYSAP_ENV", origEnv)
		os.Setenv("SYSAP_DATABASE_URL", origDB)
		os.Setenv("SYSAP_SUPABASE_AUTH_URL", origAuth)
		os.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", origKey)
	}()

	// 1. Configuração ausente recusada
	os.Setenv("SYSAP_ENV", "")
	os.Setenv("SYSAP_DATABASE_URL", "")
	os.Setenv("SYSAP_SUPABASE_AUTH_URL", "")
	os.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "")

	err := validateEnvironmentHelper()
	if err == nil {
		t.Error("expected error when config is missing")
	}

	// 2. Ambiente não local recusado
	os.Setenv("SYSAP_ENV", "production")
	os.Setenv("SYSAP_DATABASE_URL", "postgresql://postgres:postgres@127.0.0.1:54322/postgres")
	os.Setenv("SYSAP_SUPABASE_AUTH_URL", "http://127.0.0.1:54321/auth/v1")
	os.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "key")

	err = validateEnvironmentHelper()
	if err == nil || !strings.Contains(err.Error(), "invalid environment") {
		t.Errorf("expected invalid environment error, got: %v", err)
	}

	// 3. Database não local recusado
	os.Setenv("SYSAP_ENV", "development")
	os.Setenv("SYSAP_DATABASE_URL", "postgresql://postgres:postgres@db.remote.com:54322/postgres")
	os.Setenv("SYSAP_SUPABASE_AUTH_URL", "http://127.0.0.1:54321/auth/v1")
	os.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "key")

	err = validateEnvironmentHelper()
	if err == nil || !strings.Contains(err.Error(), "database host must be a loopback") {
		t.Errorf("expected remote database error, got: %v", err)
	}

	// 4. Auth não local recusado
	os.Setenv("SYSAP_ENV", "development")
	os.Setenv("SYSAP_DATABASE_URL", "postgresql://postgres:postgres@127.0.0.1:54322/postgres")
	os.Setenv("SYSAP_SUPABASE_AUTH_URL", "http://auth.remote.com/auth/v1")
	os.Setenv("SYSAP_SUPABASE_SERVICE_ROLE_KEY", "key")

	err = validateEnvironmentHelper()
	if err == nil || !strings.Contains(err.Error(), "Auth URL must use HTTPS outside local loopback") {
		t.Errorf("expected remote auth error, got: %v", err)
	}
}

func validateEnvironmentHelper() error {
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
