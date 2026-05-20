package gameops

const RiskAssistantProjectName = "游戏运营日志 AI 风险分析助手"

type RiskAnalyzeRequest struct {
	FromMS     int64  `json:"from_ms,omitempty"`
	ToMS       int64  `json:"to_ms,omitempty"`
	UseAI      *bool  `json:"use_ai,omitempty"`
	AIProvider string `json:"ai_provider,omitempty"`
}

type RiskAnalyzeOptions struct {
	FromMS     int64
	ToMS       int64
	UseAI      bool
	AIProvider string
	AnalyzedAt int64
}

type RiskAnalysisWindow struct {
	FromMS int64 `json:"from_ms"`
	ToMS   int64 `json:"to_ms"`
}

type RiskAnalysisReport struct {
	ProjectName   string             `json:"project_name"`
	RiskLevel     string             `json:"risk_level"`
	Score         int                `json:"score"`
	Summary       string             `json:"summary"`
	Suggestions   []string           `json:"suggestions"`
	Findings      []RiskFinding      `json:"findings"`
	Window        RiskAnalysisWindow `json:"window"`
	AuditCount    int                `json:"audit_count"`
	EvidenceCount int                `json:"evidence_count"`
	AIProvider    string             `json:"ai_provider"`
	AnalyzedAt    int64              `json:"analyzed_at"`
}

type RiskFinding struct {
	Type       string         `json:"type"`
	Severity   string         `json:"severity"`
	Score      int            `json:"score"`
	Reason     string         `json:"reason"`
	Evidence   []RiskEvidence `json:"evidence"`
	Suggestion string         `json:"suggestion"`
}

type RiskEvidence struct {
	AuditID    string         `json:"audit_id"`
	Action     string         `json:"action"`
	AdminID    string         `json:"admin_id"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	CreatedAt  int64          `json:"created_at"`
	Message    string         `json:"message"`
	Detail     map[string]any `json:"detail,omitempty"`
}
