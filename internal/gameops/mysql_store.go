package gameops

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &MySQLStore{db: db}
	if err := store.ensureDefaultConfigs(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *MySQLStore) ensureDefaultConfigs(ctx context.Context) error {
	now := nowMS()
	for _, cfg := range defaultOpsConfigs(now) {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO ops_configs (config_key,config_value,description,updated_by,updated_at_ms)
VALUES (?,?,?,?,?)
ON DUPLICATE KEY UPDATE config_key=config_key`,
			cfg.Key, cfg.Value, cfg.Description, cfg.UpdatedBy, cfg.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) SeedPlayers(meta AuditMeta) []Player {
	seed := []Player{
		{PlayerID: "player_1001", Nickname: "SausageAce", Level: 18, Gold: 1000, Status: "normal", RealNameVerified: true, Minor: false, DailyPlaySeconds: 3600},
		{PlayerID: "player_1002", Nickname: "UlaHunter", Level: 12, Gold: 800, Status: "normal", RealNameVerified: true, Minor: true, DailyPlaySeconds: 1200},
		{PlayerID: "player_1003", Nickname: "DinoGuest", Level: 5, Gold: 300, Status: "normal", RealNameVerified: false, Minor: false, DailyPlaySeconds: 300},
	}
	now := nowMS()
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil
	}
	for i := range seed {
		seed[i].CreatedAt = now
		seed[i].UpdatedAt = now
		_, _ = tx.Exec(`
INSERT INTO players (player_id,nickname,level,gold,status,ban_reason,banned_until_ms,real_name_verified,minor,daily_play_seconds,created_at_ms,updated_at_ms)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
ON DUPLICATE KEY UPDATE nickname=VALUES(nickname),level=VALUES(level),gold=VALUES(gold),status=VALUES(status),ban_reason='',banned_until_ms=0,real_name_verified=VALUES(real_name_verified),minor=VALUES(minor),daily_play_seconds=VALUES(daily_play_seconds),updated_at_ms=VALUES(updated_at_ms)`,
			seed[i].PlayerID, seed[i].Nickname, seed[i].Level, seed[i].Gold, seed[i].Status, seed[i].BanReason, seed[i].BannedUntil,
			seed[i].RealNameVerified, seed[i].Minor, seed[i].DailyPlaySeconds, seed[i].CreatedAt, seed[i].UpdatedAt)
	}
	s.appendAuditTx(tx, meta.AdminID, "players.seed", "players", "demo", "", marshalCompact(seed), meta)
	_ = tx.Commit()
	return seed
}

func (s *MySQLStore) ListPlayers() []Player {
	rows, err := s.db.Query(`
SELECT player_id,nickname,level,gold,status,ban_reason,banned_until_ms,real_name_verified,minor,daily_play_seconds,created_at_ms,updated_at_ms
FROM players ORDER BY player_id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var players []Player
	for rows.Next() {
		player, err := scanPlayer(rows)
		if err == nil {
			players = append(players, player)
		}
	}
	return players
}

func (s *MySQLStore) GetPlayer(playerID string) (*Player, error) {
	row := s.db.QueryRow(`
SELECT player_id,nickname,level,gold,status,ban_reason,banned_until_ms,real_name_verified,minor,daily_play_seconds,created_at_ms,updated_at_ms
FROM players WHERE player_id=?`, playerID)
	player, err := scanPlayer(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (s *MySQLStore) BanPlayer(playerID, reason string, until int64, meta AuditMeta) (*Player, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	before, err := s.getPlayerTx(tx, playerID, true)
	if err != nil {
		return nil, err
	}
	now := nowMS()
	if _, err := tx.Exec(`UPDATE players SET status='banned',ban_reason=?,banned_until_ms=?,updated_at_ms=? WHERE player_id=?`, reason, until, now, playerID); err != nil {
		return nil, err
	}
	after, err := s.getPlayerTx(tx, playerID, false)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(`INSERT INTO player_status_logs (id,player_id,status,reason,banned_until_ms,created_by,created_at_ms) VALUES (?,?,?,?,?,?,?)`,
		newStoreID("status"), playerID, "banned", reason, until, meta.AdminID, now)
	s.appendAuditTx(tx, meta.AdminID, "player.ban", "player", playerID, marshalCompact(before), marshalCompact(after), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *MySQLStore) UnbanPlayer(playerID string, meta AuditMeta) (*Player, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	before, err := s.getPlayerTx(tx, playerID, true)
	if err != nil {
		return nil, err
	}
	now := nowMS()
	if _, err := tx.Exec(`UPDATE players SET status='normal',ban_reason='',banned_until_ms=0,updated_at_ms=? WHERE player_id=?`, now, playerID); err != nil {
		return nil, err
	}
	after, err := s.getPlayerTx(tx, playerID, false)
	if err != nil {
		return nil, err
	}
	_, _ = tx.Exec(`INSERT INTO player_status_logs (id,player_id,status,reason,banned_until_ms,created_by,created_at_ms) VALUES (?,?,?,?,?,?,?)`,
		newStoreID("status"), playerID, "normal", "", 0, meta.AdminID, now)
	s.appendAuditTx(tx, meta.AdminID, "player.unban", "player", playerID, marshalCompact(before), marshalCompact(after), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (s *MySQLStore) ListConfigs() []OpsConfig {
	rows, err := s.db.Query(`SELECT config_key,config_value,description,updated_by,updated_at_ms FROM ops_configs ORDER BY config_key`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var configs []OpsConfig
	for rows.Next() {
		var cfg OpsConfig
		if err := rows.Scan(&cfg.Key, &cfg.Value, &cfg.Description, &cfg.UpdatedBy, &cfg.UpdatedAt); err == nil {
			configs = append(configs, cfg)
		}
	}
	return configs
}

func (s *MySQLStore) UpdateConfig(key, value, description string, meta AuditMeta) OpsConfig {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return OpsConfig{}
	}
	defer tx.Rollback()
	before := ""
	if existing, ok := s.getConfigTx(tx, key); ok {
		before = marshalCompact(existing)
	}
	cfg := OpsConfig{Key: key, Value: value, Description: description, UpdatedBy: meta.AdminID, UpdatedAt: nowMS()}
	_, _ = tx.Exec(`
INSERT INTO ops_configs (config_key,config_value,description,updated_by,updated_at_ms)
VALUES (?,?,?,?,?)
ON DUPLICATE KEY UPDATE config_value=VALUES(config_value),description=VALUES(description),updated_by=VALUES(updated_by),updated_at_ms=VALUES(updated_at_ms)`,
		cfg.Key, cfg.Value, cfg.Description, cfg.UpdatedBy, cfg.UpdatedAt)
	s.appendAuditTx(tx, meta.AdminID, "ops_config.update", "ops_config", key, before, marshalCompact(cfg), meta)
	_ = tx.Commit()
	return cfg
}

func (s *MySQLStore) OpsState() map[string]string {
	state := map[string]string{}
	for _, cfg := range s.ListConfigs() {
		state[cfg.Key] = cfg.Value
	}
	return state
}

func (s *MySQLStore) CreateMail(playerID, title, body string, gold int64, items []string, expiresAt int64, meta AuditMeta) (*Mail, error) {
	if _, err := s.GetPlayer(playerID); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	mail := &Mail{MailID: newStoreID("mail"), PlayerID: playerID, Title: title, Body: body, Gold: gold, Items: append([]string(nil), items...), Status: "unclaimed", ExpiresAt: expiresAt, CreatedBy: meta.AdminID, CreatedAt: nowMS()}
	if _, err := tx.Exec(`
INSERT INTO mails (mail_id,player_id,title,body,gold,items_json,status,claimed_at_ms,expires_at_ms,created_by,created_at_ms)
VALUES (?,?,?,?,?,?,?,?,?,?,?)`, mail.MailID, mail.PlayerID, mail.Title, mail.Body, mail.Gold, encodeJSON(mail.Items), mail.Status, mail.ClaimedAt, mail.ExpiresAt, mail.CreatedBy, mail.CreatedAt); err != nil {
		return nil, err
	}
	s.appendAuditTx(tx, meta.AdminID, "mail.create", "mail", mail.MailID, "", marshalCompact(mail), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return mail, nil
}

func (s *MySQLStore) ListPlayerMails(playerID string) ([]Mail, error) {
	if _, err := s.GetPlayer(playerID); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT mail_id,player_id,title,body,gold,items_json,status,claimed_at_ms,expires_at_ms,created_by,created_at_ms FROM mails WHERE player_id=? ORDER BY created_at_ms`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mails []Mail
	for rows.Next() {
		mail, err := scanMail(rows)
		if err == nil {
			mails = append(mails, mail)
		}
	}
	return mails, nil
}

func (s *MySQLStore) ClaimMail(playerID, mailID, requestID, clientIP string) (*Mail, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	player, err := s.getPlayerTx(tx, playerID, true)
	if err != nil {
		return nil, err
	}
	mail, err := s.getMailTx(tx, playerID, mailID, true)
	if err != nil {
		return nil, err
	}
	if mail.Status == "claimed" {
		return mail, ErrAlreadyClaimed
	}
	if mail.ExpiresAt > 0 && mail.ExpiresAt < nowMS() {
		return mail, ErrExpired
	}
	before := marshalCompact(mail)
	now := nowMS()
	if _, err := tx.Exec(`UPDATE mails SET status='claimed',claimed_at_ms=? WHERE mail_id=?`, now, mailID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE players SET gold=?,updated_at_ms=? WHERE player_id=?`, player.Gold+mail.Gold, now, playerID); err != nil {
		return nil, err
	}
	updated, err := s.getMailTx(tx, playerID, mailID, false)
	if err != nil {
		return nil, err
	}
	s.appendAuditTx(tx, "player:"+playerID, "mail.claim", "mail", mailID, before, marshalCompact(updated), AuditMeta{AdminID: "player:" + playerID, RequestID: requestID, ClientIP: clientIP})
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *MySQLStore) CreateCDKBatch(name string, gold int64, items []string, count int, maxUses int, expiresAt int64, meta AuditMeta) (*CDKBatch, error) {
	if count <= 0 || count > 100 {
		return nil, fmt.Errorf("%w: count must be 1..100", ErrInvalidOperation)
	}
	if maxUses <= 0 {
		maxUses = 1
	}
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	now := nowMS()
	batch := &CDKBatch{BatchID: newStoreID("batch"), Name: name, Gold: gold, Items: append([]string(nil), items...), Status: "active", MaxUsesPerCode: maxUses, ExpiresAt: expiresAt, CreatedBy: meta.AdminID, CreatedAt: now}
	if _, err := tx.Exec(`INSERT INTO cdk_batches (batch_id,name,gold,items_json,status,max_uses_per_code,expires_at_ms,created_by,created_at_ms) VALUES (?,?,?,?,?,?,?,?,?)`,
		batch.BatchID, batch.Name, batch.Gold, encodeJSON(batch.Items), batch.Status, batch.MaxUsesPerCode, batch.ExpiresAt, batch.CreatedBy, batch.CreatedAt); err != nil {
		return nil, err
	}
	for i := 0; i < count; i++ {
		code := fmt.Sprintf("GO-%s-%04d", batch.BatchID, i+1)
		batch.Codes = append(batch.Codes, code)
		if _, err := tx.Exec(`INSERT INTO cdks (code,batch_id,status,used_count,max_uses,expires_at_ms,created_at_ms) VALUES (?,?,?,?,?,?,?)`,
			code, batch.BatchID, "active", 0, maxUses, expiresAt, now); err != nil {
			return nil, err
		}
	}
	s.appendAuditTx(tx, meta.AdminID, "cdk_batch.create", "cdk_batch", batch.BatchID, "", marshalCompact(batch), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *MySQLStore) ListCDKBatches() []CDKBatch {
	rows, err := s.db.Query(`SELECT batch_id,name,gold,items_json,status,max_uses_per_code,expires_at_ms,created_by,created_at_ms FROM cdk_batches ORDER BY batch_id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var batches []CDKBatch
	for rows.Next() {
		var batch CDKBatch
		var itemsJSON string
		if err := rows.Scan(&batch.BatchID, &batch.Name, &batch.Gold, &itemsJSON, &batch.Status, &batch.MaxUsesPerCode, &batch.ExpiresAt, &batch.CreatedBy, &batch.CreatedAt); err == nil {
			batch.Items = decodeStringSlice(itemsJSON)
			batches = append(batches, batch)
		}
	}
	return batches
}

func (s *MySQLStore) GetCDK(code string) (*CDK, error) {
	row := s.db.QueryRow(`SELECT code,batch_id,status,used_count,max_uses,expires_at_ms,created_at_ms FROM cdks WHERE code=?`, code)
	cdk, err := scanCDK(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &cdk, nil
}

func (s *MySQLStore) FreezeCDKBatch(batchID string, meta AuditMeta) (*CDKBatch, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	batch, err := s.getBatchTx(tx, batchID)
	if err != nil {
		return nil, err
	}
	before := marshalCompact(batch)
	if _, err := tx.Exec(`UPDATE cdk_batches SET status='frozen' WHERE batch_id=?`, batchID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE cdks SET status='frozen' WHERE batch_id=? AND status='active'`, batchID); err != nil {
		return nil, err
	}
	batch.Status = "frozen"
	s.appendAuditTx(tx, meta.AdminID, "cdk_batch.freeze", "cdk_batch", batchID, before, marshalCompact(batch), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return batch, nil
}

func (s *MySQLStore) FreezeCDK(code string, meta AuditMeta) (*CDK, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	cdk, err := s.getCDKTx(tx, code, true)
	if err != nil {
		return nil, err
	}
	if cdk.Status == "frozen" {
		return cdk, nil
	}
	if cdk.Status != "active" {
		return nil, ErrInvalidOperation
	}
	before := marshalCompact(cdk)
	cdk.Status = "frozen"
	if _, err := tx.Exec(`UPDATE cdks SET status='frozen' WHERE code=?`, code); err != nil {
		return nil, err
	}
	s.appendAuditTx(tx, meta.AdminID, "cdk.freeze", "cdk", code, before, marshalCompact(cdk), meta)
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cdk, nil
}

func (s *MySQLStore) RedeemCDK(code, playerID, requestID, clientIP string) (*CDKRedemption, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	player, err := s.getPlayerTx(tx, playerID, true)
	if err != nil {
		return nil, err
	}
	if existing, ok := s.getRedemptionTx(tx, code, playerID); ok {
		return existing, ErrAlreadyRedeemed
	}
	cdk, err := s.getCDKTx(tx, code, true)
	if err != nil {
		return nil, err
	}
	if cdk.ExpiresAt > 0 && cdk.ExpiresAt < nowMS() {
		return nil, ErrExpired
	}
	if cdk.Status != "active" {
		return nil, ErrInvalidOperation
	}
	if cdk.UsedCount >= cdk.MaxUses {
		return nil, ErrConflict
	}
	batch, err := s.getBatchTx(tx, cdk.BatchID)
	if err != nil {
		return nil, err
	}
	before := marshalCompact(cdk)
	cdk.UsedCount++
	if cdk.UsedCount >= cdk.MaxUses {
		cdk.Status = "used"
	}
	now := nowMS()
	if _, err := tx.Exec(`UPDATE cdks SET used_count=?,status=? WHERE code=?`, cdk.UsedCount, cdk.Status, cdk.Code); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`UPDATE players SET gold=?,updated_at_ms=? WHERE player_id=?`, player.Gold+batch.Gold, now, playerID); err != nil {
		return nil, err
	}
	redemption := &CDKRedemption{ID: newStoreID("redeem"), Code: code, PlayerID: playerID, RewardGold: batch.Gold, RewardItems: append([]string(nil), batch.Items...), RequestID: requestID, CreatedAt: now}
	if _, err := tx.Exec(`INSERT INTO cdk_redemptions (id,code,player_id,reward_gold,reward_items_json,request_id,created_at_ms) VALUES (?,?,?,?,?,?,?)`,
		redemption.ID, redemption.Code, redemption.PlayerID, redemption.RewardGold, encodeJSON(redemption.RewardItems), redemption.RequestID, redemption.CreatedAt); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return redemption, ErrAlreadyRedeemed
		}
		return nil, err
	}
	s.appendAuditTx(tx, "player:"+playerID, "cdk.redeem", "cdk", code, before, marshalCompact(cdk), AuditMeta{AdminID: "player:" + playerID, RequestID: requestID, ClientIP: clientIP})
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return redemption, nil
}

func (s *MySQLStore) AddEvent(eventType, playerID string, payload map[string]any) (*GameEvent, error) {
	if eventType == "" {
		return nil, fmt.Errorf("%w: event type is required", ErrInvalidOperation)
	}
	event := &GameEvent{EventID: newStoreID("event"), Type: eventType, PlayerID: playerID, Payload: payload, CreatedAt: nowMS()}
	_, err := s.db.Exec(`INSERT INTO game_events (event_id,event_type,player_id,payload_json,created_at_ms) VALUES (?,?,?,?,?)`,
		event.EventID, event.Type, event.PlayerID, encodeJSON(event.Payload), event.CreatedAt)
	return event, err
}

func (s *MySQLStore) ListAudits(filter AuditFilter) []AuditLog {
	args := []any{}
	clauses := []string{"1=1"}
	if filter.AdminID != "" {
		clauses = append(clauses, "admin_id=?")
		args = append(args, filter.AdminID)
	}
	if filter.Action != "" {
		clauses = append(clauses, "action=?")
		args = append(args, filter.Action)
	}
	if filter.TargetType != "" {
		clauses = append(clauses, "target_type=?")
		args = append(args, filter.TargetType)
	}
	if filter.TargetID != "" {
		clauses = append(clauses, "target_id=?")
		args = append(args, filter.TargetID)
	}
	if filter.FromMS > 0 {
		clauses = append(clauses, "created_at_ms>=?")
		args = append(args, filter.FromMS)
	}
	if filter.ToMS > 0 {
		clauses = append(clauses, "created_at_ms<=?")
		args = append(args, filter.ToMS)
	}
	rows, err := s.db.Query(`SELECT id,admin_id,action,target_type,target_id,before_json,after_json,request_id,client_ip,agent_session_id,agent_mode,confirmation_id,confirmed_by,confirmed_at_ms,created_at_ms FROM audit_logs WHERE `+strings.Join(clauses, " AND ")+` ORDER BY created_at_ms`, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var audits []AuditLog
	for rows.Next() {
		var audit AuditLog
		if err := rows.Scan(&audit.ID, &audit.AdminID, &audit.Action, &audit.TargetType, &audit.TargetID, &audit.BeforeJSON, &audit.AfterJSON, &audit.RequestID, &audit.ClientIP, &audit.AgentSessionID, &audit.AgentMode, &audit.ConfirmationID, &audit.ConfirmedBy, &audit.ConfirmedAt, &audit.CreatedAt); err == nil {
			audits = append(audits, audit)
		}
	}
	return audits
}

func (s *MySQLStore) Stats() map[string]int {
	tables := map[string]string{
		"players":     "players",
		"mails":       "mails",
		"cdks":        "cdks",
		"redemptions": "cdk_redemptions",
		"events":      "game_events",
		"audits":      "audit_logs",
	}
	stats := map[string]int{}
	for key, table := range tables {
		row := s.db.QueryRow("SELECT COUNT(*) FROM " + table)
		var count int
		_ = row.Scan(&count)
		stats[key] = count
	}
	return stats
}

func (s *MySQLStore) getPlayerTx(tx *sql.Tx, playerID string, lock bool) (*Player, error) {
	query := `SELECT player_id,nickname,level,gold,status,ban_reason,banned_until_ms,real_name_verified,minor,daily_play_seconds,created_at_ms,updated_at_ms FROM players WHERE player_id=?`
	if lock {
		query += " FOR UPDATE"
	}
	player, err := scanPlayer(tx.QueryRow(query, playerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &player, err
}

func (s *MySQLStore) getConfigTx(tx *sql.Tx, key string) (OpsConfig, bool) {
	var cfg OpsConfig
	err := tx.QueryRow(`SELECT config_key,config_value,description,updated_by,updated_at_ms FROM ops_configs WHERE config_key=?`, key).
		Scan(&cfg.Key, &cfg.Value, &cfg.Description, &cfg.UpdatedBy, &cfg.UpdatedAt)
	return cfg, err == nil
}

func (s *MySQLStore) getMailTx(tx *sql.Tx, playerID, mailID string, lock bool) (*Mail, error) {
	query := `SELECT mail_id,player_id,title,body,gold,items_json,status,claimed_at_ms,expires_at_ms,created_by,created_at_ms FROM mails WHERE mail_id=? AND player_id=?`
	if lock {
		query += " FOR UPDATE"
	}
	mail, err := scanMail(tx.QueryRow(query, mailID, playerID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &mail, err
}

func (s *MySQLStore) getCDKTx(tx *sql.Tx, code string, lock bool) (*CDK, error) {
	query := `SELECT code,batch_id,status,used_count,max_uses,expires_at_ms,created_at_ms FROM cdks WHERE code=?`
	if lock {
		query += " FOR UPDATE"
	}
	cdk, err := scanCDK(tx.QueryRow(query, code))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &cdk, err
}

func (s *MySQLStore) getBatchTx(tx *sql.Tx, batchID string) (*CDKBatch, error) {
	var batch CDKBatch
	var itemsJSON string
	err := tx.QueryRow(`SELECT batch_id,name,gold,items_json,status,max_uses_per_code,expires_at_ms,created_by,created_at_ms FROM cdk_batches WHERE batch_id=?`, batchID).
		Scan(&batch.BatchID, &batch.Name, &batch.Gold, &itemsJSON, &batch.Status, &batch.MaxUsesPerCode, &batch.ExpiresAt, &batch.CreatedBy, &batch.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	batch.Items = decodeStringSlice(itemsJSON)
	return &batch, nil
}

func (s *MySQLStore) getRedemptionTx(tx *sql.Tx, code, playerID string) (*CDKRedemption, bool) {
	row := tx.QueryRow(`SELECT id,code,player_id,reward_gold,reward_items_json,request_id,created_at_ms FROM cdk_redemptions WHERE code=? AND player_id=?`, code, playerID)
	redemption, err := scanRedemption(row)
	if err != nil {
		return nil, false
	}
	return &redemption, true
}

func (s *MySQLStore) appendAuditTx(tx *sql.Tx, adminID, action, targetType, targetID, before, after string, meta AuditMeta) {
	if adminID == "" {
		adminID = meta.AdminID
	}
	_, _ = tx.Exec(`INSERT INTO audit_logs (id,admin_id,action,target_type,target_id,before_json,after_json,request_id,client_ip,agent_session_id,agent_mode,confirmation_id,confirmed_by,confirmed_at_ms,created_at_ms) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		newStoreID("audit"), adminID, action, targetType, targetID, before, after, meta.RequestID, meta.ClientIP, meta.Agent.AgentSessionID, meta.Agent.AgentMode, meta.Agent.ConfirmationID, meta.Agent.ConfirmedBy, meta.Agent.ConfirmedAt, nowMS())
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlayer(row rowScanner) (Player, error) {
	var player Player
	err := row.Scan(&player.PlayerID, &player.Nickname, &player.Level, &player.Gold, &player.Status, &player.BanReason, &player.BannedUntil, &player.RealNameVerified, &player.Minor, &player.DailyPlaySeconds, &player.CreatedAt, &player.UpdatedAt)
	return player, err
}

func scanMail(row rowScanner) (Mail, error) {
	var mail Mail
	var itemsJSON string
	err := row.Scan(&mail.MailID, &mail.PlayerID, &mail.Title, &mail.Body, &mail.Gold, &itemsJSON, &mail.Status, &mail.ClaimedAt, &mail.ExpiresAt, &mail.CreatedBy, &mail.CreatedAt)
	mail.Items = decodeStringSlice(itemsJSON)
	return mail, err
}

func scanCDK(row rowScanner) (CDK, error) {
	var cdk CDK
	err := row.Scan(&cdk.Code, &cdk.BatchID, &cdk.Status, &cdk.UsedCount, &cdk.MaxUses, &cdk.ExpiresAt, &cdk.CreatedAt)
	return cdk, err
}

func scanRedemption(row rowScanner) (CDKRedemption, error) {
	var redemption CDKRedemption
	var itemsJSON string
	err := row.Scan(&redemption.ID, &redemption.Code, &redemption.PlayerID, &redemption.RewardGold, &itemsJSON, &redemption.RequestID, &redemption.CreatedAt)
	redemption.RewardItems = decodeStringSlice(itemsJSON)
	return redemption, err
}

func encodeJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(data)
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil
	}
	return result
}

func newStoreID(prefix string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return prefix + "_" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
