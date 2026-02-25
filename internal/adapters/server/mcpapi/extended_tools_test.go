package mcpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/evanschultz/kan/internal/adapters/server/common"
	"github.com/evanschultz/kan/internal/domain"
)

// stubExpandedService provides deterministic responses for expanded MCP tool coverage tests.
type stubExpandedService struct {
	stubCaptureStateReader
}

// GetBootstrapGuide returns one deterministic bootstrap payload.
func (s *stubExpandedService) GetBootstrapGuide(_ context.Context) (common.BootstrapGuide, error) {
	return common.BootstrapGuide{
		Mode:    "bootstrap_required",
		Summary: "create project",
	}, nil
}

// ListProjects returns one deterministic project row.
func (s *stubExpandedService) ListProjects(_ context.Context, _ bool) ([]domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Project{
		{
			ID:        "p1",
			Slug:      "proj-1",
			Name:      "Project One",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

// CreateProject returns one deterministic project row.
func (s *stubExpandedService) CreateProject(_ context.Context, _ common.CreateProjectRequest) (domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One", CreatedAt: now, UpdatedAt: now}, nil
}

// UpdateProject returns one deterministic updated project row.
func (s *stubExpandedService) UpdateProject(_ context.Context, _ common.UpdateProjectRequest) (domain.Project, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One Updated", CreatedAt: now, UpdatedAt: now}, nil
}

// ListTasks returns one deterministic task row.
func (s *stubExpandedService) ListTasks(_ context.Context, _ string, _ bool) ([]domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Task{
		{
			ID:             "t1",
			ProjectID:      "p1",
			ColumnID:       "c1",
			Position:       0,
			Title:          "Task One",
			Kind:           domain.WorkKindTask,
			Scope:          domain.KindAppliesToTask,
			LifecycleState: domain.StateTodo,
			Priority:       domain.PriorityMedium,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}, nil
}

// CreateTask returns one deterministic created task row.
func (s *stubExpandedService) CreateTask(_ context.Context, _ common.CreateTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// UpdateTask returns one deterministic updated task row.
func (s *stubExpandedService) UpdateTask(_ context.Context, _ common.UpdateTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One Updated",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MoveTask returns one deterministic moved task row.
func (s *stubExpandedService) MoveTask(_ context.Context, _ common.MoveTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c2",
		Position:       1,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateProgress,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// DeleteTask reports deterministic success.
func (s *stubExpandedService) DeleteTask(_ context.Context, _ common.DeleteTaskRequest) error {
	return nil
}

// RestoreTask returns one deterministic restored row.
func (s *stubExpandedService) RestoreTask(_ context.Context, _ string) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ReparentTask returns one deterministic reparented row.
func (s *stubExpandedService) ReparentTask(_ context.Context, _ common.ReparentTaskRequest) (domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Task{
		ID:             "t1",
		ProjectID:      "p1",
		ParentID:       "parent-1",
		ColumnID:       "c1",
		Position:       0,
		Title:          "Task One",
		Kind:           domain.WorkKindTask,
		Scope:          domain.KindAppliesToTask,
		LifecycleState: domain.StateTodo,
		Priority:       domain.PriorityMedium,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// ListChildTasks returns one deterministic child row.
func (s *stubExpandedService) ListChildTasks(_ context.Context, _, _ string, _ bool) ([]domain.Task, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.Task{
		{
			ID:             "child-1",
			ProjectID:      "p1",
			ParentID:       "parent-1",
			ColumnID:       "c1",
			Position:       0,
			Title:          "Child",
			Kind:           domain.WorkKindSubtask,
			Scope:          domain.KindAppliesToSubtask,
			LifecycleState: domain.StateTodo,
			Priority:       domain.PriorityMedium,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}, nil
}

// SearchTasks returns one deterministic match row.
func (s *stubExpandedService) SearchTasks(_ context.Context, _ common.SearchTasksRequest) ([]common.SearchTaskMatch, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []common.SearchTaskMatch{
		{
			Project: domain.Project{ID: "p1", Slug: "proj-1", Name: "Project One", CreatedAt: now, UpdatedAt: now},
			Task: domain.Task{
				ID:             "t1",
				ProjectID:      "p1",
				ColumnID:       "c1",
				Position:       0,
				Title:          "Task One",
				Kind:           domain.WorkKindTask,
				Scope:          domain.KindAppliesToTask,
				LifecycleState: domain.StateTodo,
				Priority:       domain.PriorityMedium,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			StateID: "todo",
		},
	}, nil
}

// ListProjectChangeEvents returns one deterministic change row.
func (s *stubExpandedService) ListProjectChangeEvents(_ context.Context, _ string, _ int) ([]domain.ChangeEvent, error) {
	return []domain.ChangeEvent{
		{
			ID:         1,
			ProjectID:  "p1",
			WorkItemID: "t1",
			Operation:  domain.ChangeOperationUpdate,
			ActorID:    "tester",
			ActorType:  domain.ActorTypeUser,
			Metadata:   map[string]string{"field": "title"},
			OccurredAt: time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

// GetProjectDependencyRollup returns one deterministic dependency rollup.
func (s *stubExpandedService) GetProjectDependencyRollup(_ context.Context, _ string) (domain.DependencyRollup, error) {
	return domain.DependencyRollup{
		ProjectID:                 "p1",
		TotalItems:                2,
		ItemsWithDependencies:     1,
		DependencyEdges:           1,
		BlockedItems:              1,
		BlockedByEdges:            1,
		UnresolvedDependencyEdges: 1,
	}, nil
}

// ListKindDefinitions returns one deterministic kind row.
func (s *stubExpandedService) ListKindDefinitions(_ context.Context, _ bool) ([]domain.KindDefinition, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return []domain.KindDefinition{
		{
			ID:          domain.KindID("phase"),
			DisplayName: "Phase",
			AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPhase},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}, nil
}

// UpsertKindDefinition returns one deterministic kind row.
func (s *stubExpandedService) UpsertKindDefinition(_ context.Context, _ common.UpsertKindDefinitionRequest) (domain.KindDefinition, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.KindDefinition{
		ID:          domain.KindID("phase"),
		DisplayName: "Phase",
		AppliesTo:   []domain.KindAppliesTo{domain.KindAppliesToPhase},
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// SetProjectAllowedKinds reports deterministic success.
func (s *stubExpandedService) SetProjectAllowedKinds(_ context.Context, _ common.SetProjectAllowedKindsRequest) error {
	return nil
}

// ListProjectAllowedKinds returns deterministic allowlist rows.
func (s *stubExpandedService) ListProjectAllowedKinds(_ context.Context, _ string) ([]string, error) {
	return []string{"phase", "task"}, nil
}

// IssueCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) IssueCapabilityLease(_ context.Context, _ common.IssueCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)
	return domain.CapabilityLease{
		InstanceID:  "inst-1",
		LeaseToken:  "tok-1",
		AgentName:   "agent-1",
		ProjectID:   "p1",
		ScopeType:   domain.CapabilityScopeProject,
		ScopeID:     "p1",
		Role:        domain.CapabilityRoleWorker,
		IssuedAt:    now,
		ExpiresAt:   expiresAt,
		HeartbeatAt: now,
	}, nil
}

// HeartbeatCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) HeartbeatCapabilityLease(_ context.Context, _ common.HeartbeatCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
}

// RenewCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RenewCapabilityLease(_ context.Context, _ common.RenewCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	return s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
}

// RevokeCapabilityLease returns one deterministic lease row.
func (s *stubExpandedService) RevokeCapabilityLease(_ context.Context, _ common.RevokeCapabilityLeaseRequest) (domain.CapabilityLease, error) {
	lease, _ := s.IssueCapabilityLease(context.Background(), common.IssueCapabilityLeaseRequest{})
	now := time.Date(2026, 2, 24, 13, 0, 0, 0, time.UTC)
	lease.RevokedAt = &now
	lease.RevokedReason = "test revoke"
	return lease, nil
}

// RevokeAllCapabilityLeases reports deterministic success.
func (s *stubExpandedService) RevokeAllCapabilityLeases(_ context.Context, _ common.RevokeAllCapabilityLeasesRequest) error {
	return nil
}

// CreateComment returns one deterministic comment row.
func (s *stubExpandedService) CreateComment(_ context.Context, _ common.CreateCommentRequest) (domain.Comment, error) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	return domain.Comment{
		ID:           "c1",
		ProjectID:    "p1",
		TargetType:   domain.CommentTargetTypeTask,
		TargetID:     "t1",
		BodyMarkdown: "hello",
		ActorType:    domain.ActorTypeUser,
		AuthorName:   "tester",
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// ListCommentsByTarget returns one deterministic comment row.
func (s *stubExpandedService) ListCommentsByTarget(_ context.Context, _ common.ListCommentsByTargetRequest) ([]domain.Comment, error) {
	comment, _ := s.CreateComment(context.Background(), common.CreateCommentRequest{})
	return []domain.Comment{comment}, nil
}

// TestHandlerExpandedToolSurfaceSuccessPaths exercises success paths for the expanded MCP tool set.
func TestHandlerExpandedToolSurfaceSuccessPaths(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	_, toolsResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	toolsRaw, ok := toolsResp.Result["tools"].([]any)
	if !ok {
		t.Fatalf("tools list payload missing tools: %#v", toolsResp.Result)
	}
	toolNames := make([]string, 0, len(toolsRaw))
	for _, toolRaw := range toolsRaw {
		toolMap, ok := toolRaw.(map[string]any)
		if !ok {
			continue
		}
		name, _ := toolMap["name"].(string)
		toolNames = append(toolNames, name)
	}
	requiredTools := []string{
		"kan.get_bootstrap_guide",
		"kan.list_projects",
		"kan.create_project",
		"kan.update_project",
		"kan.list_tasks",
		"kan.create_task",
		"kan.update_task",
		"kan.move_task",
		"kan.delete_task",
		"kan.restore_task",
		"kan.reparent_task",
		"kan.list_child_tasks",
		"kan.search_task_matches",
		"kan.list_project_change_events",
		"kan.get_project_dependency_rollup",
		"kan.list_kind_definitions",
		"kan.upsert_kind_definition",
		"kan.set_project_allowed_kinds",
		"kan.list_project_allowed_kinds",
		"kan.issue_capability_lease",
		"kan.heartbeat_capability_lease",
		"kan.renew_capability_lease",
		"kan.revoke_capability_lease",
		"kan.revoke_all_capability_leases",
		"kan.create_comment",
		"kan.list_comments_by_target",
	}
	for _, toolName := range requiredTools {
		found := false
		for _, candidate := range toolNames {
			if candidate == toolName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("tool %q missing from expanded surface: %#v", toolName, toolNames)
		}
	}

	calls := []struct {
		name string
		args map[string]any
	}{
		{name: "kan.get_bootstrap_guide", args: map[string]any{}},
		{name: "kan.list_projects", args: map[string]any{"include_archived": true}},
		{name: "kan.create_project", args: map[string]any{"name": "Project One"}},
		{name: "kan.update_project", args: map[string]any{"project_id": "p1", "name": "Project One Updated"}},
		{name: "kan.list_tasks", args: map[string]any{"project_id": "p1"}},
		{name: "kan.create_task", args: map[string]any{"project_id": "p1", "column_id": "c1", "title": "Task One"}},
		{name: "kan.update_task", args: map[string]any{"task_id": "t1", "title": "Task One Updated"}},
		{name: "kan.move_task", args: map[string]any{"task_id": "t1", "to_column_id": "c2", "position": 1}},
		{name: "kan.delete_task", args: map[string]any{"task_id": "t1"}},
		{name: "kan.restore_task", args: map[string]any{"task_id": "t1"}},
		{name: "kan.reparent_task", args: map[string]any{"task_id": "t1", "parent_id": "parent-1"}},
		{name: "kan.list_child_tasks", args: map[string]any{"project_id": "p1", "parent_id": "parent-1"}},
		{name: "kan.search_task_matches", args: map[string]any{"project_id": "p1", "query": "task"}},
		{name: "kan.list_project_change_events", args: map[string]any{"project_id": "p1", "limit": 25}},
		{name: "kan.get_project_dependency_rollup", args: map[string]any{"project_id": "p1"}},
		{name: "kan.list_kind_definitions", args: map[string]any{}},
		{name: "kan.upsert_kind_definition", args: map[string]any{"id": "phase", "applies_to": []any{"phase"}}},
		{name: "kan.set_project_allowed_kinds", args: map[string]any{"project_id": "p1", "kind_ids": []any{"phase", "task"}}},
		{name: "kan.list_project_allowed_kinds", args: map[string]any{"project_id": "p1"}},
		{name: "kan.issue_capability_lease", args: map[string]any{"project_id": "p1", "scope_type": "project", "role": "worker", "agent_name": "agent-1"}},
		{name: "kan.heartbeat_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1"}},
		{name: "kan.renew_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1", "ttl_seconds": 60}},
		{name: "kan.revoke_capability_lease", args: map[string]any{"agent_instance_id": "inst-1"}},
		{name: "kan.revoke_all_capability_leases", args: map[string]any{"project_id": "p1", "scope_type": "project"}},
		{name: "kan.create_comment", args: map[string]any{"project_id": "p1", "target_type": "task", "target_id": "t1", "body_markdown": "hello"}},
		{name: "kan.list_comments_by_target", args: map[string]any{"project_id": "p1", "target_type": "task", "target_id": "t1"}},
	}
	for idx, tc := range calls {
		resp, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(100+idx, tc.name, tc.args))
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("tool %q status = %d, want %d", tc.name, resp.StatusCode, http.StatusOK)
		}
		if isError, _ := callResp.Result["isError"].(bool); isError {
			t.Fatalf("tool %q returned isError=true: %#v", tc.name, callResp.Result)
		}
	}
}

// TestHandlerExpandedToolInvalidBindArguments verifies bind failures map to invalid_request errors.
func TestHandlerExpandedToolInvalidBindArguments(t *testing.T) {
	service := &stubExpandedService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, service, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(201, "kan.create_project", map[string]any{
		"name": 123,
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}
