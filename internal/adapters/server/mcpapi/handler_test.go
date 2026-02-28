package mcpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hylla/hakoll/internal/adapters/server/common"
	"github.com/hylla/hakoll/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// stubCaptureStateReader provides deterministic capture-state responses for MCP tool tests.
type stubCaptureStateReader struct {
	captureState common.CaptureState
	err          error
	lastRequest  common.CaptureStateRequest
}

// CaptureState records the latest request and returns one fixture result.
func (s *stubCaptureStateReader) CaptureState(_ context.Context, req common.CaptureStateRequest) (common.CaptureState, error) {
	s.lastRequest = req
	if s.err != nil {
		return common.CaptureState{}, s.err
	}
	return s.captureState, nil
}

// stubAttentionService provides deterministic attention responses for MCP tool tests.
type stubAttentionService struct {
	items       []common.AttentionItem
	raised      common.AttentionItem
	resolved    common.AttentionItem
	listErr     error
	raiseErr    error
	resolveErr  error
	lastList    common.ListAttentionItemsRequest
	lastRaise   common.RaiseAttentionItemRequest
	lastResolve common.ResolveAttentionItemRequest
}

// stubProjectService provides deterministic project responses for expanded MCP tool registration tests.
type stubProjectService struct {
	stubCaptureStateReader
	projects            []domain.Project
	createResult        domain.Project
	updateResult        domain.Project
	listErr             error
	createErr           error
	updateErr           error
	lastIncludeArchived bool
	lastCreate          common.CreateProjectRequest
	lastUpdate          common.UpdateProjectRequest
}

// ListProjects returns deterministic project list rows.
func (s *stubProjectService) ListProjects(_ context.Context, includeArchived bool) ([]domain.Project, error) {
	s.lastIncludeArchived = includeArchived
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]domain.Project(nil), s.projects...), nil
}

// CreateProject records and returns deterministic project creation results.
func (s *stubProjectService) CreateProject(_ context.Context, req common.CreateProjectRequest) (domain.Project, error) {
	s.lastCreate = req
	if s.createErr != nil {
		return domain.Project{}, s.createErr
	}
	return s.createResult, nil
}

// UpdateProject records and returns deterministic project update results.
func (s *stubProjectService) UpdateProject(_ context.Context, req common.UpdateProjectRequest) (domain.Project, error) {
	s.lastUpdate = req
	if s.updateErr != nil {
		return domain.Project{}, s.updateErr
	}
	return s.updateResult, nil
}

// ListAttentionItems returns deterministic list data.
func (s *stubAttentionService) ListAttentionItems(_ context.Context, req common.ListAttentionItemsRequest) ([]common.AttentionItem, error) {
	s.lastList = req
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]common.AttentionItem(nil), s.items...), nil
}

// RaiseAttentionItem records and returns one fixture item.
func (s *stubAttentionService) RaiseAttentionItem(_ context.Context, req common.RaiseAttentionItemRequest) (common.AttentionItem, error) {
	s.lastRaise = req
	if s.raiseErr != nil {
		return common.AttentionItem{}, s.raiseErr
	}
	return s.raised, nil
}

// ResolveAttentionItem records and returns one fixture item.
func (s *stubAttentionService) ResolveAttentionItem(_ context.Context, req common.ResolveAttentionItemRequest) (common.AttentionItem, error) {
	s.lastResolve = req
	if s.resolveErr != nil {
		return common.AttentionItem{}, s.resolveErr
	}
	return s.resolved, nil
}

// jsonRPCResponse models minimal JSON-RPC response fields used in MCP adapter tests.
type jsonRPCResponse struct {
	ID     float64        `json:"id"`
	Result map[string]any `json:"result"`
}

// callToolRequest constructs one deterministic tools/call JSON-RPC request payload.
func callToolRequest(id int, toolName string, arguments map[string]any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": arguments,
		},
	}
}

// toolResultText decodes the first text entry from one tool-call result payload.
func toolResultText(t *testing.T, result map[string]any) string {
	t.Helper()

	contentRaw, ok := result["content"].([]any)
	if !ok || len(contentRaw) == 0 {
		t.Fatalf("content missing in tool result: %#v", result)
	}
	first, ok := contentRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("first content entry has unexpected type: %#v", contentRaw[0])
	}
	text, ok := first["text"].(string)
	if !ok {
		t.Fatalf("content text missing in tool result: %#v", first)
	}
	return text
}

// toolResultStructured decodes structuredContent as one map for stable assertions.
func toolResultStructured(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing in tool result: %#v", result)
	}
	return structured
}

// postJSONRPC sends one JSON-RPC payload and decodes the response body.
func postJSONRPC(t *testing.T, client *http.Client, url string, payload any) (*http.Response, jsonRPCResponse) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	var decoded jsonRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return resp, decoded
}

// initializeRequest builds a deterministic MCP initialize request payload.
func initializeRequest() map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcp.LATEST_PROTOCOL_VERSION,
			"clientInfo": map[string]any{
				"name":    "hakoll-test",
				"version": "1.0.0",
			},
		},
	}
}

// callToolResultText decodes the first textual content block from a CallToolResult.
func callToolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatalf("result = nil, want non-nil")
	}
	if len(result.Content) == 0 {
		t.Fatalf("result content is empty")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] has unexpected type %T", result.Content[0])
	}
	return text.Text
}

// TestHandlerUsesStatelessTransport verifies MCP transport does not issue session ids.
func TestHandlerUsesStatelessTransport(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	resp, decoded := postJSONRPC(t, server.Client(), server.URL, initializeRequest())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if decoded.ID != 1 {
		t.Fatalf("id = %v, want 1", decoded.ID)
	}
	if got := resp.Header.Get("Mcp-Session-Id"); got != "" {
		t.Fatalf("Mcp-Session-Id header = %q, want empty (stateless transport)", got)
	}
}

// TestHandlerRegistersCaptureStateTool verifies MCP tool discovery includes koll.capture_state.
func TestHandlerRegistersCaptureStateTool(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
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
	if !slices.Contains(toolNames, "koll.capture_state") {
		t.Fatalf("tool list missing koll.capture_state: %#v", toolNames)
	}
	if slices.Contains(toolNames, "koll.list_attention_items") {
		t.Fatalf("unexpected attention tool without attention service: %#v", toolNames)
	}
}

// TestHandlerRegistersAttentionToolsWhenAvailable verifies optional attention tools are exposed.
func TestHandlerRegistersAttentionToolsWhenAvailable(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			StateHash: "abc123",
		},
	}
	attention := &stubAttentionService{}
	handler, err := NewHandler(Config{}, capture, attention)
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
	for _, required := range []string{
		"koll.capture_state",
		"koll.list_attention_items",
		"koll.raise_attention_item",
		"koll.resolve_attention_item",
	} {
		if !slices.Contains(toolNames, required) {
			t.Fatalf("tool list missing %q: %#v", required, toolNames)
		}
	}
}

// TestHandlerRegistersProjectToolsWhenAvailable verifies expanded project tools register when the capture adapter exposes project APIs.
func TestHandlerRegistersProjectToolsWhenAvailable(t *testing.T) {
	capture := &stubProjectService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
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
	for _, required := range []string{
		"koll.capture_state",
		"koll.list_projects",
		"koll.create_project",
		"koll.update_project",
	} {
		if !slices.Contains(toolNames, required) {
			t.Fatalf("tool list missing %q: %#v", required, toolNames)
		}
	}
}

// TestHandlerProjectToolCall verifies expanded project tool wiring returns structured project rows.
func TestHandlerProjectToolCall(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubProjectService{
		stubCaptureStateReader: stubCaptureStateReader{
			captureState: common.CaptureState{StateHash: "abc123"},
		},
		projects: []domain.Project{
			{
				ID:        "p1",
				Slug:      "roadmap",
				Name:      "Roadmap",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "koll.list_projects", map[string]any{
		"include_archived": true,
	}))
	structured := toolResultStructured(t, callResp.Result)
	projectsRaw, ok := structured["projects"].([]any)
	if !ok || len(projectsRaw) != 1 {
		t.Fatalf("projects = %#v, want one row", structured["projects"])
	}
	if !capture.lastIncludeArchived {
		t.Fatalf("include_archived = false, want true")
	}
}

// TestHandlerCaptureStateToolCall verifies tool-call wiring returns structured capture data.
func TestHandlerCaptureStateToolCall(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			CapturedAt: now,
			StateHash:  "abc123",
			GoalOverview: common.GoalOverview{
				ProjectID:   "p1",
				ProjectName: "Roadmap",
			},
		},
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "koll.capture_state",
			"arguments": map[string]any{
				"project_id": "p1",
				"view":       "full",
			},
		},
	})
	result, ok := callResp.Result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing in response: %#v", callResp.Result)
	}
	if got, _ := result["state_hash"].(string); got != "abc123" {
		t.Fatalf("state_hash = %q, want abc123", got)
	}
	if capture.lastRequest.ProjectID != "p1" {
		t.Fatalf("project_id = %q, want p1", capture.lastRequest.ProjectID)
	}
	if capture.lastRequest.View != "full" {
		t.Fatalf("view = %q, want full", capture.lastRequest.View)
	}
}

// TestNewHandlerRequiresCaptureState verifies capture_state dependency enforcement.
func TestNewHandlerRequiresCaptureState(t *testing.T) {
	handler, err := NewHandler(Config{}, nil, nil)
	if err == nil {
		t.Fatalf("NewHandler() error = nil, want non-nil")
	}
	if handler != nil {
		t.Fatalf("handler = %#v, want nil", handler)
	}
}

// TestNormalizeConfig verifies deterministic config defaults and path normalization.
func TestNormalizeConfig(t *testing.T) {
	cases := []struct {
		name string
		in   Config
		want Config
	}{
		{
			name: "defaults",
			in:   Config{},
			want: Config{
				ServerName:    "hakoll",
				ServerVersion: "dev",
				EndpointPath:  "/mcp",
			},
		},
		{
			name: "trimmed values and slash prefix",
			in: Config{
				ServerName:    " hakoll-server ",
				ServerVersion: " v1.2.3 ",
				EndpointPath:  "custom/path",
			},
			want: Config{
				ServerName:    "hakoll-server",
				ServerVersion: "v1.2.3",
				EndpointPath:  "/custom/path",
			},
		},
		{
			name: "endpoint trim of repeated slashes",
			in: Config{
				ServerName:    "hakoll",
				ServerVersion: "dev",
				EndpointPath:  "///mcp///",
			},
			want: Config{
				ServerName:    "hakoll",
				ServerVersion: "dev",
				EndpointPath:  "/mcp",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeConfig(tt.in)
			if got.ServerName != tt.want.ServerName {
				t.Fatalf("ServerName = %q, want %q", got.ServerName, tt.want.ServerName)
			}
			if got.ServerVersion != tt.want.ServerVersion {
				t.Fatalf("ServerVersion = %q, want %q", got.ServerVersion, tt.want.ServerVersion)
			}
			if got.EndpointPath != tt.want.EndpointPath {
				t.Fatalf("EndpointPath = %q, want %q", got.EndpointPath, tt.want.EndpointPath)
			}
		})
	}
}

// TestHandlerServeHTTPUnavailable verifies nil handler paths fail closed with 503.
func TestHandlerServeHTTPUnavailable(t *testing.T) {
	cases := []struct {
		name    string
		handler *Handler
	}{
		{
			name:    "nil receiver",
			handler: nil,
		},
		{
			name:    "missing inner http handler",
			handler: &Handler{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{}`))
			rec := httptest.NewRecorder()

			tt.handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
			}
			if !strings.Contains(rec.Body.String(), "mcp handler unavailable") {
				t.Fatalf("body = %q, want mcp handler unavailable", rec.Body.String())
			}
		})
	}
}

// TestToolResultFromErrorMapping verifies deterministic error-to-tool-result mapping.
func TestToolResultFromErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantPrefix string
	}{
		{
			name:       "nil error",
			err:        nil,
			wantPrefix: "unknown error",
		},
		{
			name:       "bootstrap required",
			err:        errors.Join(common.ErrBootstrapRequired, errors.New("no projects")),
			wantPrefix: "bootstrap_required:",
		},
		{
			name:       "guardrail violation",
			err:        errors.Join(common.ErrGuardrailViolation, errors.New("lease mismatch")),
			wantPrefix: "guardrail_failed:",
		},
		{
			name:       "invalid capture request",
			err:        errors.Join(common.ErrInvalidCaptureStateRequest, errors.New("bad request")),
			wantPrefix: "invalid_request:",
		},
		{
			name:       "unsupported scope",
			err:        errors.Join(common.ErrUnsupportedScope, errors.New("scope mismatch")),
			wantPrefix: "invalid_request:",
		},
		{
			name:       "not found",
			err:        errors.Join(common.ErrNotFound, errors.New("missing")),
			wantPrefix: "not_found:",
		},
		{
			name:       "attention unavailable",
			err:        errors.Join(common.ErrAttentionUnavailable, errors.New("disabled")),
			wantPrefix: "not_implemented:",
		},
		{
			name:       "internal",
			err:        errors.New("boom"),
			wantPrefix: "internal_error:",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result := toolResultFromError(tt.err)
			if !result.IsError {
				t.Fatalf("IsError = false, want true")
			}
			if got := callToolResultText(t, result); !strings.HasPrefix(got, tt.wantPrefix) {
				t.Fatalf("text = %q, want prefix %q", got, tt.wantPrefix)
			}
		})
	}
}

// TestHandlerCaptureStateToolCallErrorPaths verifies required-arg and mapped-service errors.
func TestHandlerCaptureStateToolCallErrorPaths(t *testing.T) {
	capture := &stubCaptureStateReader{
		err: errors.Join(common.ErrUnsupportedScope, errors.New("scope mismatch")),
	}
	handler, err := NewHandler(Config{}, capture, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, missingArgResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "koll.capture_state", map[string]any{}))
	if isError, _ := missingArgResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", missingArgResp.Result["isError"])
	}
	if got := toolResultText(t, missingArgResp.Result); !strings.Contains(got, `required argument "project_id" not found`) {
		t.Fatalf("error text = %q, want required project_id message", got)
	}

	_, mappedErrResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "koll.capture_state", map[string]any{
		"project_id": "p1",
	}))
	if isError, _ := mappedErrResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", mappedErrResp.Result["isError"])
	}
	if got := toolResultText(t, mappedErrResp.Result); !strings.HasPrefix(got, "invalid_request:") {
		t.Fatalf("error text = %q, want prefix invalid_request:", got)
	}
}

// TestHandlerAttentionToolCalls verifies optional attention tools execute and map request arguments.
func TestHandlerAttentionToolCalls(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{StateHash: "abc123"},
	}
	attention := &stubAttentionService{
		items: []common.AttentionItem{
			{
				ID:        "a1",
				ProjectID: "p1",
				ScopeType: common.ScopeTypeProject,
				ScopeID:   "p1",
				State:     common.AttentionStateOpen,
				Kind:      "risk_note",
				Summary:   "Need user action",
				CreatedAt: now,
			},
		},
		raised: common.AttentionItem{
			ID:                 "a2",
			ProjectID:          "p1",
			ScopeType:          common.ScopeTypeProject,
			ScopeID:            "p1",
			State:              common.AttentionStateOpen,
			Kind:               "blocker",
			Summary:            "Raised by tool",
			BodyMarkdown:       "Details",
			RequiresUserAction: true,
			CreatedAt:          now,
		},
		resolved: common.AttentionItem{
			ID:        "a1",
			ProjectID: "p1",
			ScopeType: common.ScopeTypeProject,
			ScopeID:   "p1",
			State:     common.AttentionStateResolved,
			Kind:      "risk_note",
			Summary:   "Need user action",
			CreatedAt: now,
		},
	}

	handler, err := NewHandler(Config{}, capture, attention)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, listResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "koll.list_attention_items", map[string]any{
		"project_id": "p1",
		"scope_type": "project",
		"scope_id":   "p1",
		"state":      "open",
	}))
	listStructured := toolResultStructured(t, listResp.Result)
	itemsRaw, ok := listStructured["items"].([]any)
	if !ok || len(itemsRaw) != 1 {
		t.Fatalf("list structured items = %#v, want one item", listStructured["items"])
	}
	if attention.lastList.ProjectID != "p1" {
		t.Fatalf("list project_id = %q, want p1", attention.lastList.ProjectID)
	}
	if attention.lastList.ScopeType != "project" {
		t.Fatalf("list scope_type = %q, want project", attention.lastList.ScopeType)
	}
	if attention.lastList.ScopeID != "p1" {
		t.Fatalf("list scope_id = %q, want p1", attention.lastList.ScopeID)
	}
	if attention.lastList.State != "open" {
		t.Fatalf("list state = %q, want open", attention.lastList.State)
	}

	_, raiseResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(3, "koll.raise_attention_item", map[string]any{
		"project_id":           "p1",
		"scope_type":           "project",
		"scope_id":             "p1",
		"kind":                 "blocker",
		"summary":              "Raised by tool",
		"body_markdown":        "Details",
		"requires_user_action": true,
	}))
	raiseStructured := toolResultStructured(t, raiseResp.Result)
	if got, _ := raiseStructured["id"].(string); got != "a2" {
		t.Fatalf("raised id = %q, want a2", got)
	}
	if attention.lastRaise.ProjectID != "p1" {
		t.Fatalf("raise project_id = %q, want p1", attention.lastRaise.ProjectID)
	}
	if attention.lastRaise.ScopeType != "project" {
		t.Fatalf("raise scope_type = %q, want project", attention.lastRaise.ScopeType)
	}
	if attention.lastRaise.ScopeID != "p1" {
		t.Fatalf("raise scope_id = %q, want p1", attention.lastRaise.ScopeID)
	}
	if attention.lastRaise.Kind != "blocker" {
		t.Fatalf("raise kind = %q, want blocker", attention.lastRaise.Kind)
	}
	if attention.lastRaise.Summary != "Raised by tool" {
		t.Fatalf("raise summary = %q, want Raised by tool", attention.lastRaise.Summary)
	}
	if attention.lastRaise.BodyMarkdown != "Details" {
		t.Fatalf("raise body_markdown = %q, want Details", attention.lastRaise.BodyMarkdown)
	}
	if !attention.lastRaise.RequiresUserAction {
		t.Fatalf("raise requires_user_action = false, want true")
	}

	_, resolveResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(4, "koll.resolve_attention_item", map[string]any{
		"id":          "a1",
		"resolved_by": "tester",
		"reason":      "approved",
	}))
	resolveStructured := toolResultStructured(t, resolveResp.Result)
	if got, _ := resolveStructured["state"].(string); got != common.AttentionStateResolved {
		t.Fatalf("resolved state = %q, want %q", got, common.AttentionStateResolved)
	}
	if attention.lastResolve.ID != "a1" {
		t.Fatalf("resolve id = %q, want a1", attention.lastResolve.ID)
	}
	if attention.lastResolve.ResolvedBy != "tester" {
		t.Fatalf("resolve resolved_by = %q, want tester", attention.lastResolve.ResolvedBy)
	}
	if attention.lastResolve.Reason != "approved" {
		t.Fatalf("resolve reason = %q, want approved", attention.lastResolve.Reason)
	}
}

// TestHandlerAttentionToolCallErrorMapping verifies attention tool errors surface as tool-result errors.
func TestHandlerAttentionToolCallErrorMapping(t *testing.T) {
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{StateHash: "abc123"},
	}
	attention := &stubAttentionService{
		listErr: errors.Join(common.ErrNotFound, errors.New("attention missing")),
	}

	handler, err := NewHandler(Config{}, capture, attention)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := httptest.NewServer(handler)
	defer server.Close()
	_, _ = postJSONRPC(t, server.Client(), server.URL, initializeRequest())

	_, callResp := postJSONRPC(t, server.Client(), server.URL, callToolRequest(2, "koll.list_attention_items", map[string]any{
		"project_id": "p1",
	}))
	if isError, _ := callResp.Result["isError"].(bool); !isError {
		t.Fatalf("isError = %v, want true", callResp.Result["isError"])
	}
	if got := toolResultText(t, callResp.Result); !strings.HasPrefix(got, "not_found:") {
		t.Fatalf("error text = %q, want prefix not_found:", got)
	}
}
