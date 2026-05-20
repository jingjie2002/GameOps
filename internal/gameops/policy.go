package gameops

import (
	"fmt"
	"strings"
)

const (
	MaxSingleMailGold         int64 = 5000
	MaxMailItemCount                = 10
	MaxMailTargetCount              = 100
	DefaultMailExpiresSeconds int64 = 7 * 24 * 60 * 60
	MinMailExpiresSeconds     int64 = 60
	MaxMailExpiresSeconds     int64 = 30 * 24 * 60 * 60
	MinBanSeconds             int64 = 60
	MaxBanSeconds             int64 = 30 * 24 * 60 * 60
)

func PreviewMailDraft(draft MailDraft, now int64) MailPreview {
	targets := mailTargets(draft)
	expiresInSeconds, expiresAt, expiresErr := resolveMailExpiration(draft.ExpiresInSeconds, now)
	preview := MailPreview{
		TargetCount:      len(targets),
		Gold:             draft.Gold,
		Items:            append([]string(nil), draft.Items...),
		ExpiresInSeconds: expiresInSeconds,
		ExpiresAt:        expiresAt,
	}
	if len(targets) == 0 {
		preview.Violations = append(preview.Violations, "player_id or player_ids is required")
	}
	if len(targets) > MaxMailTargetCount {
		preview.Violations = append(preview.Violations, fmt.Sprintf("target count exceeds limit %d", MaxMailTargetCount))
	}
	if strings.TrimSpace(draft.Title) == "" {
		preview.Violations = append(preview.Violations, "title is required")
	}
	if strings.TrimSpace(draft.Body) == "" {
		preview.Violations = append(preview.Violations, "body is required")
	}
	if draft.Gold < 0 {
		preview.Violations = append(preview.Violations, "gold must not be negative")
	}
	if draft.Gold > MaxSingleMailGold {
		preview.Violations = append(preview.Violations, fmt.Sprintf("gold exceeds single mail limit %d", MaxSingleMailGold))
	}
	if len(draft.Items) > MaxMailItemCount {
		preview.Violations = append(preview.Violations, fmt.Sprintf("item count exceeds limit %d", MaxMailItemCount))
	}
	if expiresErr != nil {
		preview.Violations = append(preview.Violations, expiresErr.Error())
	}
	if draft.Gold >= 1000 {
		preview.Warnings = append(preview.Warnings, "high value reward mail requires approval evidence")
	}
	if len(targets) >= 10 {
		preview.Warnings = append(preview.Warnings, "bulk reward mail requires target list review")
	}
	if len(preview.Violations) > 0 {
		preview.RiskLevel = "blocked"
	} else if draft.Gold >= 3000 || len(targets) >= 50 {
		preview.RiskLevel = "high"
	} else if draft.Gold >= 1000 || len(targets) >= 10 {
		preview.RiskLevel = "medium"
	} else {
		preview.RiskLevel = "low"
	}
	preview.Allowed = len(preview.Violations) == 0
	return preview
}

func validateMailDraft(draft MailDraft, now int64) (MailPreview, error) {
	preview := PreviewMailDraft(draft, now)
	if !preview.Allowed {
		return preview, fmt.Errorf("%w: %s", ErrInvalidOperation, strings.Join(preview.Violations, "; "))
	}
	return preview, nil
}

func validateBanSeconds(seconds int64) error {
	if seconds < MinBanSeconds || seconds > MaxBanSeconds {
		return fmt.Errorf("%w: banned_seconds must be %d..%d", ErrInvalidOperation, MinBanSeconds, MaxBanSeconds)
	}
	return nil
}

func mailTargets(draft MailDraft) []string {
	seen := map[string]bool{}
	var targets []string
	add := func(playerID string) {
		playerID = strings.TrimSpace(playerID)
		if playerID == "" || seen[playerID] {
			return
		}
		seen[playerID] = true
		targets = append(targets, playerID)
	}
	add(draft.PlayerID)
	for _, playerID := range draft.PlayerIDs {
		add(playerID)
	}
	return targets
}

func resolveMailExpiration(expiresInSeconds int64, now int64) (int64, int64, error) {
	if expiresInSeconds == 0 {
		expiresInSeconds = DefaultMailExpiresSeconds
	}
	if expiresInSeconds < MinMailExpiresSeconds || expiresInSeconds > MaxMailExpiresSeconds {
		return expiresInSeconds, 0, fmt.Errorf("expires_in_seconds must be %d..%d", MinMailExpiresSeconds, MaxMailExpiresSeconds)
	}
	return expiresInSeconds, now + expiresInSeconds*1000, nil
}
