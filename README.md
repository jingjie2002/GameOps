# GameOps

GameOps 是一个 Go 游戏运营支撑与 GM 管理服务示例项目。它负责游戏运营侧的基础能力，包括玩家状态、封禁与解封、公告和维护配置、奖励邮件、CDK、数据事件、审计日志，以及对 CoreRank 排行榜的只读查询。

GameOps 不实现完整游戏服务器，也不包含后台前端页面。它主要用于演示运营后台服务如何组织敏感操作、审计记录和最小联动接口。

## 功能

- 健康检查：`GET /healthz`
- Prometheus 指标：`GET /metrics`
- 管理员登录与最小 HMAC token
- 玩家 seed、列表、详情、封禁、解封
- 实名、未成年、当日游戏时长字段模拟
- 运营配置：公告、活动开关、登录维护、排位维护
- 公共运营状态：`GET /api/public/ops-state`
- 玩家封禁状态：`GET /api/public/players/{player_id}/state`
- 奖励邮件预检、发放、查询和领取
- 奖励策略校验：金币上限、目标数量上限、道具数量上限、有效期范围、封禁时长范围
- CDK 批次、单码查询、兑换和冻结
- 最小数据事件上报
- 审计日志查询
- CoreRank 只读联动：health、leaderboard、player rank
- 运营风险分析：基于审计日志和规则引擎生成风险摘要

## 技术栈

- Go 1.25
- RESTful HTTP
- MySQL 可选持久化
- Prometheus metrics
- HMAC token
- Python demo scripts

## 目录结构

```text
cmd/server/              服务入口
internal/gameops/        业务模型、接口、存储、策略和风险分析逻辑
scripts/                 本地演示脚本
docs/                    API、存储、验证和 Agent 接入文档
agent.yaml               Agent 能力声明
```

## 快速开始

### 1. 启动服务

默认使用内存存储，适合本地演示和测试。

```powershell
$env:GAMEOPS_ADDR = "127.0.0.1:18090"
go run ./cmd/server
```

默认管理员：

```text
username: admin
password: admin_demo
```

### 2. 启用 MySQL

如果需要启用 MySQL 持久化：

```powershell
$env:GAMEOPS_MYSQL_DSN = "gameops:<password>@tcp(127.0.0.1:3306)/gameops?parseTime=true&charset=utf8mb4&loc=Local"
go run ./cmd/server
```

表结构位置：

```text
internal/gameops/mysql_schema.sql
```

既有数据库升级可参考：

```text
docs/mysql-migrations-v2.5.sql
```

## Demo

完整业务流程演示：

```powershell
python scripts\demo_flow.py
```

脚本会覆盖管理员登录、玩家封禁与解封、邮件预检与发放、邮件领取、重复领取拦截、CDK 创建与兑换、重复兑换拦截、CDK 冻结和审计字段记录。

运营风险分析演示：

```powershell
python scripts\risk_ai_demo.py
```

风险分析接口：

```http
POST /api/risk/analyze
```

示例请求：

```json
{"use_ai": true}
```

响应包含：

- `risk_level`：`normal` / `low` / `medium` / `high`
- `score`：规则引擎风险分
- `findings`：命中规则、证据列表和建议排查步骤
- `summary`：本地 mock 摘要
- `ai_provider`：默认 `mock-ai`

## 验证

推荐在修改后执行：

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
go test ./...
go vet ./...
go build -o tmp\gameops-server.exe ./cmd/server
python scripts\demo_flow.py
python scripts\risk_ai_demo.py
```

## Agent 接入

GameOps 提供 Agent-ready 接入口，供外部工具读取项目声明、健康状态和能力表，并在受控条件下调用 GM 工具。

| 能力 | 入口 |
|---|---|
| 项目声明 | `agent.yaml` |
| 健康检查 | `GET /healthz` |
| 能力声明 | `GET /api/agent/capabilities` |
| Agent events | `GET /api/agent/events` |
| Agent logs | `GET /api/agent/logs` |
| 邮件预检 | `POST /api/mails/preview` |
| 确认后发邮件 | `POST /api/mails` |
| 确认后封禁/解封 | `POST /api/players/{player_id}/ban`、`POST /api/players/{player_id}/unban` |
| CDK 冻结 | `POST /api/cdk/batches/{batch_id}/freeze`、`POST /api/cdk/{code}/freeze` |
| Agent smoke test | `python scripts\agent_smoke.py` |

离线检查：

```powershell
python scripts\agent_smoke.py --offline
```

服务启动后检查：

```powershell
python scripts\agent_smoke.py --base-url http://127.0.0.1:18090
```

## 文档

- [API 文档](docs/api.md)
- [Agent 接入说明](docs/agent-integration.md)
- [运营日志风险分析说明](docs/ai-risk-assistant.md)
- [存储设计](docs/storage.md)
- [验证指南](docs/verification.md)

## 当前限制

- 不包含完整后台前端。
- 不接入真实支付 SDK 或账号 SDK。
- 不包含完整防沉迷系统。
- 不包含背包、商城、订单和充值回调。
- 不包含 Kafka、MQ 或大数据平台。
- 不包含复杂 RBAC、动态菜单和多租户。
- 不写入 CoreRank 排行榜，只读取 CoreRank 状态。
- 风险分析默认使用本地规则和 mock 摘要，不依赖真实大模型服务。

## License

未指定。
