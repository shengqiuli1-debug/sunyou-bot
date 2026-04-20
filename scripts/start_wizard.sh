#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "========================================"
echo "  sunyou-bot 交互式启动向导"
echo "========================================"
echo "项目目录: $ROOT_DIR"
echo

if ! command -v docker >/dev/null 2>&1; then
  echo "[错误] 未检测到 docker，请先安装 Docker Desktop / Docker Engine。"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "[错误] 未检测到 docker compose 子命令。"
  exit 1
fi

read -r -p "1) 是否从 .env.example 生成 .env？(Y/n): " ENV_ANSWER
ENV_ANSWER=${ENV_ANSWER:-Y}
if [[ "$ENV_ANSWER" =~ ^[Yy]$ ]]; then
  if [[ -f .env ]]; then
    echo "[提示] .env 已存在，跳过覆盖。"
  else
    cp .env.example .env
    echo "[完成] 已生成 .env"
  fi
fi

echo
read -r -p "2) 是否启动服务（docker compose up -d --build）？(Y/n): " UP_ANSWER
UP_ANSWER=${UP_ANSWER:-Y}
if [[ "$UP_ANSWER" =~ ^[Yy]$ ]]; then
  docker compose up -d --build
  echo "[完成] 服务已启动"
else
  echo "[跳过] 未执行启动"
fi

echo
read -r -p "3) 是否查看容器状态（docker compose ps）？(Y/n): " PS_ANSWER
PS_ANSWER=${PS_ANSWER:-Y}
if [[ "$PS_ANSWER" =~ ^[Yy]$ ]]; then
  docker compose ps
fi

echo
read -r -p "4) 是否打印最近日志（每个服务 50 行）？(y/N): " LOG_ANSWER
LOG_ANSWER=${LOG_ANSWER:-N}
if [[ "$LOG_ANSWER" =~ ^[Yy]$ ]]; then
  docker compose logs --tail=50
fi

echo
cat <<MSG
========================================
下一步访问：
- 首页:   http://localhost
- 健康:   http://localhost/healthz

停止服务：
- docker compose down
========================================
MSG
