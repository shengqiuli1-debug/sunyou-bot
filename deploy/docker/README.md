# Docker Notes

- `docker-compose.yml` 在仓库根目录。
- 前端镜像由 `frontend/Dockerfile` 负责构建并内置 Nginx。
- `deploy/nginx/default.conf` 已配置 SPA 路由回退与 WebSocket 代理。
