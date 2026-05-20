# GameOps Agent 接入说明

GameOps 已补齐 Agent-ready 基础能力，供后续 `GameServerProjectAgent` 通过统一规范识别、审查和诊断 GM 管理服务。

## 接入入口

| 项目 | 路径 |
|---|---|
| 项目声明 | `agent.yaml` |
| 健康检查 | `GET /healthz` |
| 能力声明 | `GET /api/agent/capabilities` |
| 指标 | `GET /metrics` |
| 风险分析 | `POST /api/risk/analyze` |
| 邮件预检 | `POST /api/mails/preview` |
| 确认后发邮件 | `POST /api/mails` |
| 确认后封禁/解封 | `POST /api/players/{player_id}/ban`、`POST /api/players/{player_id}/unban` |
| CDK 冻结 | `POST /api/cdk/batches/{batch_id}/freeze`、`POST /api/cdk/{code}/freeze` |
| Agent events | `GET /api/agent/events` |
| Agent logs | `GET /api/agent/logs` |
| Smoke test | `python scripts/agent_smoke.py` |

## Agent 可做

- 读取 `agent.yaml`、README、API 文档和验证文档。
- 调用 `/healthz` 判断 GameOps 是否在线。
- 调用 `/api/agent/capabilities` 获取 GM、审计、CDK、邮件和风险分析能力表。
- 在自动审查模式运行 `go test ./...`、`go vet ./...`、`python scripts/demo_flow.py`、`python scripts/risk_ai_demo.py` 和 Agent smoke test。
- 将现有 `POST /api/risk/analyze` 作为 Agent 工具 `gameops_analyze_gm_risk` 复用，分析奖励邮件、CDK、封禁和运营配置变更风险。
- 调用 `POST /api/mails/preview` 生成邮件预检结果，不落库、不发奖。
- 在完全访问权限和人工确认后，通过 GameOps 白名单接口发送带有效期邮件、封禁/解封、冻结 CDK，并写入 Agent 审计字段。

## Agent 不默认做

- 不绕过 GameOps 直接写数据库。
- 不无确认发送奖励邮件或冻结 CDK。
- 不无确认封禁或解封玩家。
- 不无确认修改运营配置。
- 不自动部署或重启生产服务。

阶段 4 已补 GameOps V2.5 GM 安全基础能力：邮件有效期、邮件预检、奖励限制、封禁时长限制、CDK 冻结和 Agent 审计字段。阶段 5 已由 `GameServerProjectAgent` 通过 `gsa gm` 接入确认后发邮件、封禁/解封和冻结 CDK。

## Agent 审计字段

写接口可在 JSON 请求体或 HTTP Header 中带上：

```text
agent_session_id / X-Agent-Session-ID
agent_mode / X-Agent-Mode
confirmation_id / X-Agent-Confirmation-ID
confirmed_by / X-Agent-Confirmed-By
confirmed_at / X-Agent-Confirmed-At
```

## 本地验证

服务运行后：

```powershell
python scripts\agent_smoke.py
```

只验证本地声明文件，不要求服务已启动：

```powershell
python scripts\agent_smoke.py --offline
```
