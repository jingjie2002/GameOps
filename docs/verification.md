# GameOps 验证指南

## 基础验证

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
go test ./...
go vet ./...
go build -o tmp\gameops-server.exe ./cmd/server
```

## Demo 验证

```powershell
python scripts\demo_flow.py
```

脚本会自动：

1. 构建并启动 GameOps。
2. 管理员登录。
3. 创建测试玩家。
4. 封禁和解封玩家。
5. 修改排位维护配置。
6. 查询公开运营状态。
7. 预检奖励邮件。
8. 发送带有效期和 Agent 审计字段的奖励邮件。
9. 玩家领取邮件，并验证重复领取被阻止。
10. 创建 CDK 批次。
11. 玩家兑换 CDK，并验证重复兑换被阻止。
12. 冻结单个 CDK 与 CDK 批次，并验证冻结后不可兑换。
13. 上报数据事件。
14. 查询审计日志，并验证 Agent 审计字段。
15. 检查 `/metrics`。

## AI 风险分析 Demo

```powershell
python scripts\risk_ai_demo.py
```

脚本会自动：

1. 构建并启动 GameOps。
2. 构造高频奖励邮件、多次 CDK 兑换、频繁配置变更和多次封禁。
3. 调用 `POST /api/risk/analyze`。
4. 输出风险等级、规则命中、证据列表、mock-ai 中文摘要和建议排查步骤。

成功时应看到：

```text
risk_project=游戏运营日志 AI 风险分析助手
risk_level=high score=100 ai_provider=mock-ai
summary=mock-ai 摘要：当前分析窗口风险等级为 high...
GameOps AI risk demo completed
```

## V2.5 GM 安全能力验证点

- `POST /api/mails/preview` 返回 `allowed`、`risk_level`、`violations`、`expires_at`。
- `POST /api/mails` 支持 `expires_in_seconds`，响应中包含 `expires_at`。
- 过期邮件领取会返回错误。
- 超过金币、道具数量、目标数量或有效期范围时，预检返回 blocked。
- `POST /api/cdk/{code}/freeze` 可冻结单个 CDK。
- `POST /api/cdk/batches/{batch_id}/freeze` 可冻结批次内仍 active 的 CDK。
- 冻结后的 CDK 兑换返回错误。
- 审计日志可记录 `agent_session_id`、`agent_mode`、`confirmation_id`、`confirmed_by`、`confirmed_at`。

## 当前验证边界

- 当前使用内存仓库，适合本地 demo 和 CI。
- MySQL/Redis 是 V1+ 的正式演进接口方向，当前不要求本机必须安装数据库。
- 未做 Linux 部署验证。
- 未做真实后台前端验证。
