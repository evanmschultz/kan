package mcpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hylla/hakoll/internal/adapters/server/common"
	"github.com/hylla/hakoll/internal/domain"
)

// stubExpandedService provides deterministic responses for expanded MCP tool coverage tests.
type stubExpandedService struct {
	stubCaptureStateReader
	lastUpdateTaskReq  common.UpdateTaskRequest
	lastRestoreTaskReq common.RestoreTaskRequest
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
func (s *stubExpandedService) UpdateTask(_ context.Context, in common.UpdateTaskRequest) (domain.Task, error) {
	s.lastUpdateTaskReq = in
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
func (s *stubExpandedService) RestoreTask(_ context.Context, in common.RestoreTaskRequest) (domain.Task, error) {
	s.lastRestoreTaskReq = in
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
		"koll.get_bootstrap_guide",
		"koll.list_projects",
		"koll.create_project",
		"koll.update_project",
		"koll.list_tasks",
		"koll.create_task",
		"koll.update_task",
		"koll.move_task",
		"koll.delete_task",
		"koll.restore_task",
		"koll.reparent_task",
		"koll.list_child_tasks",
		"koll.search_task_matches",
		"koll.list_project_change_events",
		"koll.get_project_dependency_rollup",
		"koll.list_kind_definitions",
		"koll.upsert_kind_definition",
		"koll.set_project_allowed_kinds",
		"koll.list_project_allowed_kinds",
		"koll.issue_capability_lease",
		"koll.heartbeat_capability_lease",
		"koll.renew_capability_lease",
		"koll.revoke_capability_lease",
		"koll.revoke_all_capability_leases",
		"koll.create_comment",
		"koll.list_comments_by_target",
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
		{name: "koll.get_bootstrap_guide", args: map[string]any{}},
		{name: "koll.list_projects", args: map[string]any{"include_archived": true}},
		{name: "koll.create_project", args: map[string]any{"name": "Project One"}},
		{name: "koll.update_project", args: map[string]any{"project_id": "p1", "name": "Project One Updated"}},
		{name: "koll.list_tasks", args: map[string]any{"project_id": "p1"}},
		{name: "koll.create_task", args: map[string]any{"project_id": "p1", "column_id": "c1", "title": "Task One"}},
		{name: "koll.update_task", args: map[string]any{"task_id": "t1", "title": "Task One Updated"}},
		{name: "koll.move_task", args: map[string]any{"task_id": "t1", "to_column_id": "c2", "position": 1}},
		{name: "koll.delete_task", args: map[string]any{"task_id": "t1"}},
		{name: "koll.restore_task", args: map[string]any{"task_id": "t1"}},
		{name: "koll.reparent_task", args: map[string]any{"task_id": "t1", "parent_id": "parent-1"}},
		{name: "koll.list_child_tasks", args: map[string]any{"project_id": "p1", "parent_id": "parent-1"}},
		{name: "koll.search_task_matches", args: map[string]any{"project_id": "p1", "query": "task"}},
		{name: "koll.list_project_change_events", args: map[string]any{"project_id": "p1", "limit": 25}},
		{name: "koll.get_project_dependency_rollup", args: map[string]any{"project_id": "p1"}},
		{name: "koll.list_kind_definitions", args: map[string]any{}},
		{name: "koll.upsert_kind_definition", args: map[string]any{"id": "phase", "applies_to": []any{"phase"}}},
		{name: "koll.set_project_allowed_kinds", args: map[string]any{"project_id": "p1", "kind_ids": []any{"phase", "task"}}},
		{name: "koll.list_project_allowed_kinds", args: map[string]any{"project_id": "p1"}},
		{name: "koll.issue_capability_lease", args: map[string]any{"project_id": "p1", "scope_type": "project", "role": "worker", "agent_name": "agent-1"}},
		{name: "koll.heartbeat_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1"}},
		{name: "koll.renew_capability_lease", args: map[string]any{"agent_instance_id": "inst-1", "lease_token": "tok-1", "ttl_seconds": 60}},
		{name: "koll.revoke_capability_lease", args: map[string]any{"agent_instance_id": "inst-1"}},
		{name: "koll.revoke_all_capability_leases", args: map[string]any{"project_id": "p1", "scope_type": "project"}},
		{name: "koll.create_comment", args: map[string]any{"project_id": "p1", "target_type": "task", "target_id": "t1", "body_markdown": "hello"}},
		{name: "koll.list_comments_by_target", args: map[string]any{"project_id": "p1", "target_type": "task", "target_id": "t1"}},
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

// TestHandlerExpandedToolForwardsActorTupleFields verifies actor tuple fields flow through update/restore tool requests.
func TestHandlerExpandedToolForwardsActorTupleFields(t *testing.T) {
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

	_, updateResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(301, "koll.update_task", map[string]any{
		"task_id":    "t1",
		"title":      "Task One Updated",
		"actor_type": "user",
		"agent_name": "EVAN",
	}))
	if isError, _ := updateResp.Result["isError"].(bool); isError {
		t.Fatalf("update_task returned isError=true: %#v", updateResp.Result)
	}
	if got := service.lastUpdateTaskReq.Actor.ActorType; got != "user" {
		t.Fatalf("update_task actor_type = %q, want user", got)
	}
	if got := service.lastUpdateTaskReq.Actor.AgentName; got != "EVAN" {
		t.Fatalf("update_task agent_name = %q, want EVAN", got)
	}

	_, restoreResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(302, "koll.restore_task", map[string]any{
		"task_id":           "t1",
		"actor_type":        "agent",
		"agent_name":        "agent-1",
		"agent_instance_id": "agent-1",
		"lease_token":       "lease-1",
		"override_token":    "override-1",
	}))
	if isError, _ := restoreResp.Result["isError"].(bool); isError {
		t.Fatalf("restore_task returned isError=true: %#v", restoreResp.Result)
	}
	if got := service.lastRestoreTaskReq.Actor.ActorType; got != "agent" {
		t.Fatalf("restore_task actor_type = %q, want agent", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentName; got != "agent-1" {
		t.Fatalf("restore_task agent_name = %q, want agent-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.AgentInstanceID; got != "agent-1" {
		t.Fatalf("restore_task agent_instance_id = %q, want agent-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.LeaseToken; got != "lease-1" {
		t.Fatalf("restore_task lease_token = %q, want lease-1", got)
	}
	if got := service.lastRestoreTaskReq.Actor.OverrideToken; got != "override-1" {
		t.Fatalf("restore_task override_token = %q, want override-1", got)
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

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(201, "koll.create_project", map[string]any{
		"name": 123,
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}
