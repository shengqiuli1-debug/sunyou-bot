#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

ensure_docker_ready() {
  if docker info >/dev/null 2>&1; then
    return 0
  fi

  echo "[WARN] Docker daemon 当前不可用。"
  if [[ "$(uname -s)" == "Darwin" ]] && command -v open >/dev/null 2>&1; then
    echo "[INFO] 尝试自动启动 Docker Desktop..."
    open -a Docker >/dev/null 2>&1 || true
    for i in {1..45}; do
      if docker info >/dev/null 2>&1; then
        echo "[OK] Docker daemon 已就绪。"
        return 0
      fi
      sleep 2
    done
  fi

  echo "[ERROR] 仍无法连接 Docker daemon。"
  echo "请先启动 Docker Desktop / Docker Engine 后重试。"
  return 1
}

if ! command -v docker >/dev/null 2>&1; then
  echo "[ERROR] docker 未安装，请先安装 Docker。"
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "[ERROR] 未检测到 docker compose。"
  exit 1
fi

if [[ ! -f .env ]]; then
  cp .env.example .env
  echo "[INFO] 已自动创建 .env（来自 .env.example）"
fi

ensure_docker_ready

echo "[INFO] 正在启动 sunyou-bot..."
echo "[INFO] 自动清理旧容器（保留数据卷，不会删库）..."
docker compose down --remove-orphans || true

if lsof -nP -iTCP:80 -sTCP:LISTEN >/dev/null 2>&1; then
  echo "[ERROR] 端口 80 已被占用，无法启动 nginx。"
  echo "请先释放 80 端口，或修改 docker-compose.yml 的 nginx 映射端口。"
  lsof -nP -iTCP:80 -sTCP:LISTEN
  exit 1
fi

echo "[INFO] 启动最新容器..."
docker compose up -d --build --remove-orphans

echo "[INFO] 等待服务就绪..."
if command -v curl >/dev/null 2>&1; then
  HEALTH_OK=0
  for i in {1..40}; do
    if curl -fsS http://localhost/healthz >/dev/null 2>&1; then
      HEALTH_OK=1
      break
    fi
    sleep 1
  done

  if [[ "$HEALTH_OK" -ne 1 ]]; then
    echo "[ERROR] 健康检查未通过，请查看日志："
    docker compose logs --tail=200 backend
    exit 1
  fi
else
  echo "[WARN] 未检测到 curl，跳过 /healthz 主动探测。"
fi

echo
echo "[OK] 启动完成"
echo "- 首页:   http://localhost"
echo "- 健康:   http://localhost/healthz"
echo
docker compose ps
