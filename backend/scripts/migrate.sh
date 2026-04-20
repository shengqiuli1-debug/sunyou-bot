#!/usr/bin/env bash
set -euo pipefail

: "${POSTGRES_HOST:=127.0.0.1}"
: "${POSTGRES_PORT:=5432}"
: "${POSTGRES_DB:=sunyou_bot}"
: "${POSTGRES_USER:=sunyou}"
: "${POSTGRES_PASSWORD:=sunyou123}"

for f in $(ls ../migrations/*.sql | sort); do
  echo "Applying $f"
  PGPASSWORD="$POSTGRES_PASSWORD" psql \
    -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -f "$f"
done
