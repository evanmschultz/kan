package domain

import (
	"testing"
	"time"
)

// TestNewKindDefinitionValidation verifies catalog normalization and validation behavior.
func TestNewKindDefinitionValidation(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	kind, err := NewKindDefinition(KindDefinitionInput{
		ID:                  " Refactor ",
		DisplayName:         " Refactor Work ",
		DescriptionMarkdown: " refactor tasks ",
		AppliesTo:           []KindAppliesTo{KindAppliesToTask, KindAppliesToTask, KindAppliesToSubtask},
		AllowedParentScopes: []KindAppliesTo{KindAppliesToPhase},
		PayloadSchemaJSON:   `{"type":"object","required":["package"],"properties":{"package":{"type":"string"}}}`,
		Template: KindTemplate{
			CompletionChecklist: []ChecklistItem{{ID: "c1", Text: "run tests", Done: false}},
			AutoCreateChildren: []KindTemplateChildSpec{{
				Title:     "scan packages",
				Kind:      "task",
				AppliesTo: KindAppliesToSubtask,
			}},
		},
	}, now)
	if err != nil {
		t.Fatalf("NewKindDefinition() error = %v", err)
	}
	if kind.ID != KindID("refactor") {
		t.Fatalf("expected normalized id refactor, got %q", kind.ID)
	}
	if !kind.AppliesToScope(KindAppliesToTask) {
		t.Fatal("expected applies_to task")
	}
	if !kind.AllowsParentScope(KindAppliesToPhase) {
		t.Fatal("expected allowed parent scope phase")
	}
	if len(kind.Template.AutoCreateChildren) != 1 {
		t.Fatalf("expected one child template, got %d", len(kind.Template.AutoCreateChildren))
	}
	if !kind.CreatedAt.Equal(now.UTC()) || !kind.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("expected UTC timestamps, got created=%s updated=%s", kind.CreatedAt, kind.UpdatedAt)
	}
}

// TestNewKindDefinitionRejectsInvalidValues verifies validation errors for malformed entries.
func TestNewKindDefinitionRejectsInvalidValues(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "", AppliesTo: []KindAppliesTo{KindAppliesToTask}}, now); err != ErrInvalidKindID {
		t.Fatalf("expected ErrInvalidKindID, got %v", err)
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesTo("bad")}}, now); err == nil {
		t.Fatal("expected invalid applies_to error")
	}
	if _, err := NewKindDefinition(KindDefinitionInput{ID: "x", AppliesTo: []KindAppliesTo{KindAppliesToTask}, PayloadSchemaJSON: "{"}, now); err != ErrInvalidKindPayloadSchema {
		t.Fatalf("expected ErrInvalidKindPayloadSchema, got %v", err)
	}
}

// TestCapabilityLeaseLifecycle verifies active/expired/revoked lease behavior.
func TestCapabilityLeaseLifecycle(t *testing.T) {
	now := time.Date(2026, 2, 24, 10, 0, 0, 0, time.UTC)
	lease, err := NewCapabilityLease(CapabilityLeaseInput{
		InstanceID: "inst-1",
		LeaseToken: "token-1",
		AgentName:  "orch-1",
		ProjectID:  "p1",
		ScopeType:  CapabilityScopeProject,
		Role:       CapabilityRoleOrchestrator,
		ExpiresAt:  now.Add(time.Hour),
	}, now)
	if err != nil {
		t.Fatalf("NewCapabilityLease() error = %v", err)
	}
	if !lease.IsActive(now.Add(10 * time.Minute)) {
		t.Fatal("expected lease to be active")
	}
	if !lease.MatchesScope(CapabilityScopeTask, "any") {
		t.Fatal("expected project-scope lease to match descendant scope")
	}
	if !lease.MatchesIdentity("orch-1", "token-1") {
		t.Fatal("expected identity match")
	}

	lease.Revoke("manual", now.Add(5*time.Minute))
	if !lease.IsRevoked() {
		t.Fatal("expected revoked lease")
	}
	if lease.IsActive(now.Add(6 * time.Minute)) {
		t.Fatal("expected revoked lease to be inactive")
	}
}
