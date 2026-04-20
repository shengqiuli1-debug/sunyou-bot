#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

echo "[INFO] 正在停止 sunyou-bot..."
docker compose down --remove-orphans

echo "[OK] 已停止"
