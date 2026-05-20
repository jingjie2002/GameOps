package gameops

import (
	"encoding/json"
	"sort"
	"strings"
)

const riskEvidenceLimit = 5

type riskMailAudit struct {
	audit AuditLog
	mail  Mail
}

func AnalyzeOperationalRisk(audits []AuditLog, options RiskAnalyzeOptions) RiskAnalysisReport {
	ordered := append([]AuditLog(nil), audits...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].CreatedAt == ordered[j].CreatedAt {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].CreatedAt < ordered[j].CreatedAt
	})

	window := RiskAnalysisWindow{FromMS: options.FromMS, ToMS: options.ToMS}
	if len(ordered) > 0 {
		if window.FromMS == 0 {
			window.FromMS = ordered[0].CreatedAt
		}
		if window.ToMS == 0 {
			window.ToMS = ordered[len(ordered)-1].CreatedAt
		}
	}

	findings := make([]RiskFinding, 0)
	findings = append(findings, detectRewardMailRisks(ordered)...)
	findings = append(findings, detectCDKRedeemRisks(ordered)...)
	findings = append(findings, detectBanBurstRisks(ordered)...)
	findings = append(findings, detectConfigChurnRisks(ordered)...)
	findings = append(findings, detectSensitiveOperationBurstRisks(ordered)...)
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Score == findings[j].Score {
			return findings[i].Type < findings[j].Type
		}
		return findings[i].Score > findings[j].Score
	})

	provider := options.AIProvider
	if provider == "" {
		provider = "mock-ai"
	}
	if !options.UseAI {
		provider = "rules-only"
	}

	report := RiskAnalysisReport{
		ProjectName:   RiskAssistantProjectName,
		RiskLevel:     riskLevel(findings),
		Score:         riskScore(findings),
		Suggestions:   collectSuggestions(findings),
		Findings:      findings,
		Window:        window,
		AuditCount:    len(ordered),
		EvidenceCount: countEvidence(findings),
		AIProvider:    provider,
		AnalyzedAt:    options.AnalyzedAt,
	}
	if report.AnalyzedAt == 0 {
		report.AnalyzedAt = nowMS()
	}
	if options.UseAI {
		report.Summary = buildMockAISummary(report)
	} else {
		report.Summary = buildRulesOnlySummary(report)
	}
	return report
}

func detectRewardMailRisks(audits []AuditLog) []RiskFinding {
	var mails []riskMailAudit
	byAdmin := map[string][]riskMailAudit{}
	for _, audit := range audits {
		if audit.Action != "mail.create" {
			continue
		}
		var mail Mail
		_ = json.Unmarshal([]byte(audit.AfterJSON), &mail)
		item := riskMailAudit{audit: audit, mail: mail}
		mails = append(mails, item)
		byAdmin[audit.AdminID] = append(byAdmin[audit.AdminID], item)
	}

	findings := make([]RiskFinding, 0)
	for adminID, items := range byAdmin {
		if len(items) < 3 {
			continue
		}
		severity := "medium"
		score := 28
		if len(items) >= 5 {
			severity = "high"
			score = 38
		}
		findings = append(findings, RiskFinding{
			Type:       "high_frequency_reward_mail",
			Severity:   severity,
			Score:      score,
			Reason:     "管理员 " + adminID + " 在分析窗口内创建了 " + strconvFormatInt(int64(len(items))) + " 封奖励邮件，存在误发或越权发奖风险",
			Evidence:   mailEvidence(items),
			Suggestion: "核对奖励邮件是否对应活动补偿审批单，抽查目标玩家名单、金币数量和操作来源 IP。",
		})
	}

	highValue := make([]riskMailAudit, 0)
	for _, item := range mails {
		if item.mail.Gold >= 1000 {
			highValue = append(highValue, item)
		}
	}
	if len(highValue) > 0 {
		maxGold := int64(0)
		for _, item := range highValue {
			if item.mail.Gold > maxGold {
				maxGold = item.mail.Gold
			}
		}
		severity := "medium"
		score := 26
		if maxGold >= 3000 {
			severity = "high"
			score = 36
		}
		findings = append(findings, RiskFinding{
			Type:       "high_value_reward_mail",
			Severity:   severity,
			Score:      score,
			Reason:     "发现 " + strconvFormatInt(int64(len(highValue))) + " 封单封金币不低于 1000 的高额奖励邮件，最高金币为 " + strconvFormatInt(maxGold),
			Evidence:   mailEvidence(highValue),
			Suggestion: "复核高额奖励邮件的业务原因、审批记录和玩家是否属于正常补偿名单。",
		})
	}
	return findings
}

func detectCDKRedeemRisks(audits []AuditLog) []RiskFinding {
	byPlayer := map[string][]AuditLog{}
	for _, audit := range audits {
		if audit.Action != "cdk.redeem" {
			continue
		}
		playerID := strings.TrimPrefix(audit.AdminID, "player:")
		if playerID == "" {
			playerID = audit.AdminID
		}
		byPlayer[playerID] = append(byPlayer[playerID], audit)
	}

	findings := make([]RiskFinding, 0)
	for playerID, items := range byPlayer {
		if len(items) < 2 {
			continue
		}
		severity := "medium"
		score := 28
		if len(items) >= 3 {
			severity = "high"
			score = 36
		}
		findings = append(findings, RiskFinding{
			Type:       "repeated_cdk_redeem",
			Severity:   severity,
			Score:      score,
			Reason:     "同一玩家 " + playerID + " 在分析窗口内完成 " + strconvFormatInt(int64(len(items))) + " 次 CDK 兑换",
			Evidence:   auditEvidence(items, "CDK 兑换记录"),
			Suggestion: "检查 CDK 批次规则、兑换来源、活动投放范围和该玩家近期奖励增长是否异常。",
		})
	}
	return findings
}

func detectBanBurstRisks(audits []AuditLog) []RiskFinding {
	byAdmin := map[string][]AuditLog{}
	for _, audit := range audits {
		if audit.Action == "player.ban" {
			byAdmin[audit.AdminID] = append(byAdmin[audit.AdminID], audit)
		}
	}

	findings := make([]RiskFinding, 0)
	for adminID, items := range byAdmin {
		if len(items) < 2 {
			continue
		}
		severity := "medium"
		score := 30
		if len(items) >= 3 {
			severity = "high"
			score = 40
		}
		findings = append(findings, RiskFinding{
			Type:       "short_time_ban_burst",
			Severity:   severity,
			Score:      score,
			Reason:     "管理员 " + adminID + " 在分析窗口内执行 " + strconvFormatInt(int64(len(items))) + " 次封禁操作",
			Evidence:   auditEvidence(items, "玩家封禁记录"),
			Suggestion: "核查封禁依据、举报单或反作弊证据，确认是否存在批量误封或账号权限被滥用。",
		})
	}
	return findings
}

func detectConfigChurnRisks(audits []AuditLog) []RiskFinding {
	configs := make([]AuditLog, 0)
	byKey := map[string][]AuditLog{}
	for _, audit := range audits {
		if audit.Action != "ops_config.update" {
			continue
		}
		configs = append(configs, audit)
		byKey[audit.TargetID] = append(byKey[audit.TargetID], audit)
	}
	if len(configs) < 3 {
		return nil
	}

	reason := "分析窗口内运营配置共变更 " + strconvFormatInt(int64(len(configs))) + " 次"
	for key, items := range byKey {
		if len(items) >= 2 {
			reason = "运营配置 " + key + " 在分析窗口内被连续变更 " + strconvFormatInt(int64(len(items))) + " 次"
			break
		}
	}
	return []RiskFinding{{
		Type:       "frequent_ops_config_update",
		Severity:   "medium",
		Score:      30,
		Reason:     reason,
		Evidence:   auditEvidence(configs, "运营配置变更记录"),
		Suggestion: "确认维护、公告、活动开关等配置变更是否经过发布审批，并核对变更前后玩家侧影响。",
	}}
}

func detectSensitiveOperationBurstRisks(audits []AuditLog) []RiskFinding {
	byAdmin := map[string][]AuditLog{}
	for _, audit := range audits {
		if !isSensitiveGMAction(audit.Action) {
			continue
		}
		byAdmin[audit.AdminID] = append(byAdmin[audit.AdminID], audit)
	}

	findings := make([]RiskFinding, 0)
	for adminID, items := range byAdmin {
		if len(items) < 6 {
			continue
		}
		findings = append(findings, RiskFinding{
			Type:       "sensitive_operation_burst",
			Severity:   "high",
			Score:      34,
			Reason:     "管理员 " + adminID + " 在分析窗口内连续执行 " + strconvFormatInt(int64(len(items))) + " 次敏感运营操作",
			Evidence:   auditEvidence(items, "敏感运营操作"),
			Suggestion: "优先核查管理员登录来源、操作时间线和审批链路，确认是否为正常活动发布或异常集中操作。",
		})
	}
	return findings
}

func isSensitiveGMAction(action string) bool {
	switch action {
	case "mail.create", "cdk_batch.create", "player.ban", "player.unban", "ops_config.update":
		return true
	default:
		return false
	}
}

func mailEvidence(items []riskMailAudit) []RiskEvidence {
	evidence := make([]RiskEvidence, 0, minInt(len(items), riskEvidenceLimit))
	for i, item := range items {
		if i >= riskEvidenceLimit {
			break
		}
		evidence = append(evidence, RiskEvidence{
			AuditID:    item.audit.ID,
			Action:     item.audit.Action,
			AdminID:    item.audit.AdminID,
			TargetType: item.audit.TargetType,
			TargetID:   item.audit.TargetID,
			CreatedAt:  item.audit.CreatedAt,
			Message:    "奖励邮件 " + item.mail.MailID + " 发给玩家 " + item.mail.PlayerID + "，金币 " + strconvFormatInt(item.mail.Gold),
			Detail: map[string]any{
				"mail_id":   item.mail.MailID,
				"player_id": item.mail.PlayerID,
				"title":     item.mail.Title,
				"gold":      item.mail.Gold,
			},
		})
	}
	return evidence
}

func auditEvidence(items []AuditLog, label string) []RiskEvidence {
	evidence := make([]RiskEvidence, 0, minInt(len(items), riskEvidenceLimit))
	for i, audit := range items {
		if i >= riskEvidenceLimit {
			break
		}
		evidence = append(evidence, RiskEvidence{
			AuditID:    audit.ID,
			Action:     audit.Action,
			AdminID:    audit.AdminID,
			TargetType: audit.TargetType,
			TargetID:   audit.TargetID,
			CreatedAt:  audit.CreatedAt,
			Message:    label + "：" + audit.Action + " -> " + audit.TargetType + "/" + audit.TargetID,
		})
	}
	return evidence
}

func riskScore(findings []RiskFinding) int {
	score := 0
	for _, finding := range findings {
		score += finding.Score
	}
	if score > 100 {
		return 100
	}
	return score
}

func riskLevel(findings []RiskFinding) string {
	score := riskScore(findings)
	switch {
	case score >= 75:
		return "high"
	case score >= 40:
		return "medium"
	case score > 0:
		return "low"
	default:
		return "normal"
	}
}

func countEvidence(findings []RiskFinding) int {
	count := 0
	for _, finding := range findings {
		count += len(finding.Evidence)
	}
	return count
}

func collectSuggestions(findings []RiskFinding) []string {
	seen := map[string]bool{}
	suggestions := make([]string, 0, len(findings))
	for _, finding := range findings {
		if finding.Suggestion == "" || seen[finding.Suggestion] {
			continue
		}
		seen[finding.Suggestion] = true
		suggestions = append(suggestions, finding.Suggestion)
	}
	return suggestions
}

func buildMockAISummary(report RiskAnalysisReport) string {
	if len(report.Findings) == 0 {
		return "mock-ai 摘要：当前分析窗口未命中高风险运营规则，建议保留常规审计巡检。"
	}
	primary := report.Findings[0]
	summary := "mock-ai 摘要：当前分析窗口风险等级为 " + report.RiskLevel + "，规则引擎命中 " + strconvFormatInt(int64(len(report.Findings))) + " 类异常。重点关注：" + primary.Reason + "。"
	if len(report.Findings) > 1 {
		summary += "同时发现：" + report.Findings[1].Reason + "。"
	}
	summary += "建议按证据列表先核对管理员操作来源、业务审批记录和玩家奖励变动。"
	return summary
}

func buildRulesOnlySummary(report RiskAnalysisReport) string {
	if len(report.Findings) == 0 {
		return "规则摘要：当前分析窗口未命中高风险运营规则。"
	}
	return "规则摘要：当前分析窗口命中 " + strconvFormatInt(int64(len(report.Findings))) + " 类异常，风险等级为 " + report.RiskLevel + "，请按建议排查步骤复核。"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
