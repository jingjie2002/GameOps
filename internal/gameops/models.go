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
	CreatedBy string   `json:"created_by"`
	CreatedAt int64    `json:"created_at"`
}

type CDKBatch struct {
	BatchID        string   `json:"batch_id"`
	Name           string   `json:"name"`
	Gold           int64    `json:"gold"`
	Items          []string `json:"items"`
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
	ID         string `json:"id"`
	AdminID    string `json:"admin_id"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	BeforeJSON string `json:"before_json,omitempty"`
	AfterJSON  string `json:"after_json,omitempty"`
	RequestID  string `json:"request_id"`
	ClientIP   string `json:"client_ip"`
	CreatedAt  int64  `json:"created_at"`
}
