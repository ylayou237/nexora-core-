package domain_test

import (
	"testing"
	"time"

	"github.com/yvan/nexora-core/internal/core/domain"
)

func TestPlanIntegration(t *testing.T) {
	clock := &FakeClock{now: time.Now()}
	tenantID, _ := domain.NewTenantID("123e4567-e89b-12d3-a456-426614174000")
	planID, _ := domain.NewPlanID("plan-int-001")

	plan, err := domain.NewPlan(
		planID,
		tenantID,
		"Integration Plan",
		"Test Plan Integration",
		1024*1024,
		1000,
		2000,
		5,
		3600*time.Second,
		clock,
	)
	if err != nil {
		t.Fatalf("Failed to create plan: %v", err)
	}

	if plan.MaxSessions() != 5 {
		t.Errorf("Expected maxSessions=5, got %d", plan.MaxSessions())
	}

	// Test update limits
	err = plan.UpdateLimits(2048*1024, 2000, 4000, 10, 7200*time.Second, clock)
	if err != nil {
		t.Fatalf("Failed to update plan limits: %v", err)
	}
	if plan.MaxSessions() != 10 || plan.DataQuota() != 2048*1024 {
		t.Errorf("Plan limits not updated correctly")
	}
}
