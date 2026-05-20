package gameops

import "net/http"

func (s *Server) handleAgentCapabilities(w http.ResponseWriter, _ *http.Request) {
	writeTools := []map[string]string{
		{"endpoint": "POST /api/mails/preview", "mode": "默认权限", "note": "read-only preview for agent confirmation"},
		{"endpoint": "POST /api/mails", "mode": "完全访问权限", "note": "requires explicit confirmation before production use"},
		{"endpoint": "POST /api/players/{player_id}/ban", "mode": "完全访问权限", "note": "requires explicit confirmation"},
		{"endpoint": "POST /api/players/{player_id}/unban", "mode": "完全访问权限", "note": "requires explicit confirmation"},
		{"endpoint": "PUT /api/ops-configs/{config_key}", "mode": "完全访问权限", "note": "requires explicit confirmation"},
		{"endpoint": "POST /api/cdk/batches", "mode": "完全访问权限", "note": "requires explicit confirmation before production use"},
		{"endpoint": "POST /api/cdk/batches/{batch_id}/freeze", "mode": "完全访问权限", "note": "requires explicit confirmation"},
		{"endpoint": "POST /api/cdk/{code}/freeze", "mode": "完全访问权限", "note": "requires explicit confirmation"},
	}

	writeJSON := map[string]any{
		"agent_ready_version": "1.0",
		"project": map[string]string{
			"id":          "gameops",
			"name":        "GameOps",
			"type":        "go-service",
			"description": "Game operations and GM management backend service.",
		},
		"health": map[string]string{
			"primary": "/healthz",
		},
		"metrics": map[string]string{
			"prometheus": "/metrics",
			"default":    "http://127.0.0.1:18090/metrics",
		},
		"commands": []map[string]string{
			{"name": "test", "command": "go test ./...", "mode": "自动审查"},
			{"name": "vet", "command": "go vet ./...", "mode": "自动审查"},
			{"name": "demo_flow", "command": "python scripts/demo_flow.py", "mode": "自动审查"},
			{"name": "risk_ai_demo", "command": "python scripts/risk_ai_demo.py", "mode": "自动审查"},
			{"name": "agent_smoke", "command": "python scripts/agent_smoke.py", "mode": "自动审查"},
		},
		"capabilities": []string{
			"health",
			"metrics",
			"go_test",
			"go_vet",
			"admin_login",
			"player_query",
			"mail_query",
			"cdk_query",
			"audit_log_query",
			"ops_config_query",
			"gm_risk_analyze",
			"mail_preview",
			"mail_expiration",
			"reward_limit_policy",
			"cdk_freeze",
			"agent_audit_fields",
			"corerank_readonly_integration",
			"agent_events",
			"agent_logs",
		},
		"agent_tools": map[string]string{
			"gameops_analyze_gm_risk":  "POST /api/risk/analyze",
			"gameops_preview_mail":     "POST /api/mails/preview",
			"gameops_send_mail":        "POST /api/mails",
			"gameops_ban_player":       "POST /api/players/{player_id}/ban",
			"gameops_unban_player":     "POST /api/players/{player_id}/unban",
			"gameops_freeze_cdk":       "POST /api/cdk/{code}/freeze",
			"gameops_freeze_cdk_batch": "POST /api/cdk/batches/{batch_id}/freeze",
		},
		"read_tools": []string{
			"GET /healthz",
			"GET /metrics",
			"GET /api/agent/events",
			"GET /api/agent/logs",
			"GET /api/players",
			"GET /api/players/{player_id}",
			"GET /api/ops-configs",
			"GET /api/audit-logs",
			"POST /api/risk/analyze",
			"GET /api/integrations/corerank/health",
			"GET /api/integrations/corerank/leaderboard",
			"GET /api/integrations/corerank/players/{player_id}/rank",
		},
		"write_tools": writeTools,
		"forbidden": []string{
			"direct_database_write",
			"direct_database_delete",
			"gm_write_without_confirm",
			"bulk_reward_without_confirm",
			"production_deploy_without_confirm",
		},
		"risk_analyze": map[string]string{
			"endpoint": "/api/risk/analyze",
			"tool":     "gameops_analyze_gm_risk",
			"status":   "retained",
			"provider": s.cfg.RiskAIProvider,
		},
		"docs": []string{
			"README.md",
			"docs/api.md",
			"docs/verification.md",
			"docs/ai-risk-assistant.md",
			"docs/agent-integration.md",
			"docs/mysql-migrations-v2.5.sql",
		},
	}

	s.writeJSON(w, http.StatusOK, writeJSON)
}

func (s *Server) handleAgentEvents(w http.ResponseWriter, _ *http.Request) {
	audits := recentAudits(s.store.ListAudits(AuditFilter{}), 20)
	events := make([]map[string]any, 0, len(audits))
	for _, audit := range audits {
		events = append(events, map[string]any{
			"type":             "audit_log",
			"action":           audit.Action,
			"target_type":      audit.TargetType,
			"target_id":        audit.TargetID,
			"agent_session_id": audit.AgentSessionID,
			"confirmation_id":  audit.ConfirmationID,
			"created_at":       audit.CreatedAt,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"project": "gameops",
		"stats":   s.store.Stats(),
		"events":  events,
	})
}

func (s *Server) handleAgentLogs(w http.ResponseWriter, _ *http.Request) {
	audits := recentAudits(s.store.ListAudits(AuditFilter{}), 20)
	logs := make([]map[string]any, 0, len(audits))
	for _, audit := range audits {
		logs = append(logs, map[string]any{
			"type":             "audit",
			"action":           audit.Action,
			"target_type":      audit.TargetType,
			"target_id":        audit.TargetID,
			"request_id":       audit.RequestID,
			"agent_session_id": audit.AgentSessionID,
			"created_at":       audit.CreatedAt,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"project": "gameops",
		"stats":   s.store.Stats(),
		"logs":    logs,
		"note":    "agent logs are summarized from audit_logs; use GET /api/audit-logs with an admin/auditor token for full details.",
	})
}

func recentAudits(audits []AuditLog, limit int) []AuditLog {
	if limit <= 0 || len(audits) <= limit {
		return audits
	}
	return audits[len(audits)-limit:]
}
