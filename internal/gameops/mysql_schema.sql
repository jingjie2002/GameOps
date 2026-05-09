CREATE TABLE IF NOT EXISTS players (
  player_id VARCHAR(64) PRIMARY KEY,
  nickname VARCHAR(64) NOT NULL DEFAULT '',
  level INT NOT NULL DEFAULT 0,
  gold BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(32) NOT NULL DEFAULT 'normal',
  ban_reason VARCHAR(255) NOT NULL DEFAULT '',
  banned_until_ms BIGINT NOT NULL DEFAULT 0,
  real_name_verified BOOLEAN NOT NULL DEFAULT FALSE,
  minor BOOLEAN NOT NULL DEFAULT FALSE,
  daily_play_seconds BIGINT NOT NULL DEFAULT 0,
  created_at_ms BIGINT NOT NULL,
  updated_at_ms BIGINT NOT NULL,
  INDEX idx_players_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS player_status_logs (
  id VARCHAR(80) PRIMARY KEY,
  player_id VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  reason VARCHAR(255) NOT NULL DEFAULT '',
  banned_until_ms BIGINT NOT NULL DEFAULT 0,
  created_by VARCHAR(64) NOT NULL,
  created_at_ms BIGINT NOT NULL,
  INDEX idx_player_status_logs_player (player_id, created_at_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS ops_configs (
  config_key VARCHAR(80) PRIMARY KEY,
  config_value TEXT NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  updated_by VARCHAR(64) NOT NULL,
  updated_at_ms BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS mails (
  mail_id VARCHAR(80) PRIMARY KEY,
  player_id VARCHAR(64) NOT NULL,
  title VARCHAR(128) NOT NULL,
  body TEXT NOT NULL,
  gold BIGINT NOT NULL DEFAULT 0,
  items_json TEXT NOT NULL,
  status VARCHAR(32) NOT NULL,
  claimed_at_ms BIGINT NOT NULL DEFAULT 0,
  created_by VARCHAR(64) NOT NULL,
  created_at_ms BIGINT NOT NULL,
  INDEX idx_mails_player_status (player_id, status),
  INDEX idx_mails_created_at (created_at_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS cdk_batches (
  batch_id VARCHAR(80) PRIMARY KEY,
  name VARCHAR(128) NOT NULL,
  gold BIGINT NOT NULL DEFAULT 0,
  items_json TEXT NOT NULL,
  max_uses_per_code INT NOT NULL DEFAULT 1,
  expires_at_ms BIGINT NOT NULL DEFAULT 0,
  created_by VARCHAR(64) NOT NULL,
  created_at_ms BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS cdks (
  code VARCHAR(128) PRIMARY KEY,
  batch_id VARCHAR(80) NOT NULL,
  status VARCHAR(32) NOT NULL,
  used_count INT NOT NULL DEFAULT 0,
  max_uses INT NOT NULL DEFAULT 1,
  expires_at_ms BIGINT NOT NULL DEFAULT 0,
  created_at_ms BIGINT NOT NULL,
  INDEX idx_cdks_batch (batch_id),
  INDEX idx_cdks_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS cdk_redemptions (
  id VARCHAR(80) PRIMARY KEY,
  code VARCHAR(128) NOT NULL,
  player_id VARCHAR(64) NOT NULL,
  reward_gold BIGINT NOT NULL DEFAULT 0,
  reward_items_json TEXT NOT NULL,
  request_id VARCHAR(128) NOT NULL DEFAULT '',
  created_at_ms BIGINT NOT NULL,
  UNIQUE KEY uk_cdk_player (code, player_id),
  INDEX idx_cdk_redemptions_player (player_id, created_at_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS game_events (
  event_id VARCHAR(80) PRIMARY KEY,
  event_type VARCHAR(64) NOT NULL,
  player_id VARCHAR(64) NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL,
  created_at_ms BIGINT NOT NULL,
  INDEX idx_game_events_type_time (event_type, created_at_ms),
  INDEX idx_game_events_player_time (player_id, created_at_ms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS audit_logs (
  id VARCHAR(80) PRIMARY KEY,
  admin_id VARCHAR(64) NOT NULL,
  action VARCHAR(80) NOT NULL,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(128) NOT NULL,
  before_json LONGTEXT NOT NULL,
  after_json LONGTEXT NOT NULL,
  request_id VARCHAR(128) NOT NULL DEFAULT '',
  client_ip VARCHAR(64) NOT NULL DEFAULT '',
  created_at_ms BIGINT NOT NULL,
  INDEX idx_audit_admin_time (admin_id, created_at_ms),
  INDEX idx_audit_action_time (action, created_at_ms),
  INDEX idx_audit_target (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
