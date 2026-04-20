#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/.env.example"

mask_secret() {
  local s="${1:-}"
  local n=${#s}
  if [[ $n -le 0 ]]; then
    printf "%s" "(empty)"
    return
  fi
  if [[ $n -le 4 ]]; then
    printf "%s" "****"
    return
  fi
  printf "%s" "${s:0:2}****${s: -2}"
}

read_env_file() {
  local f="$1"
  set -a
  # shellcheck disable=SC1090
  source "$f"
  set +a
}

if [[ -f "$ENV_FILE" ]]; then
  read_env_file "$ENV_FILE"
  ACTIVE_ENV_FILE="$ENV_FILE"
elif [[ -f "$ENV_EXAMPLE" ]]; then
  read_env_file "$ENV_EXAMPLE"
  ACTIVE_ENV_FILE="$ENV_EXAMPLE (fallback)"
else
  ACTIVE_ENV_FILE="not found"
fi

POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-sunyou_bot}"
POSTGRES_USER="${POSTGRES_USER:-sunyou}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-sunyou123}"
POSTGRES_SSLMODE="${POSTGRES_SSLMODE:-disable}"
POSTGRES_BIND_ADDR="${POSTGRES_BIND_ADDR:-127.0.0.1}"
POSTGRES_PUBLIC_PORT="${POSTGRES_PUBLIC_PORT:-5432}"
REDIS_ADDR="${REDIS_ADDR:-redis:6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
REDIS_DB="${REDIS_DB:-0}"
REDIS_BIND_ADDR="${REDIS_BIND_ADDR:-127.0.0.1}"
REDIS_PUBLIC_PORT="${REDIS_PUBLIC_PORT:-6379}"
LLM_ENABLED="${LLM_ENABLED:-false}"
LOG_DIR="${LOG_DIR:-../runtime/logs/backend}"

COMPOSE_SERVICES="backend, postgres, redis, nginx"
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  if COMPOSE_LIST="$(cd "$ROOT_DIR" && docker compose config --services 2>/dev/null)"; then
    if [[ -n "${COMPOSE_LIST}" ]]; then
      COMPOSE_SERVICES="$(echo "$COMPOSE_LIST" | paste -sd "," -)"
    fi
  fi
fi

BACKEND_RUNNING="unknown"
POSTGRES_RUNTIME_PORT="unknown"
REDIS_RUNTIME_PORT="unknown"
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  if docker info >/dev/null 2>&1; then
    STATUS_LINE="$(cd "$ROOT_DIR" && docker compose ps backend --format '{{.Status}}' 2>/dev/null | head -n1 || true)"
    if [[ -n "$STATUS_LINE" ]]; then
      BACKEND_RUNNING="$STATUS_LINE"
    else
      BACKEND_RUNNING="not running"
    fi

    PG_PORT_LINE="$(cd "$ROOT_DIR" && docker compose port postgres 5432 2>/dev/null | head -n1 || true)"
    if [[ -z "$PG_PORT_LINE" || "$PG_PORT_LINE" == ":0" ]]; then
      POSTGRES_RUNTIME_PORT="not published"
    else
      POSTGRES_RUNTIME_PORT="$PG_PORT_LINE"
    fi

    REDIS_PORT_LINE="$(cd "$ROOT_DIR" && docker compose port redis 6379 2>/dev/null | head -n1 || true)"
    if [[ -z "$REDIS_PORT_LINE" || "$REDIS_PORT_LINE" == ":0" ]]; then
      REDIS_RUNTIME_PORT="not published"
    else
      REDIS_RUNTIME_PORT="$REDIS_PORT_LINE"
    fi
  else
    BACKEND_RUNNING="docker daemon unavailable"
  fi
fi

echo "============================================================"
echo "sunyou-bot 运行信息"
echo "============================================================"
echo
echo "[1] Backend 日志查看"
echo "- Docker Compose:  docker compose logs -f backend"
echo "- 本地 go run:     在执行 'cd backend && go run ./cmd/server' 的那个终端查看 stdout"
echo "- 日志文件落盘:    runtime/logs/backend/app.log"
echo "- tail 文件日志:   tail -f runtime/logs/backend/app.log"
echo
echo "[2] Compose 服务"
echo "- 服务名: ${COMPOSE_SERVICES}"
echo "- backend 当前状态: ${BACKEND_RUNNING}"
echo
echo "[3] 数据库连接信息 (当前读取值)"
echo "- host:     ${POSTGRES_HOST}"
echo "- port:     ${POSTGRES_PORT}"
echo "- compose配置映射: ${POSTGRES_BIND_ADDR}:${POSTGRES_PUBLIC_PORT} -> postgres:5432"
echo "- 运行时映射:      ${POSTGRES_RUNTIME_PORT}"
echo "- database: ${POSTGRES_DB}"
echo "- username: ${POSTGRES_USER}"
echo "- password: $(mask_secret "${POSTGRES_PASSWORD}")"
echo "- sslmode:  ${POSTGRES_SSLMODE}"
echo "- 来源文件: ${ACTIVE_ENV_FILE}"
echo
echo "[3.1] Redis 连接信息 (当前读取值)"
echo "- addr:     ${REDIS_ADDR}"
echo "- compose配置映射: ${REDIS_BIND_ADDR}:${REDIS_PUBLIC_PORT} -> redis:6379"
echo "- 运行时映射:      ${REDIS_RUNTIME_PORT}"
echo "- password: $(mask_secret "${REDIS_PASSWORD}")"
echo "- db:       ${REDIS_DB}"
echo
echo "[4] 进入数据库 (Docker Compose)"
echo "- 进入 psql:"
echo "  docker compose exec postgres psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}"
echo
echo "[5] 查看表和 bot_reply_audits"
echo "- 列表表:"
echo "  \\dt"
echo "- 查看最近 20 条:"
echo "  select id, trace_id, room_id, trigger_message_id, reply_message_id, reply_source, trigger_type, trigger_reason, force_reply, hype_score, llm_enabled, provider_initialized, api_key_present, provider, model, fallback_reason, created_at"
echo "  from bot_reply_audits"
echo "  order by created_at desc"
echo "  limit 20;"
echo
echo "[5.1] Redis 进入与检查"
echo "- 进入 redis-cli:"
echo "  docker compose exec redis redis-cli"
echo "- 查看常见 key:"
echo "  keys room:*"
echo
echo "[6] 持久化说明"
echo "- PostgreSQL 持久化卷: pgdata -> /var/lib/postgresql/data"
echo "- Redis 持久化卷:      redisdata -> /data"
echo "- 删除容器不会丢数据: 是（只要不执行 down -v）"
echo "- 会丢数据的操作:      docker compose down -v"
echo "- 查看卷详情命令:"
echo "  docker volume ls | grep -E 'pgdata|redisdata'"
echo "  docker volume inspect \$(docker volume ls -q | grep 'pgdata' | head -n1)"
echo "  docker volume inspect \$(docker volume ls -q | grep 'redisdata' | head -n1)"
echo
echo "[7] env 生效说明"
echo "- Docker backend: docker-compose.yml 使用 env_file: .env，并额外覆盖 POSTGRES_HOST/REDIS_ADDR/BACKEND_ALLOWED_ORIGIN"
echo "- 本地 go run: backend/config 会按顺序尝试加载 .env -> ../.env -> ../../.env（通常命中项目根 .env）"
echo "- 后端日志目录配置: LOG_DIR=${LOG_DIR}"
echo
echo "[8] 数据库类型提醒"
echo "- 当前项目后端数据库是 PostgreSQL（服务名 postgres），不是 MySQL。"
echo "- 如需 MySQL 需要单独新增服务和代码适配，当前版本未启用 MySQL。"
echo
echo "[9] LLM 状态 (env)"
echo "- LLM_ENABLED=${LLM_ENABLED}"
echo
echo "[10] 快速排查命令"
echo "- backend 日志:          docker compose logs -f backend"
echo "- 查看 backend 状态:     docker compose ps backend"
echo "- 端口未生效时重建:      docker compose up -d --force-recreate postgres redis"
echo "- 查看 bot 审计 SQL:     docker compose exec postgres psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c \"select id, trace_id, reply_source, trigger_type, trigger_reason, created_at from bot_reply_audits order by created_at desc limit 20;\""
echo
