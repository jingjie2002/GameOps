package gameops

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentCapabilities(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	req := httptest.NewRequest(http.MethodGet, "/api/agent/capabilities", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Project struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
		AgentTools  map[string]string `json:"agent_tools"`
		RiskAnalyze struct {
			Endpoint string `json:"endpoint"`
			Tool     string `json:"tool"`
			Status   string `json:"status"`
		} `json:"risk_analyze"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode capabilities: %v", err)
	}
	if payload.Project.ID != "gameops" || payload.Project.Name != "GameOps" {
		t.Fatalf("unexpected project identity: %#v", payload.Project)
	}
	if payload.AgentTools["gameops_analyze_gm_risk"] != "POST /api/risk/analyze" {
		t.Fatalf("risk analyze tool not declared: %#v", payload.AgentTools)
	}
	if payload.RiskAnalyze.Endpoint != "/api/risk/analyze" || payload.RiskAnalyze.Status != "retained" {
		t.Fatalf("unexpected risk analyze declaration: %#v", payload.RiskAnalyze)
	}
	if !containsString(payload.Capabilities, "gm_risk_analyze") {
		t.Fatalf("expected gm_risk_analyze capability, got %#v", payload.Capabilities)
	}
	if !containsString(payload.Capabilities, "cdk_freeze") || !containsString(payload.Capabilities, "agent_audit_fields") {
		t.Fatalf("expected V2.5 capabilities, got %#v", payload.Capabilities)
	}
	if !containsString(payload.Capabilities, "agent_events") || !containsString(payload.Capabilities, "agent_logs") {
		t.Fatalf("expected agent event/log capabilities, got %#v", payload.Capabilities)
	}
	if payload.AgentTools["gameops_preview_mail"] != "POST /api/mails/preview" {
		t.Fatalf("mail preview tool not declared: %#v", payload.AgentTools)
	}
	for tool, endpoint := range map[string]string{
		"gameops_send_mail":        "POST /api/mails",
		"gameops_ban_player":       "POST /api/players/{player_id}/ban",
		"gameops_unban_player":     "POST /api/players/{player_id}/unban",
		"gameops_freeze_cdk":       "POST /api/cdk/{code}/freeze",
		"gameops_freeze_cdk_batch": "POST /api/cdk/batches/{batch_id}/freeze",
	} {
		if payload.AgentTools[tool] != endpoint {
			t.Fatalf("%s tool not declared: %#v", tool, payload.AgentTools)
		}
	}
}

func TestAgentEventsAndLogs(t *testing.T) {
	store := NewMemoryStore()
	server := NewServer(ConfigFromEnv(), store)

	for _, path := range []string{"/api/agent/events", "/api/agent/logs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected status 200, got %d", path, rec.Code)
		}
		var payload map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("%s decode response: %v", path, err)
		}
		if payload["status"] != "ok" || payload["project"] != "gameops" {
			t.Fatalf("%s unexpected payload: %#v", path, payload)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
