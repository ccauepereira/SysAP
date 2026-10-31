/**
 * Core Identity & Authentication types for SysAP Mobile
 * Source of truth: contracts/openapi/openapi.yaml
 */

export type UserRole = "owner" | "athlete" | "trainer";

export type MembershipStatus = "active" | "suspended";

export interface Profile {
  id: string;
  displayName: string;
}

export interface Membership {
  organizationId: string;
  role: UserRole;
  status: MembershipStatus;
}

export interface CurrentIdentity {
  profile: Profile;
  memberships: Membership[];
}

export interface AuthSession {
  version: 1;
  sessionId: string;
  profileId: string;
  organizationId: string;
  role: UserRole;
  accessToken: string;
  refreshToken: string;
  expiresAt: number; // Unix timestamp in milliseconds
}

export type AuthStatus = "loading" | "unauthenticated" | "authenticated" | "error";

export type SafeAuthErrorCode =
  | "invalid_credentials"
  | "account_suspended"
  | "membership_inactive"
  | "unauthorized_role"
  | "access_denied"
  | "temporary_failure"
  | "no_connection"
  | "service_unavailable"
  | "session_expired"
  | "storage_failed"
  | "generic_error";

export interface SafeAuthError {
  code: SafeAuthErrorCode;
  message: string;
}
