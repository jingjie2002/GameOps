package gameops

import (
	"bytes"
	"encoding/json"
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

func doJSON(t *testing.T, method, url, token string, payload any, expectedStatus int, target any) {
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
