# GameOps 存储设计

## MySQL

GameOps V1.5 使用 MySQL 承载运营侧持久化数据：

- `players`：玩家基础状态和封禁状态。
- `player_status_logs`：封禁/解封状态变更记录。
- `ops_configs`：公告、活动、维护等运营配置。
- `mails`：奖励邮件。
- `cdk_batches`：CDK 批次。
- `cdks`：单个兑换码状态。
- `cdk_redemptions`：兑换记录，使用 `(code, player_id)` 唯一约束防止同一玩家重复兑换。
- `game_events`：最小运营事件。
- `audit_logs`：后台写操作审计日志。

表结构文件：

```text
internal/gameops/mysql_schema.sql
```

## Redis

GameOps 的 Redis 设计保持克制，优先覆盖热点配置、幂等和限流：

```text
gameops:ops_state
gameops:lock:cdk:{code}:{player_id}
gameops:rate_limit:admin:{user_id}
```

V1.5 先以 MySQL repository 和文档化 key 设计为主，后续再把 CDK 兑换锁和运营配置缓存接入 Redis。

## Store 抽象

`Store` 接口用于隔离 HTTP handler 与具体存储实现：

```text
MemoryStore：本地 demo / CI fallback
MySQLStore：生产化雏形
```

默认不配置 `GAMEOPS_MYSQL_DSN` 时使用 `MemoryStore`。配置 DSN 后使用 `MySQLStore`。
