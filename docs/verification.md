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
7. 发奖励邮件。
8. 玩家领取邮件，并验证重复领取被阻止。
9. 创建 CDK 批次。
10. 玩家兑换 CDK，并验证重复兑换被阻止。
11. 上报数据事件。
12. 查询审计日志。
13. 检查 `/metrics`。

## 当前验证边界

- 当前使用内存仓库，适合本地 demo 和 CI。
- MySQL/Redis 是 V1+ 的正式演进接口方向，当前不要求本机必须安装数据库。
- 未做 Linux 部署验证。
- 未做真实后台前端验证。
