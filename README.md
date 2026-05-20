# GameOps

GameOps 是一个 Go 游戏运营支撑 / GM 管理服务示例项目。

它不是完整游戏服务器，也不是后台前端。它负责游戏上线后的运营侧能力：玩家状态、封禁/解封、公告和维护配置、奖励邮件、CDK、数据事件、审计日志，以及对 CoreRank 排行榜的只读查询。

## 它在游戏里控制什么

```text
玩家客户端 / 网关
  -> ArenaGate：长连接、公告推送、维护态入场控制
  -> CoreRank：匹配、排行榜、赛季榜、房间分配
  -> GameOps：GM 操作、运营配置、发奖、CDK、审计
```

GameOps 控制的是：

- 玩家是否被封禁。
- 当前公告、活动开关和维护开关。
- GM 是否给玩家发了奖励邮件。
- 玩家是否已经领取邮件或兑换 CDK。
- 后台写操作是否有审计记录。
- 最小运营数据事件是否被接收。

GameOps 不控制的是：

- 角色移动、技能、伤害和战斗同步。
- CoreRank 的匹配算法和排行榜写入。
- ArenaGate 的 WebSocket session。
- 支付 SDK、账号 SDK 和完整防沉迷系统。

## 当前 V2.5 安全基础能力

- `/healthz` 健康检查。
- `/metrics` Prometheus 文本指标。
- 管理员登录与最小 HMAC token。
- 玩家 seed、列表、详情、封禁、解封。
- 实名/未成年/当日游戏时长字段模拟。
- 运营配置：公告、活动开关、登录维护、排位维护。
- `GET /api/public/ops-state` 给网关或客户端读取当前运营状态。
- `GET /api/public/players/{player_id}/state` 给网关读取玩家封禁状态。
- 奖励邮件预检、发放、查询、领取，支持 `expires_in_seconds` 和 `expires_at`，重复领取和过期领取会被阻止。
- 奖励上限策略：单封邮件金币上限、目标数量上限、道具数量上限、邮件有效期范围、封禁时长范围。
- CDK 批次、单码查询、兑换、冻结，重复兑换和冻结后兑换会被阻止。
- 最小数据事件上报。
- 审计日志查询，支持记录 `agent_session_id`、`agent_mode`、`confirmation_id`、`confirmed_by`、`confirmed_at`。
- CoreRank 只读联动：health、leaderboard、player rank。

## V1.6：游戏运营日志 AI 风险分析助手

这是 GameOps 的小型 AI 落地扩展，也可以作为简历中的独立项目呈现：**游戏运营日志 AI 风险分析助手**。

它基于现有审计日志，把奖励邮件、CDK 兑换、封禁、运营配置变更等敏感操作整理成风险证据。系统先用规则引擎计算风险等级和命中规则，再由 `mock-ai` 生成中文排查摘要。默认不需要 API key，也不依赖外部大模型服务，适合本地 demo、截图和面试讲解。

核心接口：

```http
POST /api/risk/analyze
```

示例请求：

```json
{"use_ai": true}
```

示例响应会包含：

- `risk_level`：`normal` / `low` / `medium` / `high`
- `score`：规则引擎风险分
- `findings`：命中规则、证据列表和建议排查步骤
- `summary`：`mock-ai` 生成的中文排查摘要
- `ai_provider`：默认 `mock-ai`

本地演示：

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
python scripts\risk_ai_demo.py
```

文档见：[游戏运营日志 AI 风险分析助手](docs/ai-risk-assistant.md)。

## Agent 接入

GameOps 已补充 Agent-ready 基础入口，供后续 `GameServerProjectAgent` 读取项目声明、健康状态和能力表。

| 能力 | 入口 |
|---|---|
| 项目声明 | `agent.yaml` |
| 健康检查 | `GET /healthz` |
| 能力声明 | `GET /api/agent/capabilities` |
| Agent events | `GET /api/agent/events` |
| Agent logs | `GET /api/agent/logs` |
| Agent 工具 | `gameops_analyze_gm_risk`、`gameops_preview_mail`、`gameops_send_mail`、`gameops_ban_player`、`gameops_unban_player`、`gameops_freeze_cdk` |
| 邮件预检 | `POST /api/mails/preview` |
| 确认后发邮件 | `POST /api/mails` |
| 确认后封禁/解封 | `POST /api/players/{player_id}/ban`、`POST /api/players/{player_id}/unban` |
| CDK 冻结 | `POST /api/cdk/batches/{batch_id}/freeze`、`POST /api/cdk/{code}/freeze` |
| Agent smoke test | `python scripts\agent_smoke.py` |

离线检查 `agent.yaml`：

```powershell
python scripts\agent_smoke.py --offline
```

服务启动后检查 Agent 接入口：

```powershell
python scripts\agent_smoke.py --base-url http://127.0.0.1:18090
```

阶段 4 已补 GameOps V2.5 GM 安全基础能力；Agent 侧自然语言确认和工具调用留到 `GameServerProjectAgent` 阶段 5。详细说明见：[Agent 接入说明](docs/agent-integration.md)。

## 快速验证

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
go test ./...
go vet ./...
go build -o tmp\gameops-server.exe ./cmd/server
python scripts\demo_flow.py
python scripts\risk_ai_demo.py
```

demo 成功时能看到：

```text
admin_login role=admin
seed_players count=3
ban_player player_1003 status=banned
unban_player player_1003 status=normal
create_mail mail_...
preview_mail allowed=True risk=low ...
claim_mail mail_... status=claimed
duplicate_claim blocked=mail already claimed
create_cdk_batch batch_... code=GO-batch_...-0001
redeem_cdk ...
duplicate_redeem blocked=cdk already redeemed by player
freeze_cdk ... status=frozen
agent_audit_fields recorded
GameOps demo completed
```

## 手动启动

```powershell
$env:GAMEOPS_ADDR = "127.0.0.1:18090"
go run ./cmd/server
```

默认使用内存仓库，适合本地 demo 和 CI。如果需要启用 MySQL：

```powershell
$env:GAMEOPS_MYSQL_DSN = "gameops:<password>@tcp(127.0.0.1:3306)/gameops?parseTime=true&charset=utf8mb4&loc=Local"
go run ./cmd/server
```

MySQL 表结构见：

```text
internal/gameops/mysql_schema.sql
```

既有 MySQL 库升级到 V2.5 时，可参考迁移脚本：

```text
docs/mysql-migrations-v2.5.sql
```

默认管理员：

```text
username: admin
password: admin_demo
```

## 当前明确不做

- 不做完整后台前端。
- 不做支付 SDK。
- 不做真实账号 SDK。
- 不做完整防沉迷系统。
- 不做背包、商城、订单和充值回调。
- 不做 Kafka / MQ / 大数据平台。
- 不做复杂 RBAC、动态菜单和多租户。
- 不做 GameOps 写入 CoreRank 排行榜。
- 不做 RAG、向量库、模型微调或聊天机器人。
- 不把 AI 风险助手做成真实风控生产系统。
- 不做 Kubernetes 和微服务拆分。

## 文档

- [API 文档](docs/api.md)
- [Agent 接入说明](docs/agent-integration.md)
- [游戏运营日志 AI 风险分析助手](docs/ai-risk-assistant.md)
- [存储设计](docs/storage.md)
- [验证指南](docs/verification.md)
- [项目方案书](GameOps_Proposal.md)
- [技术报告](GameOps_Technical_Report.md)
