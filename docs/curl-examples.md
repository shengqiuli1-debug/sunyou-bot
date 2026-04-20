# Curl 示例（MVP）

以下默认后端地址 `http://127.0.0.1:8080`。

## 1. 创建匿名用户

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/users/guest \
  -H 'Content-Type: application/json' \
  -d '{"nickname":"阿强"}'
```

返回里取 `token`。

## 2. 创建房间

```bash
TOKEN=替换成token
curl -s -X POST http://127.0.0.1:8080/api/v1/rooms \
  -H "X-User-Token: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"durationMinutes":5,"botRole":"judge","fireLevel":"medium","generateReport":true}'
```

## 3. 加入房间（target 二次确认）

```bash
ROOM_ID=替换成room.id
curl -s -X POST http://127.0.0.1:8080/api/v1/rooms/$ROOM_ID/join \
  -H "X-User-Token: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"identity":"target","confirmTarget":true}'
```

## 4. 房主切换 Bot 角色

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/rooms/$ROOM_ID/control \
  -H "X-User-Token: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"action":"switch_role","value":"narrator"}'
```

## 5. 模拟充值

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/points/recharge \
  -H "X-User-Token: $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"amount":30,"channel":"mock_wechat"}'
```

## 6. 结束房间并查看战报

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/rooms/$ROOM_ID/end \
  -H "X-User-Token: $TOKEN"

curl -s http://127.0.0.1:8080/api/v1/rooms/$ROOM_ID/report \
  -H "X-User-Token: $TOKEN"
```

## 7. 查看 Bot 回复审计链路

```bash
# 最近审计记录（支持 replySource / botRole 过滤）
curl -s "http://127.0.0.1:8080/api/v1/debug/rooms/$ROOM_ID/bot-audits?page=1&pageSize=20" \
  -H "X-User-Token: $TOKEN"

# 根据消息 ID 反查关联 Bot 审计（触发消息或 Bot 回复消息都可）
MESSAGE_ID=替换成消息ID
curl -s "http://127.0.0.1:8080/api/v1/debug/messages/$MESSAGE_ID/bot-audit" \
  -H "X-User-Token: $TOKEN"

# 按审计 ID 查看详情
AUDIT_ID=替换成审计ID
curl -s "http://127.0.0.1:8080/api/v1/debug/bot-audits/$AUDIT_ID" \
  -H "X-User-Token: $TOKEN"
```
