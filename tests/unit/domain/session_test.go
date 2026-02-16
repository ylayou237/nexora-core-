package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestActiveSessionFlow(t *testing.T) {
	clock := NewFakeClock()

	// 1. Setup IDs
	userID, _ := domain.NewUserID("550e8400-e29b-41d4-a716-446655440000")
	sessionID, _ := domain.NewSessionID("session-radius-unique-id")
	mac, _ := domain.NewMAC("AA:BB:CC:DD:EE:FF")

	// 2. Policy (1000 Octets Quota)
	policy := domain.PolicySnapshot{
		DataQuota: 1000,
		TimeQuota: 1 * time.Hour,
	}

	// 3. Create Session (Hot Aggregate)
	// LeaseBuffer = 5 minutes
	session, err := domain.NewActiveSession(
		sessionID,
		userID,
		"10.0.0.1",
		&mac,
		policy,
		5*time.Minute,
		clock,
	)
	if err != nil {
		t.Fatalf("Session creation failed: %v", err)
	}

	// 4. Valid Heartbeat (500 Octets)
	delta1 := domain.UsageDelta{InputOctets: 300, OutputOctets: 200, SessionTime: 60}
	if err := session.Heartbeat(delta1, clock); err != nil {
		t.Errorf("First heartbeat failed: %v", err)
	}
	if session.TotalUsage() != 500 {
		t.Errorf("Expected 500 usage, got %d", session.TotalUsage())
	}

	// 5. Exceed Quota (500 + 600 > 1000)
	delta2 := domain.UsageDelta{InputOctets: 600, OutputOctets: 0, SessionTime: 60}
	err = session.Heartbeat(delta2, clock)

	if err != domain.ErrQuotaExceeded {
		t.Errorf("Expected ErrQuotaExceeded, got %v", err)
	}

	// 6. Zombie Session Check
	// On n'envoie plus de heartbeat pendant 6 minutes (> 5 min lease)
	clock.Advance(6 * time.Minute)

	delta3 := domain.UsageDelta{InputOctets: 10, OutputOctets: 10, SessionTime: 10}
	err = session.Heartbeat(delta3, clock)

	if err != domain.ErrSessionExpired {
		t.Errorf("Expected ErrSessionExpired for zombie session, got %v", err)
	}
}
