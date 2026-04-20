#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v node >/dev/null 2>&1; then
  echo "[ERROR] node 未安装，请先安装 Node.js 20+。"
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] docker 未安装。"
  exit 1
fi

echo "[E2E] 检查服务状态..."
if ! docker compose ps --services --filter status=running | rg -q '^nginx$'; then
  echo "[E2E] 服务未启动，先执行 ./start.sh"
  ./start.sh
fi

echo "[E2E] 开始全链路验收..."
node "$ROOT_DIR/scripts/e2e_flow.mjs"

echo
echo "[E2E] 完成。"
