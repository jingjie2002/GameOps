package gameops

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGameOpsRewardAndAuditFlow(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)

	var mail Mail
	doJSON(t, http.MethodPost, httpServer.URL+"/api/mails", token, map[string]any{
		"player_id": "player_1001",
		"title":     "Season reward",
		"body":      "SS25 ranked reward",
		"gold":      500,
		"items":     []string{"skin_trial"},
	}, http.StatusCreated, &mail)
	if mail.MailID == "" {
		t.Fatal("expected mail id")
	}

	var claimed Mail
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/player_1001/mails/"+mail.MailID+"/claim", "", map[string]any{}, http.StatusOK, &claimed)
	if claimed.Status != "claimed" {
		t.Fatalf("expected claimed mail, got %#v", claimed)
	}
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/player_1001/mails/"+mail.MailID+"/claim", "", map[string]any{}, http.StatusConflict, nil)

	var player Player
	doJSON(t, http.MethodGet, httpServer.URL+"/api/players/player_1001", token, nil, http.StatusOK, &player)
	if player.Gold != 1500 {
		t.Fatalf("expected player gold 1500 after one claim, got %d", player.Gold)
	}

	var audits []AuditLog
	doJSON(t, http.MethodGet, httpServer.URL+"/api/audit-logs", token, nil, http.StatusOK, &audits)
	if len(audits) < 2 {
		t.Fatalf("expected audit logs, got %d", len(audits))
	}
}

func TestCDKRedeemIsSingleUsePerPlayer(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)

	var batch CDKBatch
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/batches", token, map[string]any{
		"name":               "launch gift",
		"gold":               300,
		"items":              []string{"ticket"},
		"count":              1,
		"max_uses_per_code":  1,
		"expires_in_seconds": 3600,
	}, http.StatusCreated, &batch)
	if len(batch.Codes) != 1 {
		t.Fatalf("expected one code, got %#v", batch.Codes)
	}

	var redemption CDKRedemption
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/"+batch.Codes[0]+"/redeem", "", map[string]any{
		"player_id": "player_1002",
	}, http.StatusOK, &redemption)
	if redemption.RewardGold != 300 {
		t.Fatalf("unexpected redemption: %#v", redemption)
	}
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/"+batch.Codes[0]+"/redeem", "", map[string]any{
		"player_id": "player_1002",
	}, http.StatusConflict, nil)
}

func TestOpsStateAndBanFlow(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)
	doJSON(t, http.MethodPut, httpServer.URL+"/api/ops-configs/ranked_maintenance", token, map[string]any{
		"config_value": "true",
		"description":  "ranked queue closed for maintenance",
	}, http.StatusOK, nil)

	var state map[string]string
	doJSON(t, http.MethodGet, httpServer.URL+"/api/public/ops-state", "", nil, http.StatusOK, &state)
	if state["ranked_maintenance"] != "true" {
		t.Fatalf("expected ranked maintenance true, got %#v", state)
	}

	var player Player
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/player_1003/ban", token, map[string]any{
		"reason":         "abuse_report",
		"banned_seconds": 60,
	}, http.StatusOK, &player)
	if player.Status != "banned" {
		t.Fatalf("expected banned player, got %#v", player)
	}
}

func TestRiskAnalyzeDetectsOperationalRisks(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)

	for i := 0; i < 4; i++ {
		playerID := fmt.Sprintf("player_100%d", i%3+1)
		doJSON(t, http.MethodPost, httpServer.URL+"/api/mails", token, map[string]any{
			"player_id": playerID,
			"title":     fmt.Sprintf("Emergency compensation %d", i+1),
			"body":      "risk demo reward",
			"gold":      int64(1200 + i*300),
			"items":     []string{"ticket"},
		}, http.StatusCreated, nil)
	}

	var batch CDKBatch
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/batches", token, map[string]any{
		"name":               "risk demo batch",
		"gold":               200,
		"items":              []string{"gem"},
		"count":              3,
		"max_uses_per_code":  1,
		"expires_in_seconds": 3600,
	}, http.StatusCreated, &batch)
	for _, code := range batch.Codes {
		doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/"+code+"/redeem", "", map[string]any{
			"player_id": "player_1002",
		}, http.StatusOK, nil)
	}

	for i, value := range []string{"true", "false", "true"} {
		doJSON(t, http.MethodPut, httpServer.URL+"/api/ops-configs/ranked_maintenance", token, map[string]any{
			"config_value": value,
			"description":  fmt.Sprintf("risk demo maintenance toggle %d", i+1),
		}, http.StatusOK, nil)
	}
	for i := 0; i < 3; i++ {
		doJSON(t, http.MethodPost, httpServer.URL+"/api/players/player_1003/ban", token, map[string]any{
			"reason":         fmt.Sprintf("risk_demo_%d", i+1),
			"banned_seconds": 60,
		}, http.StatusOK, nil)
	}

	var report RiskAnalysisReport
	doJSON(t, http.MethodPost, httpServer.URL+"/api/risk/analyze", token, map[string]any{
		"use_ai": true,
	}, http.StatusOK, &report)
	if report.ProjectName != RiskAssistantProjectName {
		t.Fatalf("unexpected project name: %s", report.ProjectName)
	}
	if report.RiskLevel != "high" {
		t.Fatalf("expected high risk, got %#v", report)
	}
	if report.Score < 75 {
		t.Fatalf("expected risk score >= 75, got %d", report.Score)
	}
	if report.AIProvider != "mock-ai" || report.Summary == "" {
		t.Fatalf("expected mock-ai summary, got provider=%s summary=%q", report.AIProvider, report.Summary)
	}
	for _, findingType := range []string{
		"high_frequency_reward_mail",
		"high_value_reward_mail",
		"repeated_cdk_redeem",
		"short_time_ban_burst",
		"frequent_ops_config_update",
	} {
		if !hasFinding(report, findingType) {
			t.Fatalf("expected finding %s in %#v", findingType, report.Findings)
		}
	}
}

func TestGameOpsV25MailPreviewExpiryAndAgentAudit(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)

	var preview MailPreview
	doJSON(t, http.MethodPost, httpServer.URL+"/api/mails/preview", token, map[string]any{
		"player_id":          "player_1001",
		"title":              "Agent preview reward",
		"body":               "preview only",
		"gold":               500,
		"items":              []string{"ticket"},
		"expires_in_seconds": 3600,
	}, http.StatusOK, &preview)
	if !preview.Allowed || preview.ExpiresAt == 0 || preview.ExpiresInSeconds != 3600 {
		t.Fatalf("expected allowed preview with expiration, got %#v", preview)
	}

	var blocked MailPreview
	doJSON(t, http.MethodPost, httpServer.URL+"/api/mails/preview", token, map[string]any{
		"player_id": "player_1001",
		"title":     "Too much gold",
		"body":      "blocked",
		"gold":      MaxSingleMailGold + 1,
	}, http.StatusOK, &blocked)
	if blocked.Allowed || len(blocked.Violations) == 0 {
		t.Fatalf("expected blocked preview, got %#v", blocked)
	}

	var mail Mail
	doJSON(t, http.MethodPost, httpServer.URL+"/api/mails", token, map[string]any{
		"player_id":          "player_1001",
		"title":              "Agent approved reward",
		"body":               "approved after preview",
		"gold":               600,
		"items":              []string{"ticket"},
		"expires_in_seconds": 3600,
		"agent_session_id":   "sess_001",
		"agent_mode":         "完全访问权限",
		"confirmation_id":    "confirm_001",
		"confirmed_by":       "user_001",
		"confirmed_at":       int64(1770000000000),
	}, http.StatusCreated, &mail)
	if mail.ExpiresAt == 0 {
		t.Fatalf("expected mail expiration, got %#v", mail)
	}

	var audits []AuditLog
	doJSON(t, http.MethodGet, httpServer.URL+"/api/audit-logs?action=mail.create&target_id="+mail.MailID, token, nil, http.StatusOK, &audits)
	if len(audits) != 1 {
		t.Fatalf("expected one mail audit, got %d", len(audits))
	}
	if audits[0].AgentSessionID != "sess_001" || audits[0].ConfirmationID != "confirm_001" || audits[0].ConfirmedBy != "user_001" {
		t.Fatalf("expected agent audit fields, got %#v", audits[0])
	}
}

func TestMailExpirationBlocksClaim(t *testing.T) {
	store := NewMemoryStore()
	meta := AuditMeta{AdminID: "admin", RequestID: "req_test", ClientIP: "127.0.0.1"}
	store.SeedPlayers(meta)
	mail, err := store.CreateMail("player_1001", "expired", "expired", 100, nil, nowMS()-1000, meta)
	if err != nil {
		t.Fatalf("create expired mail: %v", err)
	}
	if _, err := store.ClaimMail("player_1001", mail.MailID, "req_claim", "127.0.0.1"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestCDKFreezeFlow(t *testing.T) {
	server := NewServer(ConfigFromEnv(), NewMemoryStore())
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	token := login(t, httpServer.URL)
	doJSON(t, http.MethodPost, httpServer.URL+"/api/players/seed", token, map[string]any{}, http.StatusCreated, nil)

	var batch CDKBatch
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/batches", token, map[string]any{
		"name":               "freeze demo",
		"gold":               300,
		"items":              []string{"ticket"},
		"count":              2,
		"max_uses_per_code":  1,
		"expires_in_seconds": 3600,
	}, http.StatusCreated, &batch)

	var frozenCode CDK
	doJSONWithHeaders(t, http.MethodPost, httpServer.URL+"/api/cdk/"+batch.Codes[0]+"/freeze", token, nil, map[string]string{
		"X-Agent-Session-ID":      "sess_freeze",
		"X-Agent-Mode":            "full-access",
		"X-Agent-Confirmation-ID": "confirm_freeze",
	}, http.StatusOK, &frozenCode)
	if frozenCode.Status != "frozen" {
		t.Fatalf("expected frozen cdk, got %#v", frozenCode)
	}
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/"+batch.Codes[0]+"/redeem", "", map[string]any{
		"player_id": "player_1002",
	}, http.StatusBadRequest, nil)

	var frozenBatch CDKBatch
	doJSON(t, http.MethodPost, httpServer.URL+"/api/cdk/batches/"+batch.BatchID+"/freeze", token, nil, http.StatusOK, &frozenBatch)
	if frozenBatch.Status != "frozen" {
		t.Fatalf("expected frozen batch, got %#v", frozenBatch)
	}
	var cdk CDK
	doJSON(t, http.MethodGet, httpServer.URL+"/api/cdk/"+batch.Codes[1], token, nil, http.StatusOK, &cdk)
	if cdk.Status != "frozen" {
		t.Fatalf("expected second cdk frozen by batch, got %#v", cdk)
	}

	var audits []AuditLog
	doJSON(t, http.MethodGet, httpServer.URL+"/api/audit-logs?action=cdk.freeze&target_id="+batch.Codes[0], token, nil, http.StatusOK, &audits)
	if len(audits) != 1 || audits[0].AgentSessionID != "sess_freeze" || audits[0].ConfirmationID != "confirm_freeze" {
		t.Fatalf("expected agent audit on cdk freeze, got %#v", audits)
	}
}

func login(t *testing.T, baseURL string) string {
	t.Helper()

	var resp struct {
		Token string `json:"token"`
	}
	doJSON(t, http.MethodPost, baseURL+"/api/admin/login", "", map[string]any{
		"username": "admin",
		"password": "admin_demo",
	}, http.StatusOK, &resp)
	if resp.Token == "" {
		t.Fatal("expected token")
	}
	return resp.Token
}

func hasFinding(report RiskAnalysisReport, findingType string) bool {
	for _, finding := range report.Findings {
		if finding.Type == findingType {
			return true
		}
	}
	return false
}

func doJSON(t *testing.T, method, url, token string, payload any, expectedStatus int, target any) {
	doJSONWithHeaders(t, method, url, token, payload, nil, expectedStatus, target)
}

func doJSONWithHeaders(t *testing.T, method, url, token string, payload any, headers map[string]string, expectedStatus int, target any) {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d got %d for %s %s", expectedStatus, resp.StatusCode, method, url)
	}
	if target != nil {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			t.Fatalf("decode response: %v", err)
		}
	}
}
