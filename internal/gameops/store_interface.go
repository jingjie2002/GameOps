package gameops

type AuditFilter struct {
	AdminID    string
	Action     string
	TargetType string
	TargetID   string
	FromMS     int64
	ToMS       int64
}

type Store interface {
	SeedPlayers(adminID, requestID, clientIP string) []Player
	ListPlayers() []Player
	GetPlayer(playerID string) (*Player, error)
	BanPlayer(playerID, reason string, until int64, adminID, requestID, clientIP string) (*Player, error)
	UnbanPlayer(playerID, adminID, requestID, clientIP string) (*Player, error)
	ListConfigs() []OpsConfig
	UpdateConfig(key, value, description, adminID, requestID, clientIP string) OpsConfig
	OpsState() map[string]string
	CreateMail(playerID, title, body string, gold int64, items []string, adminID, requestID, clientIP string) (*Mail, error)
	ListPlayerMails(playerID string) ([]Mail, error)
	ClaimMail(playerID, mailID, requestID, clientIP string) (*Mail, error)
	CreateCDKBatch(name string, gold int64, items []string, count int, maxUses int, expiresAt int64, adminID, requestID, clientIP string) (*CDKBatch, error)
	ListCDKBatches() []CDKBatch
	GetCDK(code string) (*CDK, error)
	RedeemCDK(code, playerID, requestID, clientIP string) (*CDKRedemption, error)
	AddEvent(eventType, playerID string, payload map[string]any) (*GameEvent, error)
	ListAudits(filter AuditFilter) []AuditLog
	Stats() map[string]int
}
