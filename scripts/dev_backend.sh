#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

cp -n .env.example .env || true

if ! command -v go >/dev/null 2>&1; then
  echo "[ERROR] 未检测到 go（需要 Go 1.22+）。"
  echo "你可以改用 Docker 一键启动：./start.sh"
  exit 1
fi

set -a
source ./.env
set +a

cd backend
go run ./cmd/server
