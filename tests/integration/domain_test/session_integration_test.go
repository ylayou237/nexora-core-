package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestActiveSessionHeartbeat(t *testing.T) {
	clock := &FakeClock{now: time.Now()}

	userID, _ := domain.NewUserID("123e4567-e89b-12d3-a456-426614174002")
	sessionID, _ := domain.NewSessionID("session-001")

	policy := domain.PolicySnapshot{
		DataQuota:   1024 * 1024, // 1MB
		MaxUpload:   1000,
		MaxDownload: 2000,
		TimeQuota:   3600 * time.Second,
	}

	session, err := domain.NewActiveSession(
		sessionID,
		userID,
		"192.168.1.100",
		nil,
		policy,
		2*time.Minute,
		clock,
	)
	if err != nil {
		t.Fatalf("Failed to create active session: %v", err)
	}

	// Heartbeat test
	delta := domain.UsageDelta{
		InputOctets:  512,
		OutputOctets: 512,
		SessionTime:  60,
	}
	if err := session.Heartbeat(delta, clock); err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if session.TotalUsage() != 1024 {
		t.Errorf("Expected total usage=1024, got %d", session.TotalUsage())
	}

	if !session.IsAlive(clock) {
		t.Errorf("Session should be alive")
	}

	// Simulate expiration
	clock.Advance(3 * time.Minute)
	if session.IsAlive(clock) {
		t.Errorf("Session should have expired")
	}
}
