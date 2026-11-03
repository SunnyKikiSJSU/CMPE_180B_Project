#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
LOCAL_COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
RDS_COMPOSE_FILE="$ROOT_DIR/docker-compose-rdbs.yml"
LOCAL_PROJECT_NAME="book-metaverse"
RDS_PROJECT_NAME="book-metaverse-rds"
LEGACY_LOCAL_PROJECT_NAME="book-metaverse-local"
TARGET="local"
NO_SEED=false

usage() {
  cat <<'EOF'
Usage: ./scripts/dev.sh [local|rds] [--no-seed]

Examples:
  ./scripts/dev.sh local
  ./scripts/dev.sh rds
  ./scripts/dev.sh rds --no-seed
EOF
}

while (($# > 0)); do
  case "$1" in
    local|rds)
      TARGET="$1"
      ;;
    --no-seed)
      NO_SEED=true
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

step() {
  printf '\n==> %s\n' "$1"
}

die() {
  printf '\nERROR: %s\n' "$1" >&2
  exit 1
}

get_env() {
  local key="$1"
  local fallback="${2:-}"
  local value

  value="$(
    awk -F= -v wanted="$key" '
      $1 == wanted {
        sub(/^[^=]*=/, "")
        sub(/\r$/, "")
        print
        exit
      }
    ' "$ENV_FILE"
  )"

  if [[ -n "$value" ]]; then
    printf '%s' "$value"
  else
    printf '%s' "$fallback"
  fi
}

require_env() {
  local key
  local missing=()

  for key in "$@"; do
    if [[ -z "$(get_env "$key")" ]]; then
      missing+=("$key")
    fi
  done

  if ((${#missing[@]} > 0)); then
    die "Missing required .env values: ${missing[*]}"
  fi
}

compose_cmd() {
  local compose_file="$1"
  local project_name="$2"
  shift 2

  docker compose \
    --env-file "$ENV_FILE" \
    --file "$compose_file" \
    --project-name "$project_name" \
    "$@"
}

stop_other_stack() {
  local compose_file="$1"
  local project_name="$2"

  if [[ -f "$compose_file" ]]; then
    compose_cmd "$compose_file" "$project_name" down --remove-orphans \
      >/dev/null 2>&1 || true
  fi
}

test_tcp() {
  local host="$1"
  local port="$2"
  local timeout_seconds="${3:-5}"

  timeout "${timeout_seconds}s" bash -c \
    'exec 3<>"/dev/tcp/$1/$2"' _ "$host" "$port" \
    >/dev/null 2>&1
}

wait_for_ready() {
  local url="$1"
  local timeout_seconds="${2:-90}"
  local deadline=$((SECONDS + timeout_seconds))
  local body=""

  while ((SECONDS < deadline)); do
    if body="$(curl --fail --silent --show-error --max-time 5 "$url" 2>/dev/null)"; then
      if grep -Eq '"status"[[:space:]]*:[[:space:]]*"ready"' <<<"$body"; then
        printf '%s' "$body"
        return 0
      fi
    fi
    sleep 2
  done

  return 1
}

urlencode() {
  local LC_ALL=C
  local input="$1"
  local output=""
  local character=""
  local encoded=""
  local index

  for ((index = 0; index < ${#input}; index++)); do
    character="${input:index:1}"
    case "$character" in
      [A-Za-z0-9.~_-])
        output+="$character"
        ;;
      *)
        printf -v encoded '%%%02X' "'$character"
        output+="$encoded"
        ;;
    esac
  done

  printf '%s' "$output"
}

build_postgres_url() {
  local user="$1"
  local password="$2"
  local host="$3"
  local port="$4"
  local database="$5"
  local sslmode="$6"

  printf 'postgres://%s:%s@%s:%s/%s?sslmode=%s' \
    "$(urlencode "$user")" \
    "$(urlencode "$password")" \
    "$host" \
    "$port" \
    "$(urlencode "$database")" \
    "$(urlencode "$sslmode")"
}

show_api_logs() {
  local compose_file="$1"
  local project_name="$2"

  compose_cmd "$compose_file" "$project_name" logs --tail 100 api || true
}

on_error() {
  local status=$?
  local compose_file="$LOCAL_COMPOSE_FILE"
  local project_name="$LOCAL_PROJECT_NAME"

  if [[ "$TARGET" == "rds" ]]; then
    compose_file="$RDS_COMPOSE_FILE"
    project_name="$RDS_PROJECT_NAME"
  fi

  printf '\nERROR: workflow failed with status %s\n' "$status" >&2
  show_api_logs "$compose_file" "$project_name" >&2
  exit "$status"
}

trap on_error ERR

if [[ ! -f "$ENV_FILE" ]]; then
  die "Missing environment file: $ENV_FILE"
fi

if ! docker info >/dev/null 2>&1; then
  die "Docker is not running. Start Docker Desktop and try again."
fi

API_PORT="$(get_env API_PORT 8080)"
LOCAL_PROJECT_NAME="$(
  get_env COMPOSE_PROJECT_NAME "$LOCAL_PROJECT_NAME"
)"

if [[ "$TARGET" == "local" ]]; then # For local
  require_env POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD

  stop_other_stack "$RDS_COMPOSE_FILE" "$RDS_PROJECT_NAME"
  stop_other_stack "$LOCAL_COMPOSE_FILE" "$LEGACY_LOCAL_PROJECT_NAME"

  step "Starting local PostgreSQL"
  compose_cmd "$LOCAL_COMPOSE_FILE" "$LOCAL_PROJECT_NAME" \
    up --detach --wait postgres

  export MIGRATE_DATABASE_URL # used by migration up later!!!!
  MIGRATE_DATABASE_URL="$(
    build_postgres_url \
      "$(get_env POSTGRES_USER)" \
      "$(get_env POSTGRES_PASSWORD)" \
      "postgres" \
      "5432" \
      "$(get_env POSTGRES_DB)" \
      "disable"
  )"

  # Apply pending migrations, creating application tables on the first run.
  step "Applying local migrations"
  # dev.sh does not run the manual psql "SELECT version..." command. It lets golang-migrate perform the check.
  compose_cmd "$LOCAL_COMPOSE_FILE" "$LOCAL_PROJECT_NAME" \
    --profile tools run --rm --no-deps migrate # apply migration

  if [[ "$NO_SEED" == false ]]; then
    step "Loading local seed data"
    compose_cmd "$LOCAL_COMPOSE_FILE" "$LOCAL_PROJECT_NAME" \
      --profile tools run --rm --no-deps seed
  fi

  step "Building and starting the local backend"
  compose_cmd "$LOCAL_COMPOSE_FILE" "$LOCAL_PROJECT_NAME" \
    up --detach --build --wait api
else # for RDS
  require_env \
    RDS_HOST RDS_PORT RDS_DB RDS_USER RDS_PASSWORD RDS_SSLMODE

  stop_other_stack "$LOCAL_COMPOSE_FILE" "$LOCAL_PROJECT_NAME"
  stop_other_stack "$LOCAL_COMPOSE_FILE" "$LEGACY_LOCAL_PROJECT_NAME"

  step "Checking RDS network connectivity"
  if ! test_tcp "$(get_env RDS_HOST)" "$(get_env RDS_PORT)"; then
    public_ip="<current-public-ip>"
    if command -v curl >/dev/null 2>&1; then
      public_ip="$(curl --fail --silent --show-error \
        --max-time 10 https://checkip.amazonaws.com | tr -d '[:space:]')" ||
        public_ip="<current-public-ip>"
    fi

    die "RDS is not reachable at $(get_env RDS_HOST):$(get_env RDS_PORT).
Check that RDS public access is enabled and its security group allows
TCP $(get_env RDS_PORT) from ${public_ip}/32. If the backend runs inside the
VPC, allow its security group instead and run this script from that machine."
  fi

  export RDS_DATABASE_URL
  RDS_DATABASE_URL="$(
    build_postgres_url \
      "$(get_env RDS_USER)" \
      "$(get_env RDS_PASSWORD)" \
      "$(get_env RDS_HOST)" \
      "$(get_env RDS_PORT)" \
      "$(get_env RDS_DB)" \
      "$(get_env RDS_SSLMODE)"
  )"

  step "Applying RDS migrations"
  compose_cmd "$RDS_COMPOSE_FILE" "$RDS_PROJECT_NAME" \
    --profile tools run --rm --no-deps migrate

  if [[ "$NO_SEED" == false ]]; then
    step "Loading RDS seed data"
    compose_cmd "$RDS_COMPOSE_FILE" "$RDS_PROJECT_NAME" \
      --profile tools run --rm --no-deps seed
  fi

  step "Building and starting the RDS-backed backend"
  compose_cmd "$RDS_COMPOSE_FILE" "$RDS_PROJECT_NAME" \
    up --detach --build --wait api
fi

step "Checking backend database readiness"
ready_body="$(wait_for_ready "http://127.0.0.1:${API_PORT}/readyz")" ||
  die "Backend did not become database-ready."

printf '\nBackend is ready.\n'
printf 'Target:   %s\n' "$TARGET"
printf 'Health:   http://127.0.0.1:%s/healthz\n' "$API_PORT"
printf 'Ready:    http://127.0.0.1:%s/readyz\n' "$API_PORT"
printf 'Response: %s\n' "$ready_body"
