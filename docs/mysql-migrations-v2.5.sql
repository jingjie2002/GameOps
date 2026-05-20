-- GameOps V2.5 migration for an existing MySQL database.
-- Run these ALTER statements only when the target columns do not already exist.
-- The full fresh-install schema remains internal/gameops/mysql_schema.sql.

ALTER TABLE mails
  ADD COLUMN expires_at_ms BIGINT NOT NULL DEFAULT 0 AFTER claimed_at_ms;

ALTER TABLE cdk_batches
  ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'active' AFTER items_json;

ALTER TABLE audit_logs
  ADD COLUMN agent_session_id VARCHAR(128) NOT NULL DEFAULT '' AFTER client_ip,
  ADD COLUMN agent_mode VARCHAR(64) NOT NULL DEFAULT '' AFTER agent_session_id,
  ADD COLUMN confirmation_id VARCHAR(128) NOT NULL DEFAULT '' AFTER agent_mode,
  ADD COLUMN confirmed_by VARCHAR(128) NOT NULL DEFAULT '' AFTER confirmation_id,
  ADD COLUMN confirmed_at_ms BIGINT NOT NULL DEFAULT 0 AFTER confirmed_by;

CREATE INDEX idx_audit_agent_session ON audit_logs(agent_session_id);
CREATE INDEX idx_audit_confirmation ON audit_logs(confirmation_id);
