package app

import (
	"context"
	"testing"
)

// TestMutationGuardContextRoundTrip verifies normalization and retrieval from context.
func TestMutationGuardContextRoundTrip(t *testing.T) {
	ctx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       " orchestrator ",
		AgentInstanceID: " inst-1 ",
		LeaseToken:      " lease-1 ",
		OverrideToken:   " override ",
	})
	guard, ok := MutationGuardFromContext(ctx)
	if !ok {
		t.Fatal("MutationGuardFromContext() expected guard")
	}
	if guard.AgentName != "orchestrator" {
		t.Fatalf("AgentName = %q, want orchestrator", guard.AgentName)
	}
	if guard.AgentInstanceID != "inst-1" {
		t.Fatalf("AgentInstanceID = %q, want inst-1", guard.AgentInstanceID)
	}
	if guard.LeaseToken != "lease-1" {
		t.Fatalf("LeaseToken = %q, want lease-1", guard.LeaseToken)
	}
	if guard.OverrideToken != "override" {
		t.Fatalf("OverrideToken = %q, want override", guard.OverrideToken)
	}
}

// TestMutationGuardContextEmptyAndRequired verifies absence and required-flag semantics.
func TestMutationGuardContextEmptyAndRequired(t *testing.T) {
	if _, ok := MutationGuardFromContext(context.Background()); ok {
		t.Fatal("MutationGuardFromContext() expected no guard for empty context")
	}
	empty := WithMutationGuard(context.Background(), MutationGuard{})
	if _, ok := MutationGuardFromContext(empty); ok {
		t.Fatal("MutationGuardFromContext() expected no guard for empty value")
	}
	if MutationGuardRequired(context.Background()) {
		t.Fatal("MutationGuardRequired() expected false by default")
	}
	required := WithMutationGuardRequired(context.Background())
	if !MutationGuardRequired(required) {
		t.Fatal("MutationGuardRequired() expected true after marker")
	}
}
