package service

import (
	"context"
	"fmt"
	"log"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nununugraha/sains-api/internal/repository"
)

// AnomalyService handles anomaly detection based on login patterns.
// All detection is DB-based — no client fingerprinting needed.
type AnomalyService struct {
	queries *repository.Queries
}

// NewAnomalyService creates a new AnomalyService.
func NewAnomalyService(queries *repository.Queries) *AnomalyService {
	return &AnomalyService{queries: queries}
}

// Anomaly event types and scores
const (
	EventSessionDisplaced = "session_displaced" // +5: another login while session active
	EventIPChange         = "ip_change_fast"    // +8: IP changed within 1 hour
	EventCountryChange    = "country_change"    // +15: different country within 24h
	EventMultiDeviceLogin = "multi_device"      // +3: different user-agent same day

	ScoreSessionDisplaced = 5
	ScoreIPChange         = 8
	ScoreCountryChange    = 15
	ScoreMultiDeviceLogin = 3
)

// Threshold actions
const (
	ThresholdWarning  = 10 // log only, visible in admin
	ThresholdTempLock = 25 // temporary lock 24h
	ThresholdBlock    = 50 // permanent block, manual admin unlock
)

// CheckLoginAnomaly analyzes a login event and logs anomalies.
// Called after every successful user login. All data from sessions table.
func (s *AnomalyService) CheckLoginAnomaly(ctx context.Context, userID pgtype.UUID, currentIP string, currentUA string) {
	// Get user's previous sessions to compare
	prevSession, err := s.queries.GetActiveSessionByUserID(ctx, userID)
	if err != nil {
		// No previous session = first login, nothing to check
		return
	}

	// 1. Session displaced - someone was already logged in
	s.logAnomaly(ctx, userID, EventSessionDisplaced, ScoreSessionDisplaced,
		fmt.Sprintf("Previous session was active (id=%s)", prevSession.ID.Bytes))

	// 2. IP change - different IP from last login
	prevIP := ""
	if prevSession.IpAtLogin != nil {
		prevIP = prevSession.IpAtLogin.String()
	}
	if prevIP != "" && currentIP != "" && prevIP != currentIP {
		s.logAnomaly(ctx, userID, EventIPChange, ScoreIPChange,
			fmt.Sprintf("IP changed: %s → %s", maskIP(prevIP), maskIP(currentIP)))
	}

	// 3. Different user-agent (device change indicator)
	prevUA := prevSession.UserAgent.String
	if prevUA != "" && currentUA != "" && prevUA != currentUA {
		s.logAnomaly(ctx, userID, EventMultiDeviceLogin, ScoreMultiDeviceLogin,
			fmt.Sprintf("User-Agent changed"))
	}

	// Check total score and take action
	s.evaluateScore(ctx, userID)
}

// logAnomaly writes an anomaly event to the database.
func (s *AnomalyService) logAnomaly(ctx context.Context, userID pgtype.UUID, event string, score int, detail string) {
	_, err := s.queries.CreateAnomalyLog(ctx, repository.CreateAnomalyLogParams{
		UserID:     userID,
		Event:      event,
		ScoreDelta: int32(score),
		Detail:     []byte(`{"msg":"` + detail + `"}`),
	})
	if err != nil {
		log.Printf("⚠️ Failed to log anomaly: %v", err)
	}
}

// evaluateScore checks the user's total anomaly score and takes action.
func (s *AnomalyService) evaluateScore(ctx context.Context, userID pgtype.UUID) {
	score, err := s.queries.GetUserAnomalyScore(ctx, userID)
	if err != nil {
		return
	}

	switch {
	case score >= int64(ThresholdBlock):
		// Auto-lock account
		_ = s.queries.SetUserActive(ctx, repository.SetUserActiveParams{
			IsActive: pgtype.Bool{Bool: false, Valid: true},
			ID:       userID,
		})
		log.Printf("🔒 User %v auto-locked (anomaly score: %d)", userID.Bytes, score)

	case score >= int64(ThresholdTempLock):
		// Temporary lock (revoke all sessions)
		_ = s.queries.RevokeAllUserSessions(ctx, userID)
		log.Printf("⚠️ User %v sessions revoked (anomaly score: %d)", userID.Bytes, score)

	case score >= int64(ThresholdWarning):
		log.Printf("👀 User %v flagged (anomaly score: %d)", userID.Bytes, score)
	}
}

// maskIP partially hides an IP for logging (privacy)
func maskIP(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return "***"
	}
	if addr.Is4() {
		// 192.168.1.100 → 192.168.1.***
		b := addr.As4()
		return fmt.Sprintf("%d.%d.%d.***", b[0], b[1], b[2])
	}
	return ip[:len(ip)/2] + "***"
}
