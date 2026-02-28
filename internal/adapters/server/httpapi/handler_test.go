package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hylla/hakoll/internal/adapters/server/common"
)

// stubCaptureStateReader provides deterministic capture-state responses for handler tests.
type stubCaptureStateReader struct {
	captureState common.CaptureState
	err          error
	lastRequest  common.CaptureStateRequest
}

// CaptureState records the request and returns the configured response.
func (s *stubCaptureStateReader) CaptureState(_ context.Context, req common.CaptureStateRequest) (common.CaptureState, error) {
	s.lastRequest = req
	if s.err != nil {
		return common.CaptureState{}, s.err
	}
	return s.captureState, nil
}

// stubAttentionService provides deterministic attention responses for handler tests.
type stubAttentionService struct {
	items       []common.AttentionItem
	raised      common.AttentionItem
	resolved    common.AttentionItem
	err         error
	lastList    common.ListAttentionItemsRequest
	lastRaise   common.RaiseAttentionItemRequest
	lastResolve common.ResolveAttentionItemRequest
}

// ListAttentionItems returns deterministic fixture items.
func (s *stubAttentionService) ListAttentionItems(_ context.Context, req common.ListAttentionItemsRequest) ([]common.AttentionItem, error) {
	s.lastList = req
	if s.err != nil {
		return nil, s.err
	}
	return append([]common.AttentionItem(nil), s.items...), nil
}

// RaiseAttentionItem records and returns one fixture item.
func (s *stubAttentionService) RaiseAttentionItem(_ context.Context, req common.RaiseAttentionItemRequest) (common.AttentionItem, error) {
	s.lastRaise = req
	if s.err != nil {
		return common.AttentionItem{}, s.err
	}
	return s.raised, nil
}

// ResolveAttentionItem records and returns one fixture item.
func (s *stubAttentionService) ResolveAttentionItem(_ context.Context, req common.ResolveAttentionItemRequest) (common.AttentionItem, error) {
	s.lastResolve = req
	if s.err != nil {
		return common.AttentionItem{}, s.err
	}
	return s.resolved, nil
}

// decodeBody decodes one JSON response body into the requested type.
func decodeBody[T any](t *testing.T, body *strings.Reader) T {
	t.Helper()
	var out T
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return out
}

// TestHandlerCaptureStateSuccess verifies capture_state response mapping for valid requests.
func TestHandlerCaptureStateSuccess(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	capture := &stubCaptureStateReader{
		captureState: common.CaptureState{
			CapturedAt: now,
			StateHash:  "abc123",
			GoalOverview: common.GoalOverview{
				ProjectID:   "p1",
				ProjectName: "Roadmap",
			},
			WorkOverview: common.WorkOverview{TotalTasks: 3},
		},
	}
	handler := NewHandler(capture, nil)

	req := httptest.NewRequest(http.MethodGet, "/capture_state?project_id=p1&view=full", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got common.CaptureState
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.StateHash != "abc123" {
		t.Fatalf("state_hash = %q, want abc123", got.StateHash)
	}
	if capture.lastRequest.ProjectID != "p1" {
		t.Fatalf("project_id = %q, want p1", capture.lastRequest.ProjectID)
	}
	if capture.lastRequest.View != "full" {
		t.Fatalf("view = %q, want full", capture.lastRequest.View)
	}
}

// TestHandlerCaptureStateErrorMapping verifies structured status mapping for capture errors.
func TestHandlerCaptureStateErrorMapping(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid request",
			err:        errors.Join(common.ErrInvalidCaptureStateRequest, errors.New("bad input")),
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "not found",
			err:        errors.Join(common.ErrNotFound, errors.New("missing")),
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "internal error",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			capture := &stubCaptureStateReader{err: tt.err}
			handler := NewHandler(capture, nil)

			req := httptest.NewRequest(http.MethodGet, "/capture_state?project_id=p1", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var envelope ErrorEnvelope
			if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if envelope.Error.Code != tt.wantCode {
				t.Fatalf("error.code = %q, want %q", envelope.Error.Code, tt.wantCode)
			}
		})
	}
}

// TestHandlerAttentionUnavailable verifies fail-closed behavior when attention service is absent.
func TestHandlerAttentionUnavailable(t *testing.T) {
	handler := NewHandler(&stubCaptureStateReader{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/attention/items?project_id=p1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
	}
	var envelope ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if envelope.Error.Code != "not_implemented" {
		t.Fatalf("error.code = %q, want not_implemented", envelope.Error.Code)
	}
}

// TestHandlerAttentionEndpoints verifies list/raise/resolve wiring when attention service exists.
func TestHandlerAttentionEndpoints(t *testing.T) {
	now := time.Date(2026, 2, 24, 12, 0, 0, 0, time.UTC)
	attention := &stubAttentionService{
		items: []common.AttentionItem{
			{
				ID:                 "a1",
				ProjectID:          "p1",
				ScopeType:          common.ScopeTypeProject,
				ScopeID:            "p1",
				State:              common.AttentionStateOpen,
				Kind:               "blocker",
				Summary:            "Needs approval",
				RequiresUserAction: true,
				CreatedAt:          now,
			},
		},
		raised: common.AttentionItem{
			ID:        "a2",
			ProjectID: "p1",
			ScopeType: common.ScopeTypeProject,
			ScopeID:   "p1",
			State:     common.AttentionStateOpen,
			Kind:      "risk_note",
			Summary:   "Raised by API",
			CreatedAt: now,
		},
		resolved: common.AttentionItem{
			ID:        "a1",
			ProjectID: "p1",
			ScopeType: common.ScopeTypeProject,
			ScopeID:   "p1",
			State:     common.AttentionStateResolved,
			Kind:      "blocker",
			Summary:   "Needs approval",
			CreatedAt: now,
		},
	}
	handler := NewHandler(&stubCaptureStateReader{}, attention)

	// List
	listReq := httptest.NewRequest(http.MethodGet, "/attention/items?project_id=p1&scope_type=project&scope_id=p1", nil)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRec.Code, http.StatusOK)
	}
	var listed struct {
		Items []common.AttentionItem `json:"items"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listed); err != nil {
		t.Fatalf("Decode(list) error = %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != "a1" {
		t.Fatalf("unexpected list payload %#v", listed.Items)
	}

	// Raise
	raiseReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items",
		strings.NewReader(`{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk_note","summary":"Raised by API"}`),
	)
	raiseReq.Header.Set("Content-Type", "application/json")
	raiseRec := httptest.NewRecorder()
	handler.ServeHTTP(raiseRec, raiseReq)
	if raiseRec.Code != http.StatusCreated {
		t.Fatalf("raise status = %d, want %d", raiseRec.Code, http.StatusCreated)
	}
	var raised common.AttentionItem
	if err := json.NewDecoder(raiseRec.Body).Decode(&raised); err != nil {
		t.Fatalf("Decode(raise) error = %v", err)
	}
	if raised.ID != "a2" {
		t.Fatalf("raised id = %q, want a2", raised.ID)
	}

	// Resolve
	resolveReq := httptest.NewRequest(
		http.MethodPost,
		"/attention/items/a1/resolve",
		strings.NewReader(`{"resolved_by":"tester","reason":"approved"}`),
	)
	resolveReq.Header.Set("Content-Type", "application/json")
	resolveRec := httptest.NewRecorder()
	handler.ServeHTTP(resolveRec, resolveReq)
	if resolveRec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d", resolveRec.Code, http.StatusOK)
	}
	var resolved common.AttentionItem
	if err := json.NewDecoder(resolveRec.Body).Decode(&resolved); err != nil {
		t.Fatalf("Decode(resolve) error = %v", err)
	}
	if resolved.State != common.AttentionStateResolved {
		t.Fatalf("resolved state = %q, want %q", resolved.State, common.AttentionStateResolved)
	}
	if attention.lastResolve.ID != "a1" {
		t.Fatalf("resolve request id = %q, want a1", attention.lastResolve.ID)
	}
}

// decodeErrorEnvelope decodes one structured API error response from the recorder body.
func decodeErrorEnvelope(t *testing.T, rec *httptest.ResponseRecorder) ErrorEnvelope {
	t.Helper()
	var envelope ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return envelope
}

// TestHandlerRouteGuards verifies method guards and unknown-route handling.
func TestHandlerRouteGuards(t *testing.T) {
	handler := NewHandler(&stubCaptureStateReader{}, &stubAttentionService{})

	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantCode   string
		wantAllow  string
	}{
		{
			name:       "capture_state requires get",
			method:     http.MethodPost,
			path:       "/capture_state",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
			wantAllow:  http.MethodGet,
		},
		{
			name:       "attention list route only allows get and post",
			method:     http.MethodDelete,
			path:       "/attention/items",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
			wantAllow:  "GET, POST",
		},
		{
			name:       "attention resolve requires post",
			method:     http.MethodGet,
			path:       "/attention/items/a1/resolve",
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "method_not_allowed",
			wantAllow:  http.MethodPost,
		},
		{
			name:       "unknown route returns not found",
			method:     http.MethodGet,
			path:       "/not/a/route",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:       "invalid resolve path returns not found",
			method:     http.MethodPost,
			path:       "/attention/items/a1/nested/resolve",
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			envelope := decodeErrorEnvelope(t, rec)
			if envelope.Error.Code != tt.wantCode {
				t.Fatalf("error.code = %q, want %q", envelope.Error.Code, tt.wantCode)
			}
			if got := rec.Header().Get("Allow"); got != tt.wantAllow {
				t.Fatalf("Allow header = %q, want %q", got, tt.wantAllow)
			}
		})
	}
}

// TestHandlerCaptureStateServiceUnavailable verifies nil capture service maps to 503.
func TestHandlerCaptureStateServiceUnavailable(t *testing.T) {
	handler := NewHandler(nil, &stubAttentionService{})
	req := httptest.NewRequest(http.MethodGet, "/capture_state?project_id=p1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	envelope := decodeErrorEnvelope(t, rec)
	if envelope.Error.Code != "service_unavailable" {
		t.Fatalf("error.code = %q, want service_unavailable", envelope.Error.Code)
	}
}

// TestHandlerAttentionEndpointsUnavailable verifies nil attention service stays fail-closed.
func TestHandlerAttentionEndpointsUnavailable(t *testing.T) {
	handler := NewHandler(&stubCaptureStateReader{}, nil)

	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "raise endpoint unavailable",
			path: "/attention/items",
			body: `{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk_note","summary":"x"}`,
		},
		{
			name: "resolve endpoint unavailable",
			path: "/attention/items/a1/resolve",
			body: `{"resolved_by":"tester"}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotImplemented)
			}
			envelope := decodeErrorEnvelope(t, rec)
			if envelope.Error.Code != "not_implemented" {
				t.Fatalf("error.code = %q, want not_implemented", envelope.Error.Code)
			}
		})
	}
}

// TestHandlerAttentionJSONValidation verifies malformed attention payloads return invalid_request.
func TestHandlerAttentionJSONValidation(t *testing.T) {
	attention := &stubAttentionService{}
	handler := NewHandler(&stubCaptureStateReader{}, attention)

	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "raise endpoint malformed json",
			path: "/attention/items",
			body: `{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk","summary":"x"`,
		},
		{
			name: "raise endpoint unknown field",
			path: "/attention/items",
			body: `{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk","summary":"x","unknown":"field"}`,
		},
		{
			name: "raise endpoint trailing payload",
			path: "/attention/items",
			body: `{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk","summary":"x"}{"extra":true}`,
		},
		{
			name: "resolve endpoint malformed json",
			path: "/attention/items/a1/resolve",
			body: `{"resolved_by":"user"`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
			envelope := decodeErrorEnvelope(t, rec)
			if envelope.Error.Code != "invalid_request" {
				t.Fatalf("error.code = %q, want invalid_request", envelope.Error.Code)
			}
		})
	}
}

// TestHandlerRaiseAttentionScopeValidationErrorMapping verifies scope validation errors map to invalid_request responses.
func TestHandlerRaiseAttentionScopeValidationErrorMapping(t *testing.T) {
	attention := &stubAttentionService{
		err: errors.Join(common.ErrUnsupportedScope, errors.New("scope_type is required")),
	}
	handler := NewHandler(&stubCaptureStateReader{}, attention)
	req := httptest.NewRequest(
		http.MethodPost,
		"/attention/items",
		strings.NewReader(`{"project_id":"p1","kind":"risk_note","summary":"x"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	envelope := decodeErrorEnvelope(t, rec)
	if envelope.Error.Code != "invalid_request" {
		t.Fatalf("error.code = %q, want invalid_request", envelope.Error.Code)
	}
}

// TestHandlerAttentionListRequiresProjectID verifies list rejects missing project_id.
func TestHandlerAttentionListRequiresProjectID(t *testing.T) {
	handler := NewHandler(&stubCaptureStateReader{}, &stubAttentionService{})
	req := httptest.NewRequest(http.MethodGet, "/attention/items?scope_type=project", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	envelope := decodeErrorEnvelope(t, rec)
	if envelope.Error.Code != "invalid_request" {
		t.Fatalf("error.code = %q, want invalid_request", envelope.Error.Code)
	}
}

// TestHandlerResolveAttentionItemEmptyBody verifies empty resolve payloads are accepted.
func TestHandlerResolveAttentionItemEmptyBody(t *testing.T) {
	attention := &stubAttentionService{
		resolved: common.AttentionItem{
			ID:    "a1",
			State: common.AttentionStateResolved,
		},
	}
	handler := NewHandler(&stubCaptureStateReader{}, attention)
	req := httptest.NewRequest(http.MethodPost, "/attention/items/a1/resolve", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if attention.lastResolve.ID != "a1" {
		t.Fatalf("resolve request id = %q, want a1", attention.lastResolve.ID)
	}
	if attention.lastResolve.ResolvedBy != "" {
		t.Fatalf("resolved_by = %q, want empty", attention.lastResolve.ResolvedBy)
	}
	if attention.lastResolve.Reason != "" {
		t.Fatalf("reason = %q, want empty", attention.lastResolve.Reason)
	}
}

// TestDecodeJSONBodyBranches verifies decodeJSONBody trailing payload and canceled-context branches.
func TestDecodeJSONBodyBranches(t *testing.T) {
	w := httptest.NewRecorder()

	t.Run("trailing payload returns invalid capture request", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/attention/items",
			strings.NewReader(`{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk","summary":"x"}{"next":true}`),
		)
		var payload common.RaiseAttentionItemRequest
		err := decodeJSONBody(context.Background(), w, req, &payload)
		if err == nil {
			t.Fatalf("decodeJSONBody() error = nil, want non-nil")
		}
		if !errors.Is(err, common.ErrInvalidCaptureStateRequest) {
			t.Fatalf("decodeJSONBody() error = %v, want ErrInvalidCaptureStateRequest", err)
		}
	})

	t.Run("canceled context returns context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(
			http.MethodPost,
			"/attention/items",
			strings.NewReader(`{"project_id":"p1","scope_type":"project","scope_id":"p1","kind":"risk","summary":"x"}`),
		).WithContext(ctx)
		var payload common.RaiseAttentionItemRequest
		err := decodeJSONBody(req.Context(), w, req, &payload)
		if err == nil {
			t.Fatalf("decodeJSONBody() error = nil, want non-nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("decodeJSONBody() error = %v, want context.Canceled", err)
		}
	})
}

// TestDecodeOptionalJSONBodyBranches verifies optional decode behavior across branch outcomes.
func TestDecodeOptionalJSONBodyBranches(t *testing.T) {
	w := httptest.NewRecorder()

	t.Run("empty body is accepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/attention/items/a1/resolve", strings.NewReader(""))
		var payload common.ResolveAttentionItemRequest
		if err := decodeOptionalJSONBody(context.Background(), w, req, &payload); err != nil {
			t.Fatalf("decodeOptionalJSONBody() error = %v", err)
		}
	})

	t.Run("malformed body maps to invalid capture request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/attention/items/a1/resolve", strings.NewReader(`{"resolved_by":"u"`))
		var payload common.ResolveAttentionItemRequest
		err := decodeOptionalJSONBody(context.Background(), w, req, &payload)
		if err == nil {
			t.Fatalf("decodeOptionalJSONBody() error = nil, want non-nil")
		}
		if !errors.Is(err, common.ErrInvalidCaptureStateRequest) {
			t.Fatalf("decodeOptionalJSONBody() error = %v, want ErrInvalidCaptureStateRequest", err)
		}
	})

	t.Run("canceled context returns context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(http.MethodPost, "/attention/items/a1/resolve", strings.NewReader(`{"resolved_by":"u"}`)).WithContext(ctx)
		var payload common.ResolveAttentionItemRequest
		err := decodeOptionalJSONBody(req.Context(), w, req, &payload)
		if err == nil {
			t.Fatalf("decodeOptionalJSONBody() error = nil, want non-nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("decodeOptionalJSONBody() error = %v, want context.Canceled", err)
		}
	})
}

// TestWriteErrorFromMappingBranches verifies explicit error mapping for uncovered branches.
func TestWriteErrorFromMappingBranches(t *testing.T) {
	cases := []struct {
		name          string
		err           error
		wantStatus    int
		wantCode      string
		wantMsgSubstr string
	}{
		{
			name:          "nil error becomes unknown internal error",
			err:           nil,
			wantStatus:    http.StatusInternalServerError,
			wantCode:      "internal_error",
			wantMsgSubstr: "unknown error",
		},
		{
			name:          "unsupported scope is invalid request",
			err:           errors.Join(common.ErrUnsupportedScope, errors.New("scope mismatch")),
			wantStatus:    http.StatusBadRequest,
			wantCode:      "invalid_request",
			wantMsgSubstr: "scope mismatch",
		},
		{
			name:          "attention unavailable is not implemented",
			err:           errors.Join(common.ErrAttentionUnavailable, errors.New("feature disabled")),
			wantStatus:    http.StatusNotImplemented,
			wantCode:      "not_implemented",
			wantMsgSubstr: "feature disabled",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeErrorFrom(rec, tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			envelope := decodeErrorEnvelope(t, rec)
			if envelope.Error.Code != tt.wantCode {
				t.Fatalf("error.code = %q, want %q", envelope.Error.Code, tt.wantCode)
			}
			if !strings.Contains(envelope.Error.Message, tt.wantMsgSubstr) {
				t.Fatalf("error.message = %q, want substring %q", envelope.Error.Message, tt.wantMsgSubstr)
			}
		})
	}
}

// TestResolveAttentionItemID verifies attention resolve-path parsing behavior.
func TestResolveAttentionItemID(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		wantID string
		wantOK bool
	}{
		{
			name:   "valid resolve path",
			path:   "attention/items/a1/resolve",
			wantID: "a1",
			wantOK: true,
		},
		{
			name:   "missing id is invalid",
			path:   "attention/items//resolve",
			wantOK: false,
		},
		{
			name:   "nested segment is invalid",
			path:   "attention/items/a1/child/resolve",
			wantOK: false,
		},
		{
			name:   "wrong suffix is invalid",
			path:   "attention/items/a1/delete",
			wantOK: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := resolveAttentionItemID(tt.path)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotID != tt.wantID {
				t.Fatalf("id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

// TestNormalizePath verifies deterministic path normalization.
func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "/capture_state/", want: "capture_state"},
		{in: "  /attention/items/a1/resolve  ", want: "attention/items/a1/resolve"},
		{in: "///", want: ""},
		{in: "", want: ""},
	}

	for _, tt := range cases {
		if got := normalizePath(tt.in); got != tt.want {
			t.Fatalf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
