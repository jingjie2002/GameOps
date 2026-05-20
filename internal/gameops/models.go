package gameops

type Player struct {
	PlayerID         string `json:"player_id"`
	Nickname         string `json:"nickname"`
	Level            int    `json:"level"`
	Gold             int64  `json:"gold"`
	Status           string `json:"status"`
	BanReason        string `json:"ban_reason,omitempty"`
	BannedUntil      int64  `json:"banned_until,omitempty"`
	RealNameVerified bool   `json:"real_name_verified"`
	Minor            bool   `json:"minor"`
	DailyPlaySeconds int64  `json:"daily_play_seconds"`
	CreatedAt        int64  `json:"created_at"`
	UpdatedAt        int64  `json:"updated_at"`
}

type Mail struct {
	MailID    string   `json:"mail_id"`
	PlayerID  string   `json:"player_id"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Gold      int64    `json:"gold"`
	Items     []string `json:"items"`
	Status    string   `json:"status"`
	ClaimedAt int64    `json:"claimed_at,omitempty"`
	ExpiresAt int64    `json:"expires_at"`
	CreatedBy string   `json:"created_by"`
	CreatedAt int64    `json:"created_at"`
}

type CDKBatch struct {
	BatchID        string   `json:"batch_id"`
	Name           string   `json:"name"`
	Gold           int64    `json:"gold"`
	Items          []string `json:"items"`
	Status         string   `json:"status"`
	MaxUsesPerCode int      `json:"max_uses_per_code"`
	ExpiresAt      int64    `json:"expires_at"`
	CreatedBy      string   `json:"created_by"`
	CreatedAt      int64    `json:"created_at"`
	Codes          []string `json:"codes,omitempty"`
}

type CDK struct {
	Code      string `json:"code"`
	BatchID   string `json:"batch_id"`
	Status    string `json:"status"`
	UsedCount int    `json:"used_count"`
	MaxUses   int    `json:"max_uses"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
}

type CDKRedemption struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	PlayerID    string   `json:"player_id"`
	RewardGold  int64    `json:"reward_gold"`
	RewardItems []string `json:"reward_items"`
	RequestID   string   `json:"request_id"`
	CreatedAt   int64    `json:"created_at"`
}

type OpsConfig struct {
	Key         string `json:"config_key"`
	Value       string `json:"config_value"`
	Description string `json:"description"`
	UpdatedBy   string `json:"updated_by"`
	UpdatedAt   int64  `json:"updated_at"`
}

type GameEvent struct {
	EventID   string         `json:"event_id"`
	Type      string         `json:"type"`
	PlayerID  string         `json:"player_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt int64          `json:"created_at"`
}

type AuditLog struct {
	ID             string `json:"id"`
	AdminID        string `json:"admin_id"`
	Action         string `json:"action"`
	TargetType     string `json:"target_type"`
	TargetID       string `json:"target_id"`
	BeforeJSON     string `json:"before_json,omitempty"`
	AfterJSON      string `json:"after_json,omitempty"`
	RequestID      string `json:"request_id"`
	ClientIP       string `json:"client_ip"`
	AgentSessionID string `json:"agent_session_id,omitempty"`
	AgentMode      string `json:"agent_mode,omitempty"`
	ConfirmationID string `json:"confirmation_id,omitempty"`
	ConfirmedBy    string `json:"confirmed_by,omitempty"`
	ConfirmedAt    int64  `json:"confirmed_at,omitempty"`
	CreatedAt      int64  `json:"created_at"`
}

type AgentAuditFields struct {
	AgentSessionID string `json:"agent_session_id,omitempty"`
	AgentMode      string `json:"agent_mode,omitempty"`
	ConfirmationID string `json:"confirmation_id,omitempty"`
	ConfirmedBy    string `json:"confirmed_by,omitempty"`
	ConfirmedAt    int64  `json:"confirmed_at,omitempty"`
}

type AuditMeta struct {
	AdminID   string
	RequestID string
	ClientIP  string
	Agent     AgentAuditFields
}

type MailDraft struct {
	PlayerID         string
	PlayerIDs        []string
	Title            string
	Body             string
	Gold             int64
	Items            []string
	ExpiresInSeconds int64
}

type MailPreview struct {
	Allowed          bool     `json:"allowed"`
	RiskLevel        string   `json:"risk_level"`
	TargetCount      int      `json:"target_count"`
	Gold             int64    `json:"gold"`
	Items            []string `json:"items"`
	ExpiresInSeconds int64    `json:"expires_in_seconds"`
	ExpiresAt        int64    `json:"expires_at"`
	Violations       []string `json:"violations"`
	Warnings         []string `json:"warnings"`
}
