// Package httpapi provides the REST HTTP adapter for the server surfaces.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hylla/hakoll/internal/adapters/server/common"
)

// maxRequestBodyBytes limits decoded JSON payload size for fail-closed request handling.
const maxRequestBodyBytes int64 = 1 << 20

// Handler serves the versioned API subrouter mounted under `/api/v1`.
type Handler struct {
	captureState common.CaptureStateReader
	attention    common.AttentionService
}

// APIError represents one structured API failure response.
type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Hint    string         `json:"hint,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

// ErrorEnvelope wraps one structured API error.
type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

// NewHandler constructs one HTTP API adapter from capture and optional attention services.
func NewHandler(captureState common.CaptureStateReader, attention common.AttentionService) *Handler {
	return &Handler{
		captureState: captureState,
		attention:    attention,
	}
}

// ServeHTTP routes one versioned API request to the matching handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := normalizePath(r.URL.Path)
	switch {
	case path == "capture_state":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.handleCaptureState(w, r)
		return
	case path == "attention/items":
		switch r.Method {
		case http.MethodGet:
			h.handleListAttentionItems(w, r)
		case http.MethodPost:
			h.handleRaiseAttentionItem(w, r)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	default:
		itemID, ok := resolveAttentionItemID(path)
		if !ok {
			writeJSONError(w, http.StatusNotFound, APIError{
				Code:    "not_found",
				Message: "endpoint not found",
			})
			return
		}
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.handleResolveAttentionItem(w, r, itemID)
	}
}

// handleCaptureState serves GET `/capture_state`.
func (h *Handler) handleCaptureState(w http.ResponseWriter, r *http.Request) {
	if h.captureState == nil {
		writeJSONError(w, http.StatusServiceUnavailable, APIError{
			Code:    "service_unavailable",
			Message: "capture_state service is not configured",
		})
		return
	}
	req := common.CaptureStateRequest{
		ProjectID: r.URL.Query().Get("project_id"),
		ScopeType: r.URL.Query().Get("scope_type"),
		ScopeID:   r.URL.Query().Get("scope_id"),
		View:      r.URL.Query().Get("view"),
	}
	captureState, err := h.captureState.CaptureState(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, captureState)
}

// handleListAttentionItems serves GET `/attention/items`.
func (h *Handler) handleListAttentionItems(w http.ResponseWriter, r *http.Request) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}
	req := common.ListAttentionItemsRequest{
		ProjectID: strings.TrimSpace(r.URL.Query().Get("project_id")),
		ScopeType: strings.TrimSpace(r.URL.Query().Get("scope_type")),
		ScopeID:   strings.TrimSpace(r.URL.Query().Get("scope_id")),
		State:     strings.TrimSpace(r.URL.Query().Get("state")),
	}
	if req.ProjectID == "" {
		writeJSONError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_request",
			Message: "project_id is required",
		})
		return
	}
	items, err := h.attention.ListAttentionItems(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

// handleRaiseAttentionItem serves POST `/attention/items`.
func (h *Handler) handleRaiseAttentionItem(w http.ResponseWriter, r *http.Request) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}

	var req common.RaiseAttentionItemRequest
	if err := decodeJSONBody(r.Context(), w, r, &req); err != nil {
		writeErrorFrom(w, err)
		return
	}
	item, err := h.attention.RaiseAttentionItem(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleResolveAttentionItem serves POST `/attention/items/{id}/resolve`.
func (h *Handler) handleResolveAttentionItem(w http.ResponseWriter, r *http.Request, itemID string) {
	if h.attention == nil {
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: "attention APIs are not available",
		})
		return
	}

	req := common.ResolveAttentionItemRequest{
		ID: itemID,
	}
	var payload common.ResolveAttentionItemRequest
	if err := decodeOptionalJSONBody(r.Context(), w, r, &payload); err != nil {
		writeErrorFrom(w, err)
		return
	}
	if trimmed := strings.TrimSpace(payload.ResolvedBy); trimmed != "" {
		req.ResolvedBy = trimmed
	}
	if trimmed := strings.TrimSpace(payload.Reason); trimmed != "" {
		req.Reason = trimmed
	}

	item, err := h.attention.ResolveAttentionItem(r.Context(), req)
	if err != nil {
		writeErrorFrom(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// resolveAttentionItemID parses `/attention/items/{id}/resolve` and returns `{id}`.
func resolveAttentionItemID(path string) (string, bool) {
	const (
		prefix = "attention/items/"
		suffix = "/resolve"
	)
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix))
	if id == "" || strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// normalizePath canonicalizes one request path for route matching.
func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "/")
	return path
}

// writeErrorFrom maps adapter errors into structured HTTP responses.
func writeErrorFrom(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeJSONError(w, http.StatusInternalServerError, APIError{
			Code:    "internal_error",
			Message: "unknown error",
		})
	case errors.Is(err, common.ErrBootstrapRequired):
		writeJSONError(w, http.StatusConflict, APIError{
			Code:    "bootstrap_required",
			Message: err.Error(),
			Hint:    "Create the first project before calling capture_state.",
		})
	case errors.Is(err, common.ErrGuardrailViolation):
		writeJSONError(w, http.StatusConflict, APIError{
			Code:    "guardrail_failed",
			Message: err.Error(),
		})
	case errors.Is(err, common.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: err.Error(),
		})
	case errors.Is(err, common.ErrInvalidCaptureStateRequest), errors.Is(err, common.ErrUnsupportedScope):
		writeJSONError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_request",
			Message: err.Error(),
		})
	case errors.Is(err, common.ErrAttentionUnavailable):
		writeJSONError(w, http.StatusNotImplemented, APIError{
			Code:    "not_implemented",
			Message: err.Error(),
		})
	default:
		writeJSONError(w, http.StatusInternalServerError, APIError{
			Code:    "internal_error",
			Message: err.Error(),
		})
	}
}

// writeMethodNotAllowed writes a structured 405 response with `Allow` headers.
func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeJSONError(w, http.StatusMethodNotAllowed, APIError{
		Code:    "method_not_allowed",
		Message: "method not allowed",
	})
}

// writeJSONError writes one structured error envelope.
func writeJSONError(w http.ResponseWriter, statusCode int, apiErr APIError) {
	writeJSON(w, statusCode, ErrorEnvelope{Error: apiErr})
}

// writeJSON writes one JSON response envelope.
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"code":"encode_error","message":"%s"}}`, err.Error()), http.StatusInternalServerError)
	}
}

// decodeJSONBody decodes one required JSON request body with strict shape checks.
func decodeJSONBody(ctx context.Context, w http.ResponseWriter, r *http.Request, out any) error {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode request body: %w", errors.Join(common.ErrInvalidCaptureStateRequest, err))
	}
	// Reject trailing payloads so malformed JSON bodies fail closed.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode request body: trailing content: %w", common.ErrInvalidCaptureStateRequest)
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("request canceled: %w", ctx.Err())
	default:
		return nil
	}
}

// decodeOptionalJSONBody decodes one optional JSON body and ignores empty payloads.
func decodeOptionalJSONBody(ctx context.Context, w http.ResponseWriter, r *http.Request, out any) error {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(out)
	if err == nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("request canceled: %w", ctx.Err())
		default:
			return nil
		}
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return fmt.Errorf("decode request body: %w", errors.Join(common.ErrInvalidCaptureStateRequest, err))
}
