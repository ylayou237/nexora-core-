package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestPlanUpdates(t *testing.T) {
	clock := NewFakeClock()

	planID, _ := domain.NewPlanID("plan_silver")
	tenantID, _ := domain.NewTenantID("11111111-1111-4111-8111-111111111111")

	plan, err := domain.NewPlan(
		planID,
		tenantID,
		"Silver Plan",
		"For testing",
		500*1024*1024,   // 500MB
		10000,           // Up
		50000,           // Down
		2,               // Max Sessions
		30*24*time.Hour, // Time Quota
		clock,
	)
	if err != nil {
		t.Fatalf("Plan creation failed: %v", err)
	}

	// Test Limits Update
	err = plan.UpdateLimits(
		1024*1024*1024, // 1GB
		20000,
		100000,
		3, // Max Sessions incrémenté
		30*24*time.Hour,
		clock,
	)
	if err != nil {
		t.Errorf("UpdateLimits failed: %v", err)
	}

	if plan.MaxSessions() != 3 {
		t.Errorf("Expected 3 sessions, got %d", plan.MaxSessions())
	}
	if plan.Version() != 2 {
		t.Errorf("Expected version 2, got %d", plan.Version())
	}
}
