package gameops

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrAlreadyClaimed   = errors.New("mail already claimed")
	ErrAlreadyRedeemed  = errors.New("cdk already redeemed by player")
	ErrExpired          = errors.New("cdk expired")
	ErrInvalidOperation = errors.New("invalid operation")
)

type MemoryStore struct {
	mu          sync.Mutex
	nextID      int64
	players     map[string]*Player
	mails       map[string]*Mail
	mailsByUser map[string][]string
	batches     map[string]*CDKBatch
	cdks        map[string]*CDK
	redemptions map[string]*CDKRedemption
	configs     map[string]*OpsConfig
	events      []*GameEvent
	audits      []*AuditLog
}

func NewMemoryStore() *MemoryStore {
	now := nowMS()
	store := &MemoryStore{
		players:     make(map[string]*Player),
		mails:       make(map[string]*Mail),
		mailsByUser: make(map[string][]string),
		batches:     make(map[string]*CDKBatch),
		cdks:        make(map[string]*CDK),
		redemptions: make(map[string]*CDKRedemption),
		configs:     make(map[string]*OpsConfig),
	}
	for _, cfg := range defaultOpsConfigs(now) {
		copyCfg := cfg
		store.configs[cfg.Key] = &copyCfg
	}
	return store
}

func (s *MemoryStore) SeedPlayers(meta AuditMeta) []Player {
	s.mu.Lock()
	defer s.mu.Unlock()

	seed := []Player{
		{PlayerID: "player_1001", Nickname: "SausageAce", Level: 18, Gold: 1000, RealNameVerified: true, Minor: false, DailyPlaySeconds: 3600},
		{PlayerID: "player_1002", Nickname: "UlaHunter", Level: 12, Gold: 800, RealNameVerified: true, Minor: true, DailyPlaySeconds: 1200},
		{PlayerID: "player_1003", Nickname: "DinoGuest", Level: 5, Gold: 300, RealNameVerified: false, Minor: false, DailyPlaySeconds: 300},
	}
	now := nowMS()
	result := make([]Player, 0, len(seed))
	for _, player := range seed {
		player.Status = "normal"
		player.CreatedAt = now
		player.UpdatedAt = now
		copyPlayer := player
		s.players[player.PlayerID] = &copyPlayer
		result = append(result, copyPlayer)
	}
	s.appendAuditLocked(meta.AdminID, "players.seed", "players", "demo", "", marshalCompact(result), meta)
	return result
}

func (s *MemoryStore) ListPlayers() []Player {
	s.mu.Lock()
	defer s.mu.Unlock()

	players := make([]Player, 0, len(s.players))
	for _, player := range s.players {
		players = append(players, *player)
	}
	sort.Slice(players, func(i, j int) bool { return players[i].PlayerID < players[j].PlayerID })
	return players
}

func (s *MemoryStore) GetPlayer(playerID string) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	copyPlayer := *player
	return &copyPlayer, nil
}

func (s *MemoryStore) BanPlayer(playerID, reason string, until int64, meta AuditMeta) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	before := marshalCompact(player)
	player.Status = "banned"
	player.BanReason = reason
	player.BannedUntil = until
	player.UpdatedAt = nowMS()
	after := marshalCompact(player)
	s.appendAuditLocked(meta.AdminID, "player.ban", "player", playerID, before, after, meta)
	copyPlayer := *player
	return &copyPlayer, nil
}

func (s *MemoryStore) UnbanPlayer(playerID string, meta AuditMeta) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	before := marshalCompact(player)
	player.Status = "normal"
	player.BanReason = ""
	player.BannedUntil = 0
	player.UpdatedAt = nowMS()
	after := marshalCompact(player)
	s.appendAuditLocked(meta.AdminID, "player.unban", "player", playerID, before, after, meta)
	copyPlayer := *player
	return &copyPlayer, nil
}

func (s *MemoryStore) ListConfigs() []OpsConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	configs := make([]OpsConfig, 0, len(s.configs))
	for _, cfg := range s.configs {
		configs = append(configs, *cfg)
	}
	sort.Slice(configs, func(i, j int) bool { return configs[i].Key < configs[j].Key })
	return configs
}

func (s *MemoryStore) UpdateConfig(key, value, description string, meta AuditMeta) OpsConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := ""
	if existing, ok := s.configs[key]; ok {
		before = marshalCompact(existing)
	}
	cfg := &OpsConfig{
		Key:         key,
		Value:       value,
		Description: description,
		UpdatedBy:   meta.AdminID,
		UpdatedAt:   nowMS(),
	}
	s.configs[key] = cfg
	s.appendAuditLocked(meta.AdminID, "ops_config.update", "ops_config", key, before, marshalCompact(cfg), meta)
	return *cfg
}

func (s *MemoryStore) OpsState() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := make(map[string]string, len(s.configs))
	for key, cfg := range s.configs {
		state[key] = cfg.Value
	}
	return state
}

func (s *MemoryStore) CreateMail(playerID, title, body string, gold int64, items []string, expiresAt int64, meta AuditMeta) (*Mail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.players[playerID]; !ok {
		return nil, ErrNotFound
	}
	now := nowMS()
	mail := &Mail{
		MailID:    s.next("mail"),
		PlayerID:  playerID,
		Title:     title,
		Body:      body,
		Gold:      gold,
		Items:     append([]string(nil), items...),
		Status:    "unclaimed",
		ExpiresAt: expiresAt,
		CreatedBy: meta.AdminID,
		CreatedAt: now,
	}
	s.mails[mail.MailID] = mail
	s.mailsByUser[playerID] = append(s.mailsByUser[playerID], mail.MailID)
	s.appendAuditLocked(meta.AdminID, "mail.create", "mail", mail.MailID, "", marshalCompact(mail), meta)
	copyMail := *mail
	return &copyMail, nil
}

func (s *MemoryStore) ListPlayerMails(playerID string) ([]Mail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.players[playerID]; !ok {
		return nil, ErrNotFound
	}
	mails := make([]Mail, 0, len(s.mailsByUser[playerID]))
	for _, mailID := range s.mailsByUser[playerID] {
		mails = append(mails, *s.mails[mailID])
	}
	return mails, nil
}

func (s *MemoryStore) ClaimMail(playerID, mailID, requestID, clientIP string) (*Mail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	mail, ok := s.mails[mailID]
	if !ok || mail.PlayerID != playerID {
		return nil, ErrNotFound
	}
	if mail.Status == "claimed" {
		copyMail := *mail
		return &copyMail, ErrAlreadyClaimed
	}
	if mail.ExpiresAt > 0 && mail.ExpiresAt < nowMS() {
		copyMail := *mail
		return &copyMail, ErrExpired
	}
	before := marshalCompact(mail)
	mail.Status = "claimed"
	mail.ClaimedAt = nowMS()
	player.Gold += mail.Gold
	player.UpdatedAt = mail.ClaimedAt
	s.appendAuditLocked("player:"+playerID, "mail.claim", "mail", mailID, before, marshalCompact(mail), AuditMeta{AdminID: "player:" + playerID, RequestID: requestID, ClientIP: clientIP})
	copyMail := *mail
	return &copyMail, nil
}

func (s *MemoryStore) CreateCDKBatch(name string, gold int64, items []string, count int, maxUses int, expiresAt int64, meta AuditMeta) (*CDKBatch, error) {
	if count <= 0 || count > 100 {
		return nil, fmt.Errorf("%w: count must be 1..100", ErrInvalidOperation)
	}
	if maxUses <= 0 {
		maxUses = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowMS()
	batch := &CDKBatch{
		BatchID:        s.next("batch"),
		Name:           name,
		Gold:           gold,
		Items:          append([]string(nil), items...),
		Status:         "active",
		MaxUsesPerCode: maxUses,
		ExpiresAt:      expiresAt,
		CreatedBy:      meta.AdminID,
		CreatedAt:      now,
	}
	for i := 0; i < count; i++ {
		code := fmt.Sprintf("GO-%s-%04d", batch.BatchID, i+1)
		cdk := &CDK{
			Code:      code,
			BatchID:   batch.BatchID,
			Status:    "active",
			MaxUses:   maxUses,
			ExpiresAt: expiresAt,
			CreatedAt: now,
		}
		s.cdks[code] = cdk
		batch.Codes = append(batch.Codes, code)
	}
	s.batches[batch.BatchID] = batch
	s.appendAuditLocked(meta.AdminID, "cdk_batch.create", "cdk_batch", batch.BatchID, "", marshalCompact(batch), meta)
	copyBatch := *batch
	copyBatch.Codes = append([]string(nil), batch.Codes...)
	return &copyBatch, nil
}

func (s *MemoryStore) ListCDKBatches() []CDKBatch {
	s.mu.Lock()
	defer s.mu.Unlock()

	batches := make([]CDKBatch, 0, len(s.batches))
	for _, batch := range s.batches {
		copyBatch := *batch
		copyBatch.Codes = append([]string(nil), batch.Codes...)
		batches = append(batches, copyBatch)
	}
	sort.Slice(batches, func(i, j int) bool { return batches[i].BatchID < batches[j].BatchID })
	return batches
}

func (s *MemoryStore) GetCDK(code string) (*CDK, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cdk, ok := s.cdks[code]
	if !ok {
		return nil, ErrNotFound
	}
	copyCDK := *cdk
	return &copyCDK, nil
}

func (s *MemoryStore) FreezeCDKBatch(batchID string, meta AuditMeta) (*CDKBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch, ok := s.batches[batchID]
	if !ok {
		return nil, ErrNotFound
	}
	before := marshalCompact(batch)
	batch.Status = "frozen"
	for _, code := range batch.Codes {
		if cdk, ok := s.cdks[code]; ok && cdk.Status == "active" {
			cdk.Status = "frozen"
		}
	}
	s.appendAuditLocked(meta.AdminID, "cdk_batch.freeze", "cdk_batch", batchID, before, marshalCompact(batch), meta)
	copyBatch := *batch
	copyBatch.Codes = append([]string(nil), batch.Codes...)
	return &copyBatch, nil
}

func (s *MemoryStore) FreezeCDK(code string, meta AuditMeta) (*CDK, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cdk, ok := s.cdks[code]
	if !ok {
		return nil, ErrNotFound
	}
	if cdk.Status == "frozen" {
		copyCDK := *cdk
		return &copyCDK, nil
	}
	if cdk.Status != "active" {
		return nil, ErrInvalidOperation
	}
	before := marshalCompact(cdk)
	cdk.Status = "frozen"
	s.appendAuditLocked(meta.AdminID, "cdk.freeze", "cdk", code, before, marshalCompact(cdk), meta)
	copyCDK := *cdk
	return &copyCDK, nil
}

func (s *MemoryStore) RedeemCDK(code, playerID, requestID, clientIP string) (*CDKRedemption, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.players[playerID]
	if !ok {
		return nil, ErrNotFound
	}
	cdk, ok := s.cdks[code]
	if !ok {
		return nil, ErrNotFound
	}
	if cdk.ExpiresAt > 0 && cdk.ExpiresAt < nowMS() {
		return nil, ErrExpired
	}
	key := code + ":" + playerID
	if redemption, ok := s.redemptions[key]; ok {
		copyRedemption := *redemption
		copyRedemption.RewardItems = append([]string(nil), redemption.RewardItems...)
		return &copyRedemption, ErrAlreadyRedeemed
	}
	if cdk.Status != "active" {
		return nil, ErrInvalidOperation
	}
	if cdk.UsedCount >= cdk.MaxUses {
		return nil, ErrConflict
	}
	batch := s.batches[cdk.BatchID]
	before := marshalCompact(cdk)
	cdk.UsedCount++
	if cdk.UsedCount >= cdk.MaxUses {
		cdk.Status = "used"
	}
	player.Gold += batch.Gold
	player.UpdatedAt = nowMS()
	redemption := &CDKRedemption{
		ID:          s.next("redeem"),
		Code:        code,
		PlayerID:    playerID,
		RewardGold:  batch.Gold,
		RewardItems: append([]string(nil), batch.Items...),
		RequestID:   requestID,
		CreatedAt:   nowMS(),
	}
	s.redemptions[key] = redemption
	s.appendAuditLocked("player:"+playerID, "cdk.redeem", "cdk", code, before, marshalCompact(cdk), AuditMeta{AdminID: "player:" + playerID, RequestID: requestID, ClientIP: clientIP})
	copyRedemption := *redemption
	return &copyRedemption, nil
}

func (s *MemoryStore) AddEvent(eventType, playerID string, payload map[string]any) (*GameEvent, error) {
	if eventType == "" {
		return nil, fmt.Errorf("%w: event type is required", ErrInvalidOperation)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	event := &GameEvent{
		EventID:   s.next("event"),
		Type:      eventType,
		PlayerID:  playerID,
		Payload:   payload,
		CreatedAt: nowMS(),
	}
	s.events = append(s.events, event)
	copyEvent := *event
	return &copyEvent, nil
}

func (s *MemoryStore) ListAudits(filter AuditFilter) []AuditLog {
	s.mu.Lock()
	defer s.mu.Unlock()

	audits := make([]AuditLog, 0, len(s.audits))
	for _, audit := range s.audits {
		if !auditMatchesFilter(*audit, filter) {
			continue
		}
		audits = append(audits, *audit)
	}
	return audits
}

func (s *MemoryStore) Stats() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]int{
		"players":     len(s.players),
		"mails":       len(s.mails),
		"cdks":        len(s.cdks),
		"redemptions": len(s.redemptions),
		"events":      len(s.events),
		"audits":      len(s.audits),
	}
}

func (s *MemoryStore) appendAuditLocked(adminID, action, targetType, targetID, before, after string, meta AuditMeta) {
	if adminID == "" {
		adminID = meta.AdminID
	}
	s.audits = append(s.audits, &AuditLog{
		ID:             s.next("audit"),
		AdminID:        adminID,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		BeforeJSON:     before,
		AfterJSON:      after,
		RequestID:      meta.RequestID,
		ClientIP:       meta.ClientIP,
		AgentSessionID: meta.Agent.AgentSessionID,
		AgentMode:      meta.Agent.AgentMode,
		ConfirmationID: meta.Agent.ConfirmationID,
		ConfirmedBy:    meta.Agent.ConfirmedBy,
		ConfirmedAt:    meta.Agent.ConfirmedAt,
		CreatedAt:      nowMS(),
	})
}

func (s *MemoryStore) next(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s_%06d", prefix, s.nextID)
}

func nowMS() int64 {
	return time.Now().UnixMilli()
}

func marshalCompact(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func auditMatchesFilter(audit AuditLog, filter AuditFilter) bool {
	if filter.AdminID != "" && audit.AdminID != filter.AdminID {
		return false
	}
	if filter.Action != "" && audit.Action != filter.Action {
		return false
	}
	if filter.TargetType != "" && audit.TargetType != filter.TargetType {
		return false
	}
	if filter.TargetID != "" && audit.TargetID != filter.TargetID {
		return false
	}
	if filter.FromMS > 0 && audit.CreatedAt < filter.FromMS {
		return false
	}
	if filter.ToMS > 0 && audit.CreatedAt > filter.ToMS {
		return false
	}
	return true
}

func defaultOpsConfigs(now int64) []OpsConfig {
	return []OpsConfig{
		{Key: "announcement", Value: "SS25 season is live", Description: "demo ops config", UpdatedBy: "system", UpdatedAt: now},
		{Key: "event_enabled", Value: "true", Description: "demo ops config", UpdatedBy: "system", UpdatedAt: now},
		{Key: "login_maintenance", Value: "false", Description: "demo ops config", UpdatedBy: "system", UpdatedAt: now},
		{Key: "ranked_maintenance", Value: "false", Description: "demo ops config", UpdatedBy: "system", UpdatedAt: now},
	}
}
