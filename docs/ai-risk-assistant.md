# 游戏运营日志 AI 风险分析助手

## 项目定位

游戏运营日志 AI 风险分析助手是 GameOps V1.6 的 AI 落地扩展。实现上它仍然属于 GameOps 后端服务；简历呈现上可以作为一个独立小项目，用来说明“AI 大模型 / AI 编程工具 + 游戏运营后台 + 风险排查”的结合能力。

一句话说明：

```text
基于 GameOps 审计日志、CDK 兑换、奖励邮件、封禁和运营配置变更记录，先用规则识别异常运营行为，再用 mock-ai 生成中文排查摘要。
```

## 为什么这样设计

本项目没有做聊天机器人，也没有做 RAG、向量库或模型微调。原因是三七互娱后端 JD 更看重“AI 项目落地”和游戏业务理解。对运营后台来说，可信的路径不是让 AI 直接判断风险，而是：

1. 后端先沉淀结构化审计证据。
2. 规则引擎先给风险等级、命中规则和证据列表。
3. AI 层只负责把证据转成中文排查摘要。
4. 没有 API key 时默认使用 `mock-ai`，保证 demo 可以稳定运行。

这个设计能避免“AI 幻觉直接判案”，也更符合真实业务里的审计和质量管理思路。

## 当前 V1 能力

接口：

```http
POST /api/risk/analyze
```

输入：

```json
{
  "from_ms": 0,
  "to_ms": 0,
  "use_ai": true,
  "ai_provider": "mock-ai"
}
```

输出：

- `risk_level`：规则引擎风险等级。
- `score`：规则引擎风险分，最高 100。
- `findings`：命中规则、严重程度、证据列表和建议排查步骤。
- `summary`：mock-ai 生成的中文排查摘要。
- `ai_provider`：当前摘要提供方，默认 `mock-ai`。

## 风险规则

当前规则引擎覆盖 5 类最小可演示风险：

| 规则 | 说明 |
| --- | --- |
| `high_frequency_reward_mail` | 同一管理员短时间创建多封奖励邮件 |
| `high_value_reward_mail` | 单封奖励邮件金币数较高 |
| `repeated_cdk_redeem` | 同一玩家短时间多次兑换 CDK |
| `short_time_ban_burst` | 同一管理员短时间多次封禁玩家 |
| `frequent_ops_config_update` | 维护、公告、活动等运营配置频繁变更 |
| `sensitive_operation_burst` | 同一管理员集中执行多类敏感运营操作 |

## 本地演示

```powershell
$env:GOCACHE = Join-Path (Get-Location) ".gocache"
python scripts\risk_ai_demo.py
```

demo 会自动完成：

1. 构建并启动 GameOps。
2. 登录管理员账号。
3. 创建多封高额奖励邮件。
4. 创建 CDK 批次并让同一玩家多次兑换。
5. 多次切换排位维护配置。
6. 多次封禁同一玩家。
7. 调用 `/api/risk/analyze` 输出风险报告。

输出中会包含：

```text
risk_project=游戏运营日志 AI 风险分析助手
risk_level=high score=100 ai_provider=mock-ai
summary=mock-ai 摘要：当前分析窗口风险等级为 high...
findings:
- high high_frequency_reward_mail: ...
  evidence ...
  suggestion: ...
GameOps AI risk demo completed
```

## 简历叙事建议

项目名：

```text
游戏运营日志 AI 风险分析助手
```

推荐写法：

```text
基于 GameOps 运营后台审计日志实现 AI 风险分析助手，覆盖奖励邮件、CDK 兑换、封禁和运营配置变更等敏感操作；通过规则引擎输出风险等级、命中证据和建议排查步骤，并使用 mock-ai 生成中文排查摘要，保证无 API key 环境也可运行 demo。
```

面试讲法：

```text
这个项目不是做聊天机器人，而是把 AI 放在游戏运营后台的排查链路里。后端先通过审计日志和规则引擎保证证据可靠，AI 只负责把结构化证据整理成中文摘要，帮助运营或研发更快定位异常操作。
```

## 当前明确不做

- 不做后台前端。
- 不做聊天机器人。
- 不做 RAG、向量库或模型微调。
- 不接真实风控生产系统。
- 不把真实 API key 写入代码。
- 不让 AI 直接决定风险等级。
- 不替代运营审批和安全审计流程。
