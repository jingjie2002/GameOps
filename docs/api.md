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

`banned_seconds` 范围：`60` 到 `2592000`。

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
POST /api/mails/preview
POST /api/mails
GET  /api/players/{player_id}/mails
POST /api/players/{player_id}/mails/{mail_id}/claim
```

邮件预检请求：

```json
{
  "player_id": "player_1001",
  "title": "SS25 ranked reward",
  "body": "Season compensation",
  "gold": 500,
  "items": ["skin_trial"],
  "expires_in_seconds": 3600
}
```

邮件预检响应：

```json
{
  "allowed": true,
  "risk_level": "low",
  "target_count": 1,
  "gold": 500,
  "items": ["skin_trial"],
  "expires_in_seconds": 3600,
  "expires_at": 1770000000000,
  "violations": [],
  "warnings": []
}
```

发邮件请求：

```json
{
  "player_id": "player_1001",
  "title": "SS25 ranked reward",
  "body": "Season compensation",
  "gold": 500,
  "items": ["skin_trial"],
  "expires_in_seconds": 3600,
  "agent_session_id": "sess_demo_001",
  "agent_mode": "完全访问权限",
  "confirmation_id": "confirm_demo_mail",
  "confirmed_by": "demo_user",
  "confirmed_at": 1770000000000
}
```

安全限制：

- `gold` 单封上限：`5000`。
- `items` 数量上限：`10`。
- 预检 `player_ids` 目标数量上限：`100`。
- `expires_in_seconds` 范围：`60` 到 `2592000`；未传时默认 `604800`。
- 过期邮件不可领取。

## CDK

```http
POST /api/cdk/batches
GET  /api/cdk/batches
GET  /api/cdk/{code}
POST /api/cdk/batches/{batch_id}/freeze
POST /api/cdk/{code}/freeze
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

冻结后的 CDK 不可兑换。

## 数据事件与审计

```http
POST /api/events
GET  /api/audit-logs?admin_id=admin&action=player.ban&target_type=player&target_id=player_1003
POST /api/risk/analyze
```

带请求体的后台写接口可通过 JSON 传入 Agent 审计字段；所有后台写接口均可通过 HTTP Header 传入。JSON 字段：

```json
{
  "agent_session_id": "sess_demo_001",
  "agent_mode": "完全访问权限",
  "confirmation_id": "confirm_demo_mail",
  "confirmed_by": "demo_user",
  "confirmed_at": 1770000000000
}
```

Header 字段：

```text
X-Agent-Session-ID: sess_demo_001
X-Agent-Mode: full-access
X-Agent-Confirmation-ID: confirm_demo_mail
X-Agent-Confirmed-By: demo_user
X-Agent-Confirmed-At: 1770000000000
```

事件示例：

```json
{
  "type": "reward_claim",
  "player_id": "player_1001",
  "payload": {"source": "demo_flow"}
}
```

风险分析请求：

```json
{
  "from_ms": 0,
  "to_ms": 0,
  "use_ai": true,
  "ai_provider": "mock-ai"
}
```

说明：

- `from_ms` / `to_ms` 为空或为 `0` 时分析当前审计日志中可见的全部窗口。
- `use_ai` 默认为 `true`。
- 当前 demo 默认只启用 `mock-ai`，不需要 API key，不调用外部模型。
- 风险等级由规则引擎生成，AI 只负责中文排查摘要。

风险分析响应示例：

```json
{
  "project_name": "游戏运营日志 AI 风险分析助手",
  "risk_level": "high",
  "score": 100,
  "summary": "mock-ai 摘要：当前分析窗口风险等级为 high，规则引擎命中 5 类异常...",
  "suggestions": ["核对奖励邮件是否对应活动补偿审批单..."],
  "findings": [
    {
      "type": "high_frequency_reward_mail",
      "severity": "medium",
      "score": 28,
      "reason": "管理员 admin 在分析窗口内创建了 4 封奖励邮件，存在误发或越权发奖风险",
      "evidence": [
        {
          "audit_id": "audit_000001",
          "action": "mail.create",
          "admin_id": "admin",
          "target_type": "mail",
          "target_id": "mail_000001",
          "message": "奖励邮件 mail_000001 发给玩家 player_1001，金币 1200"
        }
      ],
      "suggestion": "核对奖励邮件是否对应活动补偿审批单，抽查目标玩家名单、金币数量和操作来源 IP。"
    }
  ],
  "ai_provider": "mock-ai"
}
```

## CoreRank 只读联动

```http
GET /api/integrations/corerank/health
GET /api/integrations/corerank/leaderboard?leaderboard_type=season:ss25&limit=10
GET /api/integrations/corerank/players/{player_id}/rank?leaderboard_type=season:ss25
```

CoreRank 不可用时，GameOps 返回 `status=unavailable`，主功能不受影响。
