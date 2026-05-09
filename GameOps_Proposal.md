# GameOps 项目方案书

## 项目定位

GameOps 是一个 Go 游戏运营支撑 / GM 管理 / SDK 侧配套服务。

它补齐 CoreRank 和 ArenaGate 没覆盖的运营侧能力：玩家状态管理、封禁、公告和维护配置、邮件奖励、CDK、数据事件、审计日志，以及 CoreRank 排行榜只读查询。

## 为什么需要 GameOps

线上游戏不仅需要匹配、排行榜和长连接，还需要运营后台：

- 玩家违规后需要封禁或解封。
- 活动开始时需要公告和开关。
- 线上事故后需要补偿邮件。
- 礼包码/CDK 需要防重复兑换。
- 后台写操作需要审计。
- 运营需要最小数据事件作为分析入口。

## V1+ 范围

已纳入：

- 管理员登录与最小 token。
- 玩家 seed、查询、封禁、解封。
- 实名/未成年/当日游戏时长字段模拟。
- 公告、活动、登录维护、排位维护配置。
- 公开运营状态接口。
- 邮件奖励与幂等领取。
- CDK 批次、查询和兑换防重复。
- 最小数据事件上报。
- 审计日志。
- CoreRank 只读联动。
- demo、测试、CI、README、API 文档和技术报告。

明确不做：

- 完整后台前端。
- 支付 SDK。
- 真实账号 SDK。
- 完整防沉迷系统。
- 背包、商城、订单和充值回调。
- Kafka / MQ / 大数据平台。
- 复杂 RBAC、动态菜单、多租户。
- GameOps 写入 CoreRank 排行榜。
- Kubernetes 和微服务拆分。

## 架构

```text
GM / Operator
  -> GameOps RESTful API
  -> Memory repository in V1+
  -> Future MySQL / Redis adapters

GameOps
  -> CoreRank read-only API
  -> /metrics
```

V1+ 使用内存仓库保证本地演示稳定；正式环境可替换为 MySQL 持久化和 Redis 幂等锁。
