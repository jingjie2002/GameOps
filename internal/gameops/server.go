package gameops

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	cfg      Config
	store    Store
	metrics  *Metrics
	corerank CoreRankClient
}

func NewServer(cfg Config, store Store) *Server {
	return &Server{
		cfg:      cfg,
		store:    store,
		metrics:  &Metrics{},
		corerank: NewCoreRankClient(cfg.CoreRankHTTP),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.Handle("GET /metrics", s.metrics)
	mux.HandleFunc("GET /api/agent/capabilities", s.handleAgentCapabilities)
	mux.HandleFunc("GET /api/agent/events", s.handleAgentEvents)
	mux.HandleFunc("GET /api/agent/logs", s.handleAgentLogs)
	mux.HandleFunc("POST /api/admin/login", s.handleLogin)
	mux.HandleFunc("GET /api/public/ops-state", s.handleOpsState)
	mux.HandleFunc("GET /api/public/players/{player_id}/state", s.handlePublicPlayerState)
	mux.HandleFunc("POST /api/players/{player_id}/mails/{mail_id}/claim", s.handleClaimMail)
	mux.HandleFunc("POST /api/cdk/{code}/redeem", s.handleRedeemCDK)
	mux.HandleFunc("POST /api/events", s.handleEvent)

	mux.HandleFunc("GET /api/admin/me", s.withAdmin(s.handleMe))
	mux.HandleFunc("POST /api/players/seed", s.withAdmin(s.handleSeedPlayers))
	mux.HandleFunc("GET /api/players", s.withAdmin(s.handleListPlayers))
	mux.HandleFunc("GET /api/players/{player_id}", s.withAdmin(s.handleGetPlayer))
	mux.HandleFunc("POST /api/players/{player_id}/ban", s.withAdmin(s.handleBanPlayer))
	mux.HandleFunc("POST /api/players/{player_id}/unban", s.withAdmin(s.handleUnbanPlayer))
	mux.HandleFunc("GET /api/ops-configs", s.withAdmin(s.handleListConfigs))
	mux.HandleFunc("PUT /api/ops-configs/{config_key}", s.withAdmin(s.handleUpdateConfig))
	mux.HandleFunc("POST /api/mails/preview", s.withAdmin(s.handlePreviewMail))
	mux.HandleFunc("POST /api/mails", s.withAdmin(s.handleCreateMail))
	mux.HandleFunc("GET /api/players/{player_id}/mails", s.withAdmin(s.handleListMails))
	mux.HandleFunc("POST /api/cdk/batches", s.withAdmin(s.handleCreateCDKBatch))
	mux.HandleFunc("GET /api/cdk/batches", s.withAdmin(s.handleListCDKBatches))
	mux.HandleFunc("POST /api/cdk/batches/{batch_id}/freeze", s.withAdmin(s.handleFreezeCDKBatch))
	mux.HandleFunc("GET /api/cdk/{code}", s.withAdmin(s.handleGetCDK))
	mux.HandleFunc("POST /api/cdk/{code}/freeze", s.withAdmin(s.handleFreezeCDK))
	mux.HandleFunc("GET /api/audit-logs", s.withAdmin(s.handleAuditLogs))
	mux.HandleFunc("POST /api/risk/analyze", s.withAdmin(s.handleRiskAnalyze))
	mux.HandleFunc("GET /api/integrations/corerank/health", s.withAdmin(s.handleCoreRankHealth))
	mux.HandleFunc("GET /api/integrations/corerank/leaderboard", s.withAdmin(s.handleCoreRankLeaderboard))
	mux.HandleFunc("GET /api/integrations/corerank/players/{player_id}/rank", s.withAdmin(s.handleCoreRankPlayerRank))

	return s.requestMiddleware(mux)
}

func (s *Server) requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.metrics.requests.Add(1)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = "req_" + strconvFormatInt(time.Now().UnixNano())
		}
		ctx := withRequestID(r.Context(), requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) withAdmin(next func(http.ResponseWriter, *http.Request, Principal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := s.principalFromRequest(r)
		if err != nil {
			s.writeError(w, http.StatusUnauthorized, err)
			return
		}
		next(w, r, *principal)
	}
}

func (s *Server) principalFromRequest(r *http.Request) (*Principal, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, ErrUnauthorized
	}
	return s.verifyToken(strings.TrimPrefix(auth, "Bearer "))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"stats":  s.store.Stats(),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	role := ""
	switch {
	case req.Username == s.cfg.AdminUser && req.Password == s.cfg.AdminPassword:
		role = "admin"
	case req.Username == s.cfg.OperatorUser && req.Password == s.cfg.OperatorPassword:
		role = "operator"
	case req.Username == s.cfg.AuditorUser && req.Password == s.cfg.AuditorPassword:
		role = "auditor"
	default:
		s.writeError(w, http.StatusUnauthorized, ErrUnauthorized)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"token": s.issueToken(req.Username, role),
		"role":  role,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, _ *http.Request, principal Principal) {
	s.writeJSON(w, http.StatusOK, principal)
}

func (s *Server) handleSeedPlayers(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	players := s.store.SeedPlayers(auditMetaFromRequest(r, principal, AgentAuditFields{}))
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusCreated, players)
}

func (s *Server) handleListPlayers(w http.ResponseWriter, _ *http.Request, _ Principal) {
	s.writeJSON(w, http.StatusOK, s.store.ListPlayers())
}

func (s *Server) handleGetPlayer(w http.ResponseWriter, r *http.Request, _ Principal) {
	player, err := s.store.GetPlayer(r.PathValue("player_id"))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, player)
}

func (s *Server) handleBanPlayer(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	var req struct {
		Reason        string `json:"reason"`
		BannedSeconds int64  `json:"banned_seconds"`
		AgentAuditFields
	}
	if !s.decode(w, r, &req) {
		return
	}
	if err := validateBanSeconds(req.BannedSeconds); err != nil {
		s.writeDomainError(w, err)
		return
	}
	if req.Reason == "" {
		req.Reason = "gm_action"
	}
	until := nowMS() + req.BannedSeconds*1000
	player, err := s.store.BanPlayer(r.PathValue("player_id"), req.Reason, until, auditMetaFromRequest(r, principal, req.AgentAuditFields))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, player)
}

func (s *Server) handleUnbanPlayer(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	player, err := s.store.UnbanPlayer(r.PathValue("player_id"), auditMetaFromRequest(r, principal, AgentAuditFields{}))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, player)
}

func (s *Server) handleListConfigs(w http.ResponseWriter, _ *http.Request, _ Principal) {
	s.writeJSON(w, http.StatusOK, s.store.ListConfigs())
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	var req struct {
		Value       string `json:"config_value"`
		Description string `json:"description"`
		AgentAuditFields
	}
	if !s.decode(w, r, &req) {
		return
	}
	cfg := s.store.UpdateConfig(r.PathValue("config_key"), req.Value, req.Description, auditMetaFromRequest(r, principal, req.AgentAuditFields))
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handleOpsState(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, s.store.OpsState())
}

func (s *Server) handlePublicPlayerState(w http.ResponseWriter, r *http.Request) {
	player, err := s.store.GetPlayer(r.PathValue("player_id"))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"player_id":    player.PlayerID,
		"status":       player.Status,
		"ban_reason":   player.BanReason,
		"banned_until": player.BannedUntil,
	})
}

func (s *Server) handlePreviewMail(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "operator") {
		return
	}
	var req struct {
		PlayerID         string   `json:"player_id"`
		PlayerIDs        []string `json:"player_ids"`
		Title            string   `json:"title"`
		Body             string   `json:"body"`
		Gold             int64    `json:"gold"`
		Items            []string `json:"items"`
		ExpiresInSeconds int64    `json:"expires_in_seconds"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	draft := MailDraft{
		PlayerID:         req.PlayerID,
		PlayerIDs:        req.PlayerIDs,
		Title:            req.Title,
		Body:             req.Body,
		Gold:             req.Gold,
		Items:            req.Items,
		ExpiresInSeconds: req.ExpiresInSeconds,
	}
	preview := PreviewMailDraft(draft, nowMS())
	for _, playerID := range mailTargets(draft) {
		if _, err := s.store.GetPlayer(playerID); err != nil {
			preview.Allowed = false
			preview.RiskLevel = "blocked"
			preview.Violations = append(preview.Violations, "player not found: "+playerID)
		}
	}
	s.writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleCreateMail(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "operator") {
		return
	}
	var req struct {
		PlayerID         string   `json:"player_id"`
		PlayerIDs        []string `json:"player_ids"`
		Title            string   `json:"title"`
		Body             string   `json:"body"`
		Gold             int64    `json:"gold"`
		Items            []string `json:"items"`
		ExpiresInSeconds int64    `json:"expires_in_seconds"`
		AgentAuditFields
	}
	if !s.decode(w, r, &req) {
		return
	}
	draft := MailDraft{
		PlayerID:         req.PlayerID,
		PlayerIDs:        req.PlayerIDs,
		Title:            req.Title,
		Body:             req.Body,
		Gold:             req.Gold,
		Items:            req.Items,
		ExpiresInSeconds: req.ExpiresInSeconds,
	}
	targets := mailTargets(draft)
	if len(targets) != 1 {
		s.writeDomainError(w, fmt.Errorf("%w: POST /api/mails requires exactly one player_id", ErrInvalidOperation))
		return
	}
	preview, err := validateMailDraft(draft, nowMS())
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	mail, err := s.store.CreateMail(targets[0], req.Title, req.Body, req.Gold, req.Items, preview.ExpiresAt, auditMetaFromRequest(r, principal, req.AgentAuditFields))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.mails.Add(1)
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusCreated, mail)
}

func (s *Server) handleListMails(w http.ResponseWriter, r *http.Request, _ Principal) {
	mails, err := s.store.ListPlayerMails(r.PathValue("player_id"))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, mails)
}

func (s *Server) handleClaimMail(w http.ResponseWriter, r *http.Request) {
	mail, err := s.store.ClaimMail(r.PathValue("player_id"), r.PathValue("mail_id"), requestID(r), clientIP(r))
	if err != nil && !errors.Is(err, ErrAlreadyClaimed) {
		s.writeDomainError(w, err)
		return
	}
	if errors.Is(err, ErrAlreadyClaimed) {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, mail)
}

func (s *Server) handleCreateCDKBatch(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "operator") {
		return
	}
	var req struct {
		Name           string   `json:"name"`
		Gold           int64    `json:"gold"`
		Items          []string `json:"items"`
		Count          int      `json:"count"`
		MaxUsesPerCode int      `json:"max_uses_per_code"`
		ExpiresInSec   int64    `json:"expires_in_seconds"`
		AgentAuditFields
	}
	if !s.decode(w, r, &req) {
		return
	}
	expiresAt := int64(0)
	if req.ExpiresInSec > 0 {
		if req.ExpiresInSec < MinMailExpiresSeconds || req.ExpiresInSec > MaxMailExpiresSeconds {
			s.writeDomainError(w, fmt.Errorf("%w: expires_in_seconds out of allowed range", ErrInvalidOperation))
			return
		}
		expiresAt = nowMS() + req.ExpiresInSec*1000
	}
	if req.Gold > MaxSingleMailGold || len(req.Items) > MaxMailItemCount {
		s.writeDomainError(w, fmt.Errorf("%w: reward exceeds configured safety limits", ErrInvalidOperation))
		return
	}
	batch, err := s.store.CreateCDKBatch(req.Name, req.Gold, req.Items, req.Count, req.MaxUsesPerCode, expiresAt, auditMetaFromRequest(r, principal, req.AgentAuditFields))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusCreated, batch)
}

func (s *Server) handleListCDKBatches(w http.ResponseWriter, _ *http.Request, _ Principal) {
	s.writeJSON(w, http.StatusOK, s.store.ListCDKBatches())
}

func (s *Server) handleGetCDK(w http.ResponseWriter, r *http.Request, _ Principal) {
	cdk, err := s.store.GetCDK(r.PathValue("code"))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, cdk)
}

func (s *Server) handleFreezeCDKBatch(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	batch, err := s.store.FreezeCDKBatch(r.PathValue("batch_id"), auditMetaFromRequest(r, principal, AgentAuditFields{}))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, batch)
}

func (s *Server) handleFreezeCDK(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin") {
		return
	}
	cdk, err := s.store.FreezeCDK(r.PathValue("code"), auditMetaFromRequest(r, principal, AgentAuditFields{}))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, cdk)
}

func (s *Server) handleRedeemCDK(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	redemption, err := s.store.RedeemCDK(r.PathValue("code"), req.PlayerID, requestID(r), clientIP(r))
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.cdkRedeems.Add(1)
	s.metrics.auditWrites.Add(1)
	s.writeJSON(w, http.StatusOK, redemption)
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string         `json:"type"`
		PlayerID string         `json:"player_id"`
		Payload  map[string]any `json:"payload"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	event, err := s.store.AddEvent(req.Type, req.PlayerID, req.Payload)
	if err != nil {
		s.writeDomainError(w, err)
		return
	}
	s.metrics.gameEvents.Add(1)
	s.writeJSON(w, http.StatusCreated, event)
}

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "auditor") {
		return
	}
	s.writeJSON(w, http.StatusOK, s.store.ListAudits(parseAuditFilter(r)))
}

func (s *Server) handleRiskAnalyze(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "auditor") {
		return
	}
	var req RiskAnalyzeRequest
	if !s.decode(w, r, &req) {
		return
	}
	if req.FromMS > 0 && req.ToMS > 0 && req.FromMS > req.ToMS {
		s.writeError(w, http.StatusBadRequest, errors.New("from_ms must be less than or equal to to_ms"))
		return
	}
	useAI := true
	if req.UseAI != nil {
		useAI = *req.UseAI
	}
	provider := req.AIProvider
	if provider == "" {
		provider = s.cfg.RiskAIProvider
	}
	if provider == "" {
		provider = "mock-ai"
	}
	if useAI && provider != "mock-ai" {
		s.writeError(w, http.StatusBadRequest, errors.New("unsupported ai_provider: only mock-ai is available in demo mode"))
		return
	}
	if !useAI {
		provider = "rules-only"
	}

	audits := s.store.ListAudits(AuditFilter{FromMS: req.FromMS, ToMS: req.ToMS})
	report := AnalyzeOperationalRisk(audits, RiskAnalyzeOptions{
		FromMS:     req.FromMS,
		ToMS:       req.ToMS,
		UseAI:      useAI,
		AIProvider: provider,
		AnalyzedAt: nowMS(),
	})
	s.metrics.riskAnalyses.Add(1)
	s.writeJSON(w, http.StatusOK, report)
}

func (s *Server) handleCoreRankHealth(w http.ResponseWriter, r *http.Request, _ Principal) {
	result, err := s.corerank.Health(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"status": "unavailable", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCoreRankLeaderboard(w http.ResponseWriter, r *http.Request, _ Principal) {
	result, err := s.corerank.Leaderboard(r.Context(), r.URL.Query().Get("leaderboard_type"), r.URL.Query().Get("limit"))
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"status": "unavailable", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCoreRankPlayerRank(w http.ResponseWriter, r *http.Request, _ Principal) {
	result, err := s.corerank.PlayerRank(r.Context(), r.PathValue("player_id"), r.URL.Query().Get("leaderboard_type"))
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"status": "unavailable", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		s.writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func (s *Server) writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		s.writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrConflict), errors.Is(err, ErrAlreadyClaimed), errors.Is(err, ErrAlreadyRedeemed):
		s.writeError(w, http.StatusConflict, err)
	case errors.Is(err, ErrExpired), errors.Is(err, ErrInvalidOperation):
		s.writeError(w, http.StatusBadRequest, err)
	default:
		s.writeError(w, http.StatusInternalServerError, err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, err error) {
	s.metrics.errors.Add(1)
	s.writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func requestID(r *http.Request) string {
	return requestIDFromContext(r.Context())
}

func auditMetaFromRequest(r *http.Request, principal Principal, body AgentAuditFields) AuditMeta {
	agent := body
	if agent.AgentSessionID == "" {
		agent.AgentSessionID = r.Header.Get("X-Agent-Session-ID")
	}
	if agent.AgentMode == "" {
		agent.AgentMode = r.Header.Get("X-Agent-Mode")
	}
	if agent.ConfirmationID == "" {
		agent.ConfirmationID = r.Header.Get("X-Agent-Confirmation-ID")
	}
	if agent.ConfirmedBy == "" {
		agent.ConfirmedBy = r.Header.Get("X-Agent-Confirmed-By")
	}
	if agent.ConfirmedAt == 0 {
		if value := r.Header.Get("X-Agent-Confirmed-At"); value != "" {
			agent.ConfirmedAt, _ = strconvParseInt(value)
		}
	}
	if agent.ConfirmationID != "" && agent.ConfirmedBy == "" {
		agent.ConfirmedBy = principal.UserID
	}
	return AuditMeta{
		AdminID:   principal.UserID,
		RequestID: requestID(r),
		ClientIP:  clientIP(r),
		Agent:     agent,
	}
}

func (s *Server) requireRole(w http.ResponseWriter, principal Principal, roles ...string) bool {
	for _, role := range roles {
		if principal.Role == role {
			return true
		}
	}
	s.writeError(w, http.StatusForbidden, ErrUnauthorized)
	return false
}

func parseAuditFilter(r *http.Request) AuditFilter {
	query := r.URL.Query()
	filter := AuditFilter{
		AdminID:    query.Get("admin_id"),
		Action:     query.Get("action"),
		TargetType: query.Get("target_type"),
		TargetID:   query.Get("target_id"),
	}
	if value := query.Get("from_ms"); value != "" {
		filter.FromMS, _ = strconvParseInt(value)
	}
	if value := query.Get("to_ms"); value != "" {
		filter.ToMS, _ = strconvParseInt(value)
	}
	return filter
}
