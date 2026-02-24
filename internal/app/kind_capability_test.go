package app

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/evanschultz/kan/internal/domain"
)

// boolPtr returns a pointer to one bool value.
func boolPtr(v bool) *bool {
	return &v
}

// newDeterministicService builds a service with deterministic IDs and clock values for tests.
func newDeterministicService(repo *fakeRepo, now time.Time, cfg ServiceConfig) *Service {
	idCounter := 0
	return NewService(repo, func() string {
		idCounter++
		return "id-" + time.Unix(int64(idCounter), 0).UTC().Format("150405")
	}, func() time.Time {
		return now
	}, cfg)
}

// TestServiceSetAndListProjectAllowedKindsValidation verifies allowlist write and list behavior.
func TestServiceSetAndListProjectAllowedKindsValidation(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	project, err := svc.CreateProject(context.Background(), "Kinds", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   nil,
	}); !errors.Is(err, domain.ErrKindNotAllowed) {
		t.Fatalf("SetProjectAllowedKinds(empty) error = %v, want ErrKindNotAllowed", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"unknown-kind"},
	}); !errors.Is(err, domain.ErrKindNotFound) {
		t.Fatalf("SetProjectAllowedKinds(unknown) error = %v, want ErrKindNotFound", err)
	}
	if err := svc.SetProjectAllowedKinds(context.Background(), SetProjectAllowedKindsInput{
		ProjectID: project.ID,
		KindIDs:   []domain.KindID{"task", "phase", "task"},
	}); err != nil {
		t.Fatalf("SetProjectAllowedKinds(valid) error = %v", err)
	}
	kinds, err := svc.ListProjectAllowedKinds(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("ListProjectAllowedKinds() error = %v", err)
	}
	want := []domain.KindID{"phase", "task"}
	if !slices.Equal(kinds, want) {
		t.Fatalf("ListProjectAllowedKinds() = %#v, want %#v", kinds, want)
	}
	if _, err := svc.ListProjectAllowedKinds(context.Background(), ""); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("ListProjectAllowedKinds(empty id) error = %v, want ErrInvalidID", err)
	}
}

// TestServiceListKindDefinitionsAndUpsert verifies upsert and deterministic list sorting behavior.
func TestServiceListKindDefinitionsAndUpsert(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{})

	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "zeta",
		DisplayName: "Zeta",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(create) error = %v", err)
	}
	updated, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "zeta",
		DisplayName: "Alpha",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
	})
	if err != nil {
		t.Fatalf("UpsertKindDefinition(update) error = %v", err)
	}
	if updated.DisplayName != "Alpha" {
		t.Fatalf("DisplayName = %q, want Alpha", updated.DisplayName)
	}
	kinds, err := svc.ListKindDefinitions(context.Background(), false)
	if err != nil {
		t.Fatalf("ListKindDefinitions() error = %v", err)
	}
	if len(kinds) == 0 {
		t.Fatal("ListKindDefinitions() expected non-empty catalog")
	}
	seen := false
	for _, kind := range kinds {
		if kind.ID == "zeta" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatal("ListKindDefinitions() missing upserted kind zeta")
	}
	for idx := 1; idx < len(kinds); idx++ {
		prev := kinds[idx-1]
		next := kinds[idx]
		if prev.DisplayName > next.DisplayName {
			t.Fatalf("kinds not sorted at index %d: %q > %q", idx, prev.DisplayName, next.DisplayName)
		}
	}
}

// TestServiceCapabilityLeaseLifecycleAndRevokeAll verifies lease issue/heartbeat/renew/revoke flows.
func TestServiceCapabilityLeaseLifecycleAndRevokeAll(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		RequireAgentLease: boolPtr(true),
		CapabilityLeaseTTL: time.Hour,
	})

	project, err := svc.CreateProject(context.Background(), "Leases", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
			Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-1",
		AgentInstanceID: "agent-1-instance",
		RequestedTTL:    30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	if _, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      "wrong-token",
	}); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("HeartbeatCapabilityLease(wrong token) error = %v, want ErrMutationLeaseInvalid", err)
	}
	heartbeatLease, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err != nil {
		t.Fatalf("HeartbeatCapabilityLease() error = %v", err)
	}
	if heartbeatLease.HeartbeatAt.IsZero() {
		t.Fatal("HeartbeatCapabilityLease() expected HeartbeatAt")
	}
	renewed, err := svc.RenewCapabilityLease(context.Background(), RenewCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
		TTL:             2 * time.Hour,
	})
	if err != nil {
		t.Fatalf("RenewCapabilityLease() error = %v", err)
	}
	if !renewed.ExpiresAt.After(lease.ExpiresAt) {
		t.Fatalf("RenewCapabilityLease() expiry %v must be after %v", renewed.ExpiresAt, lease.ExpiresAt)
	}
	revoked, err := svc.RevokeCapabilityLease(context.Background(), RevokeCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		Reason:          "manual revoke",
	})
	if err != nil {
		t.Fatalf("RevokeCapabilityLease() error = %v", err)
	}
	if !revoked.IsRevoked() {
		t.Fatal("RevokeCapabilityLease() expected revoked lease")
	}
	if _, err := svc.HeartbeatCapabilityLease(context.Background(), HeartbeatCapabilityLeaseInput{
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	}); !errors.Is(err, domain.ErrMutationLeaseRevoked) {
		t.Fatalf("HeartbeatCapabilityLease(revoked) error = %v, want ErrMutationLeaseRevoked", err)
	}

	second, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
			Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-2",
		AgentInstanceID: "agent-2-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(second) error = %v", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: "",
		ScopeType: domain.CapabilityScopeProject,
		ScopeID:   project.ID,
	}); !errors.Is(err, domain.ErrInvalidID) {
		t.Fatalf("RevokeAllCapabilityLeases(empty project) error = %v, want ErrInvalidID", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeType("bad"),
		ScopeID:   project.ID,
	}); !errors.Is(err, domain.ErrInvalidCapabilityScope) {
		t.Fatalf("RevokeAllCapabilityLeases(bad scope) error = %v, want ErrInvalidCapabilityScope", err)
	}
	if err := svc.RevokeAllCapabilityLeases(context.Background(), RevokeAllCapabilityLeasesInput{
		ProjectID: project.ID,
		ScopeType: domain.CapabilityScopeProject,
		ScopeID:   project.ID,
	}); err != nil {
		t.Fatalf("RevokeAllCapabilityLeases() error = %v", err)
	}
	storedSecond, err := repo.GetCapabilityLease(context.Background(), second.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(second) error = %v", err)
	}
	if !storedSecond.IsRevoked() {
		t.Fatal("RevokeAllCapabilityLeases() expected second lease to be revoked")
	}
}

// TestServiceEnforceMutationGuardBranches covers principal mutation-guard failure and success branches.
func TestServiceEnforceMutationGuardBranches(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{
		RequireAgentLease: boolPtr(true),
		CapabilityLeaseTTL: time.Hour,
	})

	project, err := svc.CreateProject(context.Background(), "Guarded", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if err := svc.enforceMutationGuard(context.Background(), project.ID, domain.ActorTypeUser, domain.CapabilityScopeProject, project.ID); err != nil {
		t.Fatalf("enforceMutationGuard(user) error = %v", err)
	}
	if err := svc.enforceMutationGuard(context.Background(), project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseRequired) {
		t.Fatalf("enforceMutationGuard(no guard) error = %v, want ErrMutationLeaseRequired", err)
	}

	missingCtx := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "agent-x",
		AgentInstanceID: "missing",
		LeaseToken:      "missing-token",
	})
	if err := svc.enforceMutationGuard(missingCtx, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(missing lease) error = %v, want ErrMutationLeaseInvalid", err)
	}

	lease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
			Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-y",
		AgentInstanceID: "agent-y-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease() error = %v", err)
	}
	badIdentity := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       "other-name",
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(badIdentity, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(identity mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}

	validGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       lease.AgentName,
		AgentInstanceID: lease.InstanceID,
		LeaseToken:      lease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(validGuard, "wrong-project", domain.ActorTypeAgent, domain.CapabilityScopeProject, "wrong-project"); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(project mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}

	lease.Revoke("revoked", now)
	if err := repo.UpdateCapabilityLease(context.Background(), lease); err != nil {
		t.Fatalf("UpdateCapabilityLease(revoke) error = %v", err)
	}
	if err := svc.enforceMutationGuard(validGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseRevoked) {
		t.Fatalf("enforceMutationGuard(revoked) error = %v, want ErrMutationLeaseRevoked", err)
	}

	expired, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeProject,
		ScopeID:         project.ID,
			Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-z",
		AgentInstanceID: "agent-z-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(expired) error = %v", err)
	}
	expired.ExpiresAt = now.Add(-time.Minute)
	if err := repo.UpdateCapabilityLease(context.Background(), expired); err != nil {
		t.Fatalf("UpdateCapabilityLease(expired) error = %v", err)
	}
	expiredGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       expired.AgentName,
		AgentInstanceID: expired.InstanceID,
		LeaseToken:      expired.LeaseToken,
	})
	if err := svc.enforceMutationGuard(expiredGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseExpired) {
		t.Fatalf("enforceMutationGuard(expired) error = %v, want ErrMutationLeaseExpired", err)
	}

	branchLease, err := svc.IssueCapabilityLease(context.Background(), IssueCapabilityLeaseInput{
		ProjectID:       project.ID,
		ScopeType:       domain.CapabilityScopeBranch,
		ScopeID:         "branch-1",
			Role:            domain.CapabilityRoleWorker,
		AgentName:       "agent-branch",
		AgentInstanceID: "agent-branch-instance",
	})
	if err != nil {
		t.Fatalf("IssueCapabilityLease(branch) error = %v", err)
	}
	branchGuard := WithMutationGuard(context.Background(), MutationGuard{
		AgentName:       branchLease.AgentName,
		AgentInstanceID: branchLease.InstanceID,
		LeaseToken:      branchLease.LeaseToken,
	})
	if err := svc.enforceMutationGuard(branchGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeProject, project.ID); !errors.Is(err, domain.ErrMutationLeaseInvalid) {
		t.Fatalf("enforceMutationGuard(scope mismatch) error = %v, want ErrMutationLeaseInvalid", err)
	}
	if err := svc.enforceMutationGuard(branchGuard, project.ID, domain.ActorTypeAgent, domain.CapabilityScopeBranch, "branch-1"); err != nil {
		t.Fatalf("enforceMutationGuard(scope match) error = %v", err)
	}
	storedBranch, err := repo.GetCapabilityLease(context.Background(), branchLease.InstanceID)
	if err != nil {
		t.Fatalf("GetCapabilityLease(branch) error = %v", err)
	}
	if storedBranch.HeartbeatAt.IsZero() {
		t.Fatal("enforceMutationGuard(scope match) expected heartbeat update")
	}
}

// TestCreateTaskAppliesKindTemplateActions verifies checklist merge and child auto-create behavior.
func TestCreateTaskAppliesKindTemplateActions(t *testing.T) {
	repo := newFakeRepo()
	now := time.Date(2026, 2, 24, 9, 0, 0, 0, time.UTC)
	svc := newDeterministicService(repo, now, ServiceConfig{DefaultDeleteMode: DeleteModeArchive})

	// Bootstrap built-in kinds first so project creation can resolve the default project kind.
	if _, err := svc.ListKindDefinitions(context.Background(), false); err != nil {
		t.Fatalf("ListKindDefinitions(bootstrap) error = %v", err)
	}
	if _, err := svc.UpsertKindDefinition(context.Background(), CreateKindDefinitionInput{
		ID:          "refactor",
		DisplayName: "Refactor",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToTask},
		Template: domain.KindTemplate{
			CompletionChecklist: []domain.ChecklistItem{
				{ID: "ck-run-tests", Text: "run package tests", Done: false},
			},
			AutoCreateChildren: []domain.KindTemplateChildSpec{
				{
					Title:       "Template Child",
					Description: "Auto-created child",
					Kind:        domain.KindID(domain.WorkKindSubtask),
					AppliesTo:   domain.KindAppliesToSubtask,
					Labels:      []string{"templated"},
				},
			},
		},
	}); err != nil {
		t.Fatalf("UpsertKindDefinition(refactor) error = %v", err)
	}

	project, err := svc.CreateProject(context.Background(), "Template Project", "")
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	column, err := svc.CreateColumn(context.Background(), project.ID, "To Do", 0, 0)
	if err != nil {
		t.Fatalf("CreateColumn() error = %v", err)
	}
	parent, err := svc.CreateTask(context.Background(), CreateTaskInput{
		ProjectID:   project.ID,
		ColumnID:    column.ID,
		Title:       "Parent Task",
		Description: "Template parent",
		Kind:        domain.WorkKind("refactor"),
		Scope:       domain.KindAppliesToTask,
	})
	if err != nil {
		t.Fatalf("CreateTask(refactor) error = %v", err)
	}
	storedParent, err := repo.GetTask(context.Background(), parent.ID)
	if err != nil {
		t.Fatalf("GetTask(parent) error = %v", err)
	}
	if len(storedParent.Metadata.CompletionContract.CompletionChecklist) != 1 {
		t.Fatalf("parent checklist len = %d, want 1", len(storedParent.Metadata.CompletionContract.CompletionChecklist))
	}

	tasks, err := svc.ListTasks(context.Background(), project.ID, true)
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	foundChild := false
	for _, task := range tasks {
		if task.ParentID == parent.ID && task.Title == "Template Child" {
			foundChild = true
			if task.Kind != domain.WorkKindSubtask {
				t.Fatalf("child kind = %q, want subtask", task.Kind)
			}
			if task.Scope != domain.KindAppliesToSubtask {
				t.Fatalf("child scope = %q, want subtask", task.Scope)
			}
		}
	}
	if !foundChild {
		t.Fatal("expected template-created child task")
	}
}

// TestKindCapabilityHelpers verifies deterministic helper behavior used by service methods.
func TestKindCapabilityHelpers(t *testing.T) {
	normalized := normalizeKindIDList([]domain.KindID{"Task", "phase", "task", "  ", "Phase"})
	wantIDs := []domain.KindID{"phase", "task"}
	if !slices.Equal(normalized, wantIDs) {
		t.Fatalf("normalizeKindIDList() = %#v, want %#v", normalized, wantIDs)
	}

	hashA := hashSchema(`{"type":"object"}`)
	hashB := hashSchema(`{"type":"object"}`)
	hashC := hashSchema(`{"type":"string"}`)
	if hashA != hashB {
		t.Fatalf("hashSchema() expected deterministic hash, got %q vs %q", hashA, hashB)
	}
	if hashA == hashC {
		t.Fatalf("hashSchema() expected different hash for different schema, got %q", hashA)
	}

	existing := []domain.ChecklistItem{{ID: "a", Text: "existing"}}
	incoming := []domain.ChecklistItem{{ID: "a", Text: "duplicate"}, {ID: "b", Text: "new"}, {ID: "", Text: "skip"}}
	merged := mergeChecklistItems(existing, incoming)
	if len(merged) != 2 {
		t.Fatalf("mergeChecklistItems() len = %d, want 2", len(merged))
	}

	if _, err := normalizeTaskMetadataFromKindPayload(json.RawMessage(`{`)); !errors.Is(err, domain.ErrInvalidKindPayload) {
		t.Fatalf("normalizeTaskMetadataFromKindPayload(invalid) error = %v, want ErrInvalidKindPayload", err)
	}
	meta, err := normalizeTaskMetadataFromKindPayload(json.RawMessage(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("normalizeTaskMetadataFromKindPayload(valid) error = %v", err)
	}
	if string(meta.KindPayload) != `{"key":"value"}` {
		t.Fatalf("KindPayload = %s, want {\"key\":\"value\"}", string(meta.KindPayload))
	}
}
