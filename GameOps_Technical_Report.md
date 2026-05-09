# GameOps 技术报告

## 摘要

GameOps 是一个 Go 游戏运营支撑服务，面向 GM 和运营后台场景。当前 V1+ 已完成 RESTful API、管理员 token、玩家状态管理、运营配置、奖励邮件、CDK、数据事件、审计日志、metrics、demo、测试和 CI。

## 关键设计

### 1. 后台写操作审计

玩家 seed、封禁、解封、配置修改、邮件创建、邮件领取、CDK 批次创建、CDK 兑换都会写入审计日志。审计字段包括管理员/操作者、动作、目标类型、目标 ID、前后摘要、request_id、IP 和时间。

### 2. 奖励邮件幂等领取

邮件领取会检查邮件状态。第一次领取会把状态改为 `claimed` 并给玩家加金币；重复领取返回冲突，不会重复加金币。

### 3. CDK 防重复兑换

V1+ 使用内存锁定语义模拟 Redis 锁和唯一兑换记录。玩家重复兑换同一 CDK 会被拒绝。正式环境可替换为：

```text
SET gameops:lock:cdk:{code}:{player_id} NX EX
UNIQUE(code, player_id)
```

### 4. 运营配置公开状态

`GET /api/public/ops-state` 暴露公告、活动、登录维护、排位维护等状态。ArenaGate 或客户端可以读取该接口后决定是否推送公告或拒绝入口。

### 5. CoreRank 只读联动

GameOps 只读查询 CoreRank health、leaderboard 和 player rank，不写入 CoreRank 排行榜，避免职责混乱。

## 验证

```powershell
go test ./...
go vet ./...
go build -o tmp\gameops-server.exe ./cmd/server
python scripts\demo_flow.py
```

## 当前边界

- 当前使用内存仓库，不声明生产持久化。
- 当前没有完整后台前端。
- 当前没有支付、账号、完整防沉迷和大数据链路。
- 当前没有 Kubernetes 或微服务拆分。
