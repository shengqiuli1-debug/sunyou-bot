# 架构说明（MVP）

- 单仓库：`frontend` + `backend`。
- 后端单体（Gin + WebSocket + Redis + PostgreSQL）。
- Redis 保存房间实时状态、消息缓存、Bot 冷却、TTL 状态。
- PostgreSQL 保存用户、点数流水、房间记录、成员记录、消息、战报、举报、风控日志。
- Bot 采用规则+模板引擎，预留 AI Provider 接口。
- 房间过期由后端 scheduler 每 5 秒扫描并自动结算。
