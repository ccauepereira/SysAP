-- Migration: 20260728114200_phase_2d_athlete_provisioning.sql
-- Description: Cria tabelas de convite e perfis pendentes de atleta para a Fase 2D.
-- Strategy: Forward-only. Revert usando DROP TABLE nas tabelas e dependências.

-- 1. Tabela app.athlete_profiles
-- Armazena o perfil do atleta (que pode ainda não ter vínculo no auth.users).
CREATE TABLE app.athlete_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES app.organizations(id) ON DELETE CASCADE,
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

CREATE POLICY insert_athlete_profile ON app.athlete_profiles
    FOR INSERT TO sysap_api
    WITH CHECK (
        organization_id = app.current_tenant_id()
        AND EXISTS (
            SELECT 1
            FROM app.organization_memberships AS membership
            JOIN app.profiles AS actor ON actor.id = membership.profile_id
            WHERE membership.organization_id = app.athlete_profiles.organization_id
              AND membership.role = 'owner'
              AND membership.status = 'active'
              AND actor.auth_user_id = app.current_auth_subject_id()
        )
    );

CREATE POLICY select_athlete_profile ON app.athlete_profiles
    FOR SELECT TO sysap_api
    USING (
        organization_id = app.current_tenant_id()
    );

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

CREATE POLICY insert_activation_invitation ON app.activation_invitations
    FOR INSERT TO sysap_api
    WITH CHECK (
        organization_id = app.current_tenant_id()
        AND EXISTS (
            SELECT 1
            FROM app.organization_memberships AS membership
            JOIN app.profiles AS actor ON actor.id = membership.profile_id
            WHERE membership.organization_id = app.activation_invitations.organization_id
              AND membership.role = 'owner'
              AND membership.status = 'active'
              AND actor.auth_user_id = app.current_auth_subject_id()
        )
        AND invited_by_profile_id = (
            SELECT membership.profile_id
            FROM app.organization_memberships AS membership
            JOIN app.profiles AS actor ON actor.id = membership.profile_id
            WHERE membership.organization_id = app.activation_invitations.organization_id
              AND membership.role = 'owner'
              AND membership.status = 'active'
              AND actor.auth_user_id = app.current_auth_subject_id()
        )
    );

CREATE POLICY select_activation_invitation ON app.activation_invitations
    FOR SELECT TO sysap_api
    USING (
        organization_id = app.current_tenant_id()
    );

GRANT SELECT, INSERT ON app.athlete_profiles TO sysap_api;
GRANT SELECT, INSERT ON app.activation_invitations TO sysap_api;

-- Revogar acessos diretos indesejados
REVOKE ALL ON app.athlete_profiles FROM PUBLIC, anon, authenticated, service_role;
REVOKE ALL ON app.activation_invitations FROM PUBLIC, anon, authenticated, service_role;

-- Garantir que a policy legada da Fase 2B que causa recursão RLS foi removida
DROP POLICY IF EXISTS select_tenant ON app.profiles;
