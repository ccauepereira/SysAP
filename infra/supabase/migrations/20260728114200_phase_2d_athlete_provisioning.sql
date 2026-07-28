-- Migration: 20260728114200_phase_2d_athlete_provisioning.sql
-- Description: Cria tabelas de convite e perfis pendentes de atleta para a Fase 2D.
-- Strategy: Forward-only. Revert usando DROP TABLE nas tabelas e dependências.

-- 1. Tabela app.athlete_profiles
-- Armazena o perfil do atleta (que pode ainda não ter vínculo no auth.users).
CREATE TABLE app.athlete_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_number text NOT NULL,
    display_name text NOT NULL,
    phone_e164 text NOT NULL,
    email text,
    status text NOT NULL DEFAULT 'pending_activation',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT athlete_profiles_enrollment_number_key UNIQUE (enrollment_number)
);

ALTER TABLE app.athlete_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.athlete_profiles FORCE ROW LEVEL SECURITY;

-- 2. Tabela app.activation_invitations
-- Registra os convites de atletas feitos por administradores de uma organização.
CREATE TABLE app.activation_invitations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id uuid NOT NULL REFERENCES app.athlete_profiles(id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES app.organizations(id) ON DELETE CASCADE,
    role text NOT NULL,
    invited_by_profile_id uuid NOT NULL REFERENCES app.profiles(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending',
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE app.activation_invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.activation_invitations FORCE ROW LEVEL SECURITY;

-- Conceder permissões para a API
GRANT SELECT, INSERT, UPDATE, DELETE ON app.athlete_profiles TO sysap_api;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.activation_invitations TO sysap_api;

-- Revogar acessos diretos indesejados
REVOKE ALL ON app.athlete_profiles FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON app.activation_invitations FROM PUBLIC, anon, authenticated, service_role;
