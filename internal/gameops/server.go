package gameops

import (
	"encoding/json"
	"errors"
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
	mux.HandleFunc("POST /api/mails", s.withAdmin(s.handleCreateMail))
	mux.HandleFunc("GET /api/players/{player_id}/mails", s.withAdmin(s.handleListMails))
	mux.HandleFunc("POST /api/cdk/batches", s.withAdmin(s.handleCreateCDKBatch))
	mux.HandleFunc("GET /api/cdk/batches", s.withAdmin(s.handleListCDKBatches))
	mux.HandleFunc("GET /api/cdk/{code}", s.withAdmin(s.handleGetCDK))
	mux.HandleFunc("GET /api/audit-logs", s.withAdmin(s.handleAuditLogs))
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
	players := s.store.SeedPlayers(principal.UserID, requestID(r), clientIP(r))
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
	}
	if !s.decode(w, r, &req) {
		return
	}
	if req.Reason == "" {
		req.Reason = "gm_action"
	}
	until := int64(0)
	if req.BannedSeconds > 0 {
		until = nowMS() + req.BannedSeconds*1000
	}
	player, err := s.store.BanPlayer(r.PathValue("player_id"), req.Reason, until, principal.UserID, requestID(r), clientIP(r))
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
	player, err := s.store.UnbanPlayer(r.PathValue("player_id"), principal.UserID, requestID(r), clientIP(r))
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
	}
	if !s.decode(w, r, &req) {
		return
	}
	cfg := s.store.UpdateConfig(r.PathValue("config_key"), req.Value, req.Description, principal.UserID, requestID(r), clientIP(r))
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

func (s *Server) handleCreateMail(w http.ResponseWriter, r *http.Request, principal Principal) {
	if !s.requireRole(w, principal, "admin", "operator") {
		return
	}
	var req struct {
		PlayerID string   `json:"player_id"`
		Title    string   `json:"title"`
		Body     string   `json:"body"`
		Gold     int64    `json:"gold"`
		Items    []string `json:"items"`
	}
	if !s.decode(w, r, &req) {
		return
	}
	mail, err := s.store.CreateMail(req.PlayerID, req.Title, req.Body, req.Gold, req.Items, principal.UserID, requestID(r), clientIP(r))
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
	}
	if !s.decode(w, r, &req) {
		return
	}
	expiresAt := int64(0)
	if req.ExpiresInSec > 0 {
		expiresAt = nowMS() + req.ExpiresInSec*1000
	}
	batch, err := s.store.CreateCDKBatch(req.Name, req.Gold, req.Items, req.Count, req.MaxUsesPerCode, expiresAt, principal.UserID, requestID(r), clientIP(r))
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
