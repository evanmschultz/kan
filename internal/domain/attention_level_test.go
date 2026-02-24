package domain

import (
	"errors"
	"testing"
	"time"
)

// TestNewLevelTupleSubphaseSupport verifies subphase-aware level tuple normalization.
func TestNewLevelTupleSubphaseSupport(t *testing.T) {
	level, err := NewLevelTuple(LevelTupleInput{
		ProjectID: " p1 ",
		BranchID:  " ",
		ScopeType: ScopeLevelSubphase,
		ScopeID:   " subphase-1 ",
	})
	if err != nil {
		t.Fatalf("NewLevelTuple() error = %v", err)
	}
	if level.ProjectID != "p1" {
		t.Fatalf("expected trimmed project id, got %q", level.ProjectID)
	}
	if level.ScopeType != ScopeLevelSubphase {
		t.Fatalf("expected subphase scope type, got %q", level.ScopeType)
	}
	if level.ScopeID != "subphase-1" {
		t.Fatalf("expected trimmed scope id, got %q", level.ScopeID)
	}

	projectLevel, err := NewLevelTuple(LevelTupleInput{ProjectID: "p2"})
	if err != nil {
		t.Fatalf("NewLevelTuple(project default) error = %v", err)
	}
	if projectLevel.ScopeType != ScopeLevelProject {
		t.Fatalf("expected project scope type default, got %q", projectLevel.ScopeType)
	}
	if projectLevel.ScopeID != "p2" {
		t.Fatalf("expected scope id to default to project id, got %q", projectLevel.ScopeID)
	}

	if _, err := NewLevelTuple(LevelTupleInput{
		ProjectID: "p1",
		ScopeType: ScopeLevelTask,
	}); !errors.Is(err, ErrInvalidScopeID) {
		t.Fatalf("expected ErrInvalidScopeID, got %v", err)
	}
}

// TestSubphaseScopeCompatibility verifies subphase support across scope type systems.
func TestSubphaseScopeCompatibility(t *testing.T) {
	if !IsValidKindAppliesTo(KindAppliesToSubphase) {
		t.Fatal("expected subphase to be valid for kind definitions")
	}
	if !IsValidWorkItemAppliesTo(KindAppliesToSubphase) {
		t.Fatal("expected subphase to be valid for work-item scope")
	}
	if !IsValidCapabilityScopeType(CapabilityScopeSubphase) {
		t.Fatal("expected subphase to be valid capability scope")
	}
	if ScopeLevelFromKindAppliesTo(KindAppliesToSubphase) != ScopeLevelSubphase {
		t.Fatalf("expected kind->level conversion to return subphase")
	}
	if ScopeLevelSubphase.ToCapabilityScopeType() != CapabilityScopeSubphase {
		t.Fatalf("expected level->capability conversion to return subphase")
	}
}

// TestAttentionItemLifecycleAndBlocking verifies attention normalization and completion-block semantics.
func TestAttentionItemLifecycleAndBlocking(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	item, err := NewAttentionItem(AttentionItemInput{
		ID:                 "attn-1",
		ProjectID:          "p1",
		ScopeType:          ScopeLevelTask,
		ScopeID:            "t1",
		Kind:               AttentionKindBlocker,
		Summary:            "need user decision",
		RequiresUserAction: true,
		CreatedByActor:     "agent-1",
		CreatedByType:      ActorTypeAgent,
	}, now)
	if err != nil {
		t.Fatalf("NewAttentionItem() error = %v", err)
	}
	if item.State != AttentionStateOpen {
		t.Fatalf("expected default open state, got %q", item.State)
	}
	if !item.BlocksCompletion() {
		t.Fatal("expected unresolved blocker to block completion")
	}

	if err := item.Resolve("user-1", ActorTypeUser, now.Add(time.Minute)); err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if item.IsUnresolved() {
		t.Fatal("expected resolved item to be resolved")
	}
	if item.BlocksCompletion() {
		t.Fatal("expected resolved item not to block completion")
	}
}

// TestNormalizeAttentionListFilter verifies list-filter validation and normalization.
func TestNormalizeAttentionListFilter(t *testing.T) {
	requiresUserAction := true
	filter, err := NormalizeAttentionListFilter(AttentionListFilter{
		ProjectID:          " p1 ",
		ScopeType:          ScopeLevelTask,
		ScopeID:            " t1 ",
		UnresolvedOnly:     true,
		States:             []AttentionState{AttentionStateOpen, AttentionState(" OPEN "), AttentionStateResolved},
		Kinds:              []AttentionKind{AttentionKindBlocker, AttentionKind(" blocker "), AttentionKindRiskNote},
		RequiresUserAction: &requiresUserAction,
		Limit:              -10,
	})
	if err != nil {
		t.Fatalf("NormalizeAttentionListFilter() error = %v", err)
	}
	if filter.ProjectID != "p1" || filter.ScopeID != "t1" {
		t.Fatalf("expected trimmed ids, got %#v", filter)
	}
	if len(filter.States) != 2 {
		t.Fatalf("expected deduplicated states, got %#v", filter.States)
	}
	if len(filter.Kinds) != 2 {
		t.Fatalf("expected deduplicated kinds, got %#v", filter.Kinds)
	}
	if filter.Limit != 0 {
		t.Fatalf("expected negative limit clamp to 0, got %d", filter.Limit)
	}

	_, err = NormalizeAttentionListFilter(AttentionListFilter{
		ProjectID: "p1",
		ScopeID:   "t1",
	})
	if !errors.Is(err, ErrInvalidScopeType) {
		t.Fatalf("expected ErrInvalidScopeType, got %v", err)
	}
}
