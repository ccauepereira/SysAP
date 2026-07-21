#!/usr/bin/env bash

set -euo pipefail

umask 077

output_file=""
environment_file=""
loopback_network="sysap-loopback"
network_managed_label="com.sysap.managed"
network_managed_value="true"
network_purpose_label="com.sysap.purpose"
network_purpose_value="local-supabase"
network_version_label="com.sysap.version"
network_version_value="1"
network_bind_address="127.0.0.1"

cleanup() {
  if [ -n "$output_file" ] && [ -f "$output_file" ]; then
    rm -f -- "$output_file"
  fi
  if [ -n "$environment_file" ] && [ -f "$environment_file" ]; then
    rm -f -- "$environment_file"
  fi
}

trap cleanup EXIT HUP INT TERM

capture_supabase() {
  output_file=$(mktemp ./.sysap-supabase.XXXXXX)
  chmod 600 "$output_file"
  set +e
  pnpm --silent exec supabase "$@" >"$output_file" 2>&1
  capture_status=$?
  set -e
  return "$capture_status"
}

network_exists() {
  docker network ls \
    --filter "name=${loopback_network}" \
    --format '{{.Name}}' 2>&1 |
    grep -Fxq "$loopback_network"
}

inspect_loopback_network() {
  network_details=$(docker network inspect --format \
    '{{.Name}}|{{.Driver}}|{{index .Labels "com.sysap.managed"}}|{{index .Labels "com.sysap.purpose"}}|{{index .Labels "com.sysap.version"}}|{{index .Options "com.docker.network.bridge.host_binding_ipv4"}}|{{len .Containers}}' \
    "$loopback_network" 2>/dev/null) || return 1

  IFS='|' read -r \
    inspected_name \
    inspected_driver \
    inspected_managed \
    inspected_purpose \
    inspected_version \
    inspected_bind_address \
    inspected_container_count <<EOF
$network_details
EOF

  case "$inspected_container_count" in
    ''|*[!0-9]*)
      return 1
      ;;
  esac
}

network_has_expected_configuration() {
  [ "$inspected_name" = "$loopback_network" ] &&
    [ "$inspected_driver" = "bridge" ] &&
    [ "$inspected_managed" = "$network_managed_value" ] &&
    [ "$inspected_purpose" = "$network_purpose_value" ] &&
    [ "$inspected_version" = "$network_version_value" ] &&
    [ "$inspected_bind_address" = "$network_bind_address" ]
}

label_is_absent() {
  [ -z "$1" ] || [ "$1" = "<no value>" ]
}

network_is_migratable_legacy() {
  [ "$inspected_name" = "$loopback_network" ] &&
    [ "$inspected_driver" = "bridge" ] &&
    [ "$inspected_bind_address" = "$network_bind_address" ] &&
    [ "$inspected_container_count" = "0" ] &&
    label_is_absent "$inspected_managed" &&
    label_is_absent "$inspected_purpose" &&
    label_is_absent "$inspected_version"
}

create_loopback_network() {
  if ! docker network create \
    --driver bridge \
    --label "${network_managed_label}=${network_managed_value}" \
    --label "${network_purpose_label}=${network_purpose_value}" \
    --label "${network_version_label}=${network_version_value}" \
    --opt "com.docker.network.bridge.host_binding_ipv4=${network_bind_address}" \
    "$loopback_network" >/dev/null 2>&1; then
    return 1
  fi

  inspect_loopback_network &&
    network_has_expected_configuration &&
    [ "$inspected_container_count" = "0" ]
}

ensure_loopback_network() {
  if ! network_exists; then
    if create_loopback_network; then
      return 0
    fi
    printf 'Supabase local: loopback network setup failed.\n' >&2
    return 1
  fi

  if ! inspect_loopback_network; then
    printf 'Supabase local: loopback network validation failed.\n' >&2
    return 1
  fi

  if network_has_expected_configuration; then
    if [ "$inspected_container_count" = "0" ]; then
      return 0
    fi
    printf 'Supabase local: loopback network is not empty.\n' >&2
    return 1
  fi

  if network_is_migratable_legacy; then
    if docker network rm "$loopback_network" >/dev/null 2>&1 &&
      create_loopback_network; then
      return 0
    fi
    printf 'Supabase local: legacy loopback network migration failed.\n' >&2
    return 1
  fi

  printf 'Supabase local: incompatible loopback network detected.\n' >&2
  return 1
}

validate_active_loopback_network() {
  if ! network_exists ||
    ! inspect_loopback_network ||
    ! network_has_expected_configuration ||
    [ "$inspected_container_count" = "0" ]; then
    printf 'Supabase local: active loopback network validation failed.\n' >&2
    return 1
  fi
}

environment_file_has_expected_shape() {
  [ "$(wc -l <"$environment_file")" -eq 2 ] &&
    [ "$(grep -c "^SYSAP_DATABASE_URL='[^']*'$" "$environment_file")" -eq 1 ] &&
    [ "$(grep -c "^SYSAP_TEST_DATABASE_URL='[^']*'$" "$environment_file")" -eq 1 ] &&
    ! grep -Ev "^(SYSAP_DATABASE_URL|SYSAP_TEST_DATABASE_URL)='[^']*'$" \
      "$environment_file" >/dev/null
}

fixed_result() {
  action=$1
  shift

  if capture_supabase "$@"; then
    printf 'Supabase local: %s succeeded.\n' "$action"
    return 0
  fi

  if [ "$action" = "reset" ] || [ "$action" = "lint" ]; then
    failure_category="unknown"
    failure_statement=""
    failure_sqlstate=""
    while IFS= read -r output_line; do
      if [[ "$output_line" =~ [Aa]t[[:space:]]statement:?[[:space:]]+([0-9]+) ]]; then
        failure_statement=${BASH_REMATCH[1]}
      fi
      if [[ "$output_line" =~ SQLSTATE[[:space:]]+([0-9A-Z]{5}) ]]; then
        failure_sqlstate=${BASH_REMATCH[1]}
      fi
      case "$output_line" in
        *"create role"*|*"CREATE ROLE"*)
          if [ "$failure_category" = "unknown" ]; then
            failure_category="restricted role creation"
          fi
          ;;
        *"alter role"*|*"ALTER ROLE"*)
          if [ "$failure_category" = "unknown" ]; then
            failure_category="restricted role alteration"
          fi
          ;;
        *"default privileges"*|*"DEFAULT PRIVILEGES"*)
          if [ "$failure_category" = "unknown" ]; then
            failure_category="default privileges"
          fi
          ;;
        *"create policy"*|*"CREATE POLICY"*)
          if [ "$failure_category" = "unknown" ]; then
            failure_category="RLS policy"
          fi
          ;;
        *" grant "*|*"GRANT "*)
          if [ "$failure_category" = "unknown" ]; then
            failure_category="explicit grant"
          fi
          ;;
        *supabase_admin*"permission denied"*|*"permission denied"*supabase_admin*|*supabase_admin*"must be member"*|*"must be member"*supabase_admin*)
          failure_category="administrative default privileges"
          break
          ;;
        *"already exists"*)
          failure_category="persistent role already exists"
          break
          ;;
        *"does not exist"*)
          failure_category="required local role is missing"
          break
          ;;
        *"cannot be dropped"*|*"dependent objects"*|*"objects depend"*)
          failure_category="persistent role dependencies"
          break
          ;;
        *"permission denied"*|*"must be member"*)
          failure_category="local role permission"
          break
          ;;
        *supabase_admin*)
          failure_category="administrative default privileges"
          break
          ;;
        *service_role*)
          failure_category="client-role privileges"
          break
          ;;
        *sysap_api*)
          failure_category="restricted API role"
          break
          ;;
        *bootstrap_metadata*)
          failure_category="bootstrap metadata"
          break
          ;;
        *migration*|*Migration*|*SQLSTATE*|*"syntax error"*|*"permission denied"*)
          failure_category="database migration"
          ;;
        *config*|*Config*|*TOML*)
          failure_category="local configuration"
          ;;
        *container*|*Container*|*Docker*|*docker*)
          failure_category="local container"
          ;;
      esac
    done <"$output_file"
    if [ -n "$failure_statement" ] && [ -n "$failure_sqlstate" ]; then
      printf 'Supabase local: %s failed (%s, statement %s, SQLSTATE %s).\n' \
        "$action" "$failure_category" "$failure_statement" "$failure_sqlstate" >&2
    elif [ -n "$failure_statement" ]; then
      printf 'Supabase local: %s failed (%s, statement %s).\n' \
        "$action" "$failure_category" "$failure_statement" >&2
    elif [ -n "$failure_sqlstate" ]; then
      printf 'Supabase local: %s failed (%s, SQLSTATE %s).\n' \
        "$action" "$failure_category" "$failure_sqlstate" >&2
    else
      printf 'Supabase local: %s failed (%s).\n' "$action" "$failure_category" >&2
    fi
    return 1
  fi

  printf 'Supabase local: %s failed.\n' "$action" >&2
  return 1
}

case "${1:-}" in
  start)
    ensure_loopback_network
    fixed_result "start" start --workdir infra --network-id "$loopback_network"
    ;;
  stop)
    fixed_result "stop" stop --workdir infra
    ;;
  status)
    if capture_supabase status --workdir infra; then
      printf 'Supabase local: running.\n'
    else
      printf 'Supabase local: not running.\n'
    fi
    ;;
  env)
    if [ -L .env.local ]; then
      printf 'Supabase local: environment file must not be a symlink.\n' >&2
      exit 1
    fi

    if ! capture_supabase status --workdir infra -o env; then
      printf 'Supabase local: environment capture failed.\n' >&2
      exit 1
    fi

    database_url=""
    database_url_count=0
    while IFS='=' read -r name value; do
      if [ "$name" = "DB_URL" ]; then
        database_url=$value
        database_url_count=$((database_url_count + 1))
      fi
    done <"$output_file"

    if [ "$database_url_count" -ne 1 ] || [ -z "$database_url" ]; then
      printf 'Supabase local: database environment unavailable.\n' >&2
      exit 1
    fi

    environment_file=$(mktemp ./.env.local.XXXXXX)
    chmod 600 "$environment_file"

    if ! printf '%s' "$database_url" |
      node scripts/local-database-url.mjs infra/supabase/config.toml \
        >"$environment_file" 2>/dev/null; then
      printf 'Supabase local: database URL validation failed.\n' >&2
      exit 1
    fi

    if ! environment_file_has_expected_shape; then
      printf 'Supabase local: generated environment file is invalid.\n' >&2
      exit 1
    fi
    if [ -L .env.local ]; then
      printf 'Supabase local: environment file must not be a symlink.\n' >&2
      exit 1
    fi
    if ! mv -fT -- "$environment_file" .env.local; then
      printf 'Supabase local: environment file update failed.\n' >&2
      exit 1
    fi
    environment_file=""
    if [ -L .env.local ] || [ "$(stat -c '%a' .env.local)" != "600" ]; then
      printf 'Supabase local: environment file protection failed.\n' >&2
      exit 1
    fi
    printf 'Supabase local: environment file updated.\n'
    ;;
  reset)
    validate_active_loopback_network
    fixed_result "reset" db reset --local --workdir infra --network-id "$loopback_network"
    ;;
  lint)
    fixed_result "lint" db lint --local --workdir infra
    ;;
  *)
    printf 'Usage: %s {start|stop|status|env|reset|lint}\n' "$0" >&2
    exit 2
    ;;
esac
