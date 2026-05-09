# GameOps API

## 基础

```http
GET /healthz
GET /metrics
```

## 管理员

```http
POST /api/admin/login
GET /api/admin/me
```

登录请求：

```json
{"username":"admin","password":"admin_demo"}
```

除公开接口外，后台接口需要：

```text
Authorization: Bearer <token>
```

## 玩家

```http
POST /api/players/seed
GET  /api/players
GET  /api/players/{player_id}
POST /api/players/{player_id}/ban
POST /api/players/{player_id}/unban
```

封禁请求：

```json
{"reason":"abuse_report","banned_seconds":3600}
```

## 运营配置

```http
GET /api/ops-configs
PUT /api/ops-configs/{config_key}
GET /api/public/ops-state
GET /api/public/players/{player_id}/state
```

配置请求：

```json
{"config_value":"true","description":"ranked queue closed for maintenance"}
```

玩家公开状态仅返回网关需要的封禁字段：

```json
{"player_id":"player_1001","status":"banned","ban_reason":"abuse_report","banned_until":1770000000000}
```

## 邮件奖励

```http
POST /api/mails
GET  /api/players/{player_id}/mails
POST /api/players/{player_id}/mails/{mail_id}/claim
```

发邮件请求：

```json
{
  "player_id": "player_1001",
  "title": "SS25 ranked reward",
  "body": "Season compensation",
  "gold": 500,
  "items": ["skin_trial"]
}
```

## CDK

```http
POST /api/cdk/batches
GET  /api/cdk/batches
GET  /api/cdk/{code}
POST /api/cdk/{code}/redeem
```

创建批次：

```json
{
  "name": "launch gift",
  "gold": 300,
  "items": ["ticket"],
  "count": 1,
  "max_uses_per_code": 1,
  "expires_in_seconds": 3600
}
```

兑换：

```json
{"player_id":"player_1002"}
```

## 数据事件与审计

```http
POST /api/events
GET  /api/audit-logs?admin_id=admin&action=player.ban&target_type=player&target_id=player_1003
```

事件示例：

```json
{
  "type": "reward_claim",
  "player_id": "player_1001",
  "payload": {"source": "demo_flow"}
}
```

## CoreRank 只读联动

```http
GET /api/integrations/corerank/health
GET /api/integrations/corerank/leaderboard?leaderboard_type=season:ss25&limit=10
GET /api/integrations/corerank/players/{player_id}/rank?leaderboard_type=season:ss25
```

CoreRank 不可用时，GameOps 返回 `status=unavailable`，主功能不受影响。
