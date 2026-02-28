//go:build commonhash

package common

import (
	"testing"
	"time"

	"github.com/hylla/hakoll/internal/app"
	"github.com/hylla/hakoll/internal/domain"
)

// TestComputeCaptureSummaryHashIgnoresCapturedAt verifies the hash excludes capture timestamp jitter.
func TestComputeCaptureSummaryHashIgnoresCapturedAt(t *testing.T) {
	summary := app.CaptureStateSummary{
		CapturedAt: time.Date(2026, 2, 25, 2, 32, 58, 600610000, time.UTC),
		Level: domain.LevelTuple{
			ProjectID: "p1",
			ScopeType: domain.ScopeLevelProject,
			ScopeID:   "p1",
		},
		GoalOverview: "scope=project:p1 project=p1 view=full",
		AttentionOverview: app.CaptureStateAttentionOverview{
			UnresolvedCount: 1,
			Items: []app.CaptureStateAttentionItem{
				{
					ID:                 "a1",
					Kind:               domain.AttentionKindApprovalRequired,
					State:              domain.AttentionStateOpen,
					Summary:            "needs approval",
					RequiresUserAction: true,
					CreatedAt:          time.Date(2026, 2, 25, 2, 32, 54, 0, time.UTC),
				},
			},
		},
		WorkOverview: app.CaptureStateWorkOverview{
			TotalItems:      2,
			ActiveItems:     2,
			InProgressItems: 1,
			DoneItems:       0,
			BlockedItems:    0,
			OpenChildItems:  0,
		},
		FollowUpPointers: app.CaptureStateFollowUpPointers{
			ListAttentionItems:      "list_attention_items(project_id=\"p1\")",
			ListProjectChangeEvents: "list_project_change_events(project_id=\"p1\")",
		},
	}

	hashA, err := computeCaptureSummaryHash(summary)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(first) error = %v", err)
	}
	summary.CapturedAt = time.Date(2026, 2, 25, 2, 32, 58, 607449000, time.UTC)
	hashB, err := computeCaptureSummaryHash(summary)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(second) error = %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch when only captured_at changed: %q != %q", hashA, hashB)
	}
}

// TestComputeCaptureSummaryHashSortsAttentionItems verifies hash stability across item ordering differences.
func TestComputeCaptureSummaryHashSortsAttentionItems(t *testing.T) {
	older := app.CaptureStateAttentionItem{
		ID:                 "a1",
		Kind:               domain.AttentionKindBlocker,
		State:              domain.AttentionStateOpen,
		Summary:            "older",
		RequiresUserAction: false,
		CreatedAt:          time.Date(2026, 2, 25, 2, 31, 0, 0, time.UTC),
	}
	newer := app.CaptureStateAttentionItem{
		ID:                 "a2",
		Kind:               domain.AttentionKindApprovalRequired,
		State:              domain.AttentionStateOpen,
		Summary:            "newer",
		RequiresUserAction: true,
		CreatedAt:          time.Date(2026, 2, 25, 2, 32, 0, 0, time.UTC),
	}
	base := app.CaptureStateSummary{
		Level: domain.LevelTuple{
			ProjectID: "p1",
			ScopeType: domain.ScopeLevelTask,
			ScopeID:   "t1",
		},
		GoalOverview: "scope=task:t1 project=p1 view=summary",
		AttentionOverview: app.CaptureStateAttentionOverview{
			UnresolvedCount: 2,
		},
		WorkOverview: app.CaptureStateWorkOverview{
			TotalItems:      4,
			ActiveItems:     3,
			InProgressItems: 1,
			DoneItems:       1,
			BlockedItems:    1,
			FocusItemID:     "t1",
			OpenChildItems:  1,
		},
		FollowUpPointers: app.CaptureStateFollowUpPointers{
			ListAttentionItems:      "list_attention_items(project_id=\"p1\",scope_id=\"t1\")",
			ListProjectChangeEvents: "list_project_change_events(project_id=\"p1\")",
			ListChildTasks:          "list_child_tasks(project_id=\"p1\",parent_id=\"t1\")",
		},
	}
	first := base
	first.AttentionOverview.Items = []app.CaptureStateAttentionItem{older, newer}
	second := base
	second.AttentionOverview.Items = []app.CaptureStateAttentionItem{newer, older}

	hashA, err := computeCaptureSummaryHash(first)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(first order) error = %v", err)
	}
	hashB, err := computeCaptureSummaryHash(second)
	if err != nil {
		t.Fatalf("computeCaptureSummaryHash(second order) error = %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch for equivalent attention sets: %q != %q", hashA, hashB)
	}
}
