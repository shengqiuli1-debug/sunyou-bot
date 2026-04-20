# sunyou-bot

手机 H5 群聊产品 MVP：用户可创建临时房间，邀请好友加入；房间内 Bot 会以「阴阳裁判 / 损友 NPC / 冷面旁白」三种风格插话吐槽。支持 `normal / target / immune` 三种身份、点数系统、模拟充值、基础风控和房间战报。

## 1. 目录结构

```text
sunyou-bot/
├── frontend/                 # Vue3 + Vite + TypeScript H5
├── backend/                  # Go + Gin + WebSocket
├── deploy/
│   ├── nginx/
│   └── docker/
├── docs/
├── scripts/
├── .gitignore
├── .env.example
├── docker-compose.yml
└── README.md
```

## 2. 功能范围（MVP）

- H5 页面：首页、创建房间、入场确认、聊天房间、战报页、点数页
- 匿名轻用户
- 房间创建/加入/自动过期
- 三种身份：`normal / target / immune`
- `target` 二次确认
- WebSocket 实时聊天
- 房主控制：切角色、切火力、临时闭嘴、结束房间
- Bot（规则+模板，不依赖大模型）
- 战报自动生成
- 点数系统（开房扣点 + 模拟充值）
- 基础风控（敏感词 + 发言频率）
- 举报接口

## 3. 技术栈

- Frontend: Vue 3 + Vite + TypeScript + Pinia + Vue Router
- Backend: Go 1.22+ + Gin + Gorilla WebSocket + Redis + PostgreSQL
- Deploy: Docker Compose + Nginx

## 4. 环境变量

```bash
cp .env.example .env
```

默认值可直接用于 `docker compose` 本地启动。

## 5. 本地开发步骤

### 5.1 使用 Docker（推荐）

```bash
cp .env.example .env
docker compose up -d --build
```

访问：

- 前端 H5: [http://localhost](http://localhost)
- 后端健康检查: [http://localhost/api/v1](http://localhost/api/v1)（API 前缀）
- 健康接口: [http://localhost/healthz](http://localhost/healthz)

### 5.2 前后端分开调试

1. 启动 PostgreSQL + Redis（可用 compose）。
2. 后端：

```bash
cd backend
# 需要本机已安装 go
go mod tidy
go run ./cmd/server
```

3. 前端：

```bash
cd frontend
npm install
npm run dev
```

前端默认代理：

- `/api` -> `http://localhost:8080`
- `/ws` -> `ws://localhost:8080`

## 6. 生产部署步骤（单机 4C4G）

1. 安装 Docker / Docker Compose。
2. 拉取代码，编辑 `.env`。
3. 启动：

```bash
docker compose up -d --build
```

4. 确认容器：

```bash
docker compose ps
```

5. 配置服务器安全组放行 `80` 端口。

## 7. 数据存储说明

### PostgreSQL 持久化

- users
- point_ledger
- rooms
- room_members
- messages
- bot_reply_audits
- room_reports
- abuse_reports
- risk_logs

### Redis 实时状态

- `room:{id}:meta`
- `room:{id}:members`
- `room:{id}:online`
- `room:{id}:messages`
- `room:{id}:bot:cooldown`
- `room:{id}:bot:muted`
- `room:{id}:stats`
- `room:{id}:last_msg_at`

## 8. API 与联调

接口示例见：

- [docs/curl-examples.md](./docs/curl-examples.md)

核心 API：

- `POST /api/v1/users/guest`
- `POST /api/v1/rooms`
- `POST /api/v1/rooms/:id/join`
- `POST /api/v1/rooms/:id/identity`
- `POST /api/v1/rooms/:id/control`
- `POST /api/v1/rooms/:id/end`
- `GET  /api/v1/rooms/:id/report`
- `POST /api/v1/points/recharge`

WebSocket：

- `GET /ws/rooms/:id?token=...`

事件类型：

- `bootstrap`
- `chat`
- `system`
- `member_update`
- `control_update`
- `room_end`
- `bot_debug`（开发排查事件流）

Bot 审计调试 API：

- `GET /api/v1/debug/rooms/:roomId/bot-audits?page=1&pageSize=20&replySource=&botRole=`
- `GET /api/v1/debug/messages/:messageId/bot-audit`
- `GET /api/v1/debug/bot-audits/:id`

## 9. Bot 日志查看

最直接命令（Docker）：

```bash
docker compose logs -f backend
# 或
./scripts/logs_backend.sh
```

日志文件（落盘）：

- 宿主机目录：`./runtime/logs`
- backend 主日志：`./runtime/logs/backend/app.log`
- 容器内路径：`/app/runtime/logs/backend/app.log`
- 自动创建：backend 启动时会自动创建 `runtime/logs`、`runtime/logs/backend`（容器内对应 `/app/runtime/logs/backend`）

直接看文件日志：

```bash
tail -f runtime/logs/backend/app.log
```

只看 Bot/LLM/Chat 关键事件：

```bash
docker compose logs -f backend | rg "\[BOT\]|\[LLM\]|\[CHAT\]|bot\.|chat\."
```

本机 go run 启动时：

```bash
cd backend
go run ./cmd/server
```

日志直接打印在当前终端。

## 快速排查：日志和数据库在哪里看

先执行一条总览脚本：

```bash
./scripts/show_runtime_info.sh
```

它会打印：

- backend 日志命令
- backend 文件日志路径与 tail 命令
- compose 服务名和 backend 运行状态
- 数据库 host/port/db/user/password（密码掩码）
- 进入数据库命令
- `bot_reply_audits` 查询 SQL
- 当前 env 读取路径
- LLM 开关状态

最短常用命令：

```bash
# 1) 看 backend 日志
docker compose logs -f backend

# 1.1) 看落盘日志文件
tail -f runtime/logs/backend/app.log

# 2) 进入数据库
docker compose exec postgres psql -U sunyou -d sunyou_bot

# 3) 看 Redis key
docker compose exec redis redis-cli keys 'room:*'

# 4) 看审计表最近 20 条
docker compose exec postgres psql -U sunyou -d sunyou_bot -c \
"select id, trace_id, reply_source, trigger_type, trigger_reason, created_at from bot_reply_audits order by created_at desc limit 20;"
```

如果你看到 `docker compose ps` 里 `postgres/redis` 仅显示 `5432/tcp` 或 `6379/tcp`（没有 `127.0.0.1:xxxx->...`），说明端口映射还没在当前容器生效，执行：

```bash
docker compose up -d --force-recreate postgres redis
```

端口暴露（宿主机访问）：

- PostgreSQL：`127.0.0.1:5432` -> 容器 `postgres:5432`
- Redis：`127.0.0.1:6379` -> 容器 `redis:6379`
- 如需改端口可在 `.env` 设置：`POSTGRES_PUBLIC_PORT`、`REDIS_PUBLIC_PORT`
- 如需对外网开放（谨慎），可把 `.env` 的 `POSTGRES_BIND_ADDR`、`REDIS_BIND_ADDR` 从 `127.0.0.1` 改成 `0.0.0.0`

持久化说明（重点）：

- PostgreSQL 使用卷：`pgdata`
- Redis 使用卷：`redisdata`
- 删除容器不会丢数据：`docker compose down` 后再 `up`，数据仍在
- 会清空数据的命令：`docker compose down -v`
- 检查卷：

```bash
docker volume ls | grep -E 'pgdata|redisdata'
```

本地直接运行后端时：

- 命令：`cd backend && go run ./cmd/server`
- 日志位置：当前终端 stdout
- 当前项目未配置后端日志文件落盘

env 生效说明：

- Docker Compose：`backend` 服务使用 `env_file: .env`，并在 `docker-compose.yml` 中覆盖 `POSTGRES_HOST/REDIS_ADDR/BACKEND_ALLOWED_ORIGIN`
- 本地 go run：后端配置加载顺序是 `.env -> ../.env -> ../../.env`（通常命中项目根目录 `.env`）

数据库类型说明：

- 当前项目数据库是 **PostgreSQL**（不是 MySQL）。
- 如你要 MySQL，需要单独新增 MySQL 服务并改后端驱动/SQL 兼容层。

## 10. 常见故障排查

1. 前端白屏：
- 检查 `docker compose logs nginx`
- 检查 Nginx 配置是否已挂载 `deploy/nginx/default.conf`

2. WebSocket 连接失败：
- 检查 Nginx `/ws/` 代理是否包含 `Upgrade/Connection` 头
- 检查浏览器请求地址是否带 `token`

3. 后端启动失败：
- 检查 `.env` 的 PostgreSQL/Redis 地址
- `docker compose logs backend`

4. 数据库初始化失败：
- 检查 `backend/migrations/*.sql` 是否自动执行
- 删除旧卷后重建（开发环境）：`docker compose down -v && docker compose up -d --build`

## 11. 验收步骤（本地）

1. 启动项目：`docker compose up -d --build`
2. 打开首页：`http://localhost`
3. 创建房间（5 分钟）
4. 两个浏览器分别进入同一房间
5. 分别发送消息
6. 触发 Bot 自动插话（可发“都行”“稳了”“?”）
7. 切换身份（normal/target/immune；target 二次确认）
8. 房主结束房间，查看战报页

---

如果你要继续迭代下一版，我建议优先加：

- 可配置敏感词后台字典
- Bot 模板可热更新
- 房间复盘导出（文本版）
- JWT 与多设备匿名会话合并
