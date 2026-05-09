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

## 当前 V1+ 能力

- `/healthz` 健康检查。
- `/metrics` Prometheus 文本指标。
- 管理员登录与最小 HMAC token。
- 玩家 seed、列表、详情、封禁、解封。
- 实名/未成年/当日游戏时长字段模拟。
- 运营配置：公告、活动开关、登录维护、排位维护。
- `GET /api/public/ops-state` 给网关或客户端读取当前运营状态。
- `GET /api/public/players/{player_id}/state` 给网关读取玩家封禁状态。
- 奖励邮件发放、查询、领取，重复领取不会重复加金币。
- CDK 批次、单码查询、兑换，重复兑换会被阻止。
- 最小数据事件上报。
- 审计日志查询。
- CoreRank 只读联动：health、leaderboard、player rank。

## 快速验证

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
go test ./...
go vet ./...
go build -o tmp\gameops-server.exe ./cmd/server
python scripts\demo_flow.py
```

demo 成功时能看到：

```text
admin_login role=admin
seed_players count=3
ban_player player_1003 status=banned
unban_player player_1003 status=normal
create_mail mail_...
claim_mail mail_... status=claimed
duplicate_claim blocked=mail already claimed
create_cdk_batch batch_... code=GO-batch_...-0001
redeem_cdk ...
duplicate_redeem blocked=cdk already redeemed by player
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
- 不做 Kubernetes 和微服务拆分。

## 文档

- [API 文档](docs/api.md)
- [存储设计](docs/storage.md)
- [验证指南](docs/verification.md)
- [项目方案书](GameOps_Proposal.md)
- [技术报告](GameOps_Technical_Report.md)
