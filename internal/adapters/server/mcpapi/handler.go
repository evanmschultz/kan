// Package mcpapi provides a stateless MCP streamable-HTTP adapter.
package mcpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evanschultz/kan/internal/adapters/server/common"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Config captures MCP transport configuration.
type Config struct {
	ServerName    string
	ServerVersion string
	EndpointPath  string
}

// Handler wraps one stateless MCP streamable HTTP handler.
type Handler struct {
	httpHandler http.Handler
}

// NewHandler builds one stateless MCP adapter with capture_state and optional attention tools.
func NewHandler(cfg Config, captureState common.CaptureStateReader, attention common.AttentionService) (*Handler, error) {
	if captureState == nil {
		return nil, fmt.Errorf("capture_state service is required")
	}
	cfg = normalizeConfig(cfg)

	mcpSrv := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)
	registerCaptureStateTool(mcpSrv, captureState)
	if attention != nil {
		registerAttentionTools(mcpSrv, attention)
	}

	streamable := mcpserver.NewStreamableHTTPServer(
		mcpSrv,
		mcpserver.WithEndpointPath(cfg.EndpointPath),
		mcpserver.WithStateLess(true),
	)
	return &Handler{httpHandler: streamable}, nil
}

// ServeHTTP handles one MCP streamable HTTP request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.httpHandler == nil {
		http.Error(w, "mcp handler unavailable", http.StatusServiceUnavailable)
		return
	}
	h.httpHandler.ServeHTTP(w, r)
}

// normalizeConfig applies deterministic defaults to MCP adapter config.
func normalizeConfig(cfg Config) Config {
	cfg.ServerName = strings.TrimSpace(cfg.ServerName)
	if cfg.ServerName == "" {
		cfg.ServerName = "kan"
	}
	cfg.ServerVersion = strings.TrimSpace(cfg.ServerVersion)
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "dev"
	}
	cfg.EndpointPath = strings.TrimSpace(cfg.EndpointPath)
	if cfg.EndpointPath == "" {
		cfg.EndpointPath = "/mcp"
	}
	if !strings.HasPrefix(cfg.EndpointPath, "/") {
		cfg.EndpointPath = "/" + cfg.EndpointPath
	}
	cfg.EndpointPath = "/" + strings.Trim(cfg.EndpointPath, "/")
	return cfg
}

// registerCaptureStateTool registers the `kan.capture_state` tool.
func registerCaptureStateTool(srv *mcpserver.MCPServer, captureState common.CaptureStateReader) {
	srv.AddTool(
		mcp.NewTool(
			"kan.capture_state",
			mcp.WithDescription("Return a summary-first state capture for one scoped level tuple."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Description("Scope type"), mcp.Enum(common.SupportedScopeTypes()...)),
			mcp.WithString("scope_id", mcp.Description("Scope identifier (defaults to project_id)")),
			mcp.WithString("view", mcp.Description("summary or full"), mcp.Enum("summary", "full")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			capture, err := captureState.CaptureState(ctx, common.CaptureStateRequest{
				ProjectID: projectID,
				ScopeType: req.GetString("scope_type", ""),
				ScopeID:   req.GetString("scope_id", ""),
				View:      req.GetString("view", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(capture)
			if err != nil {
				return nil, fmt.Errorf("encode capture_state result: %w", err)
			}
			return result, nil
		},
	)
}

// registerAttentionTools registers optional attention list/raise/resolve tools.
func registerAttentionTools(srv *mcpserver.MCPServer, attention common.AttentionService) {
	srv.AddTool(
		mcp.NewTool(
			"kan.list_attention_items",
			mcp.WithDescription("List attention items for a project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Description("Scope type")),
			mcp.WithString("scope_id", mcp.Description("Scope identifier")),
			mcp.WithString("state", mcp.Description("Filter by state")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			items, err := attention.ListAttentionItems(ctx, common.ListAttentionItemsRequest{
				ProjectID: projectID,
				ScopeType: req.GetString("scope_type", ""),
				ScopeID:   req.GetString("scope_id", ""),
				State:     req.GetString("state", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(map[string]any{
				"items": items,
			})
			if err != nil {
				return nil, fmt.Errorf("encode list_attention_items result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"kan.raise_attention_item",
			mcp.WithDescription("Create a new attention item for a project scope."),
			mcp.WithString("project_id", mcp.Required(), mcp.Description("Project identifier")),
			mcp.WithString("scope_type", mcp.Required(), mcp.Description("Scope type")),
			mcp.WithString("scope_id", mcp.Required(), mcp.Description("Scope identifier")),
			mcp.WithString("kind", mcp.Required(), mcp.Description("Attention kind")),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Short summary")),
			mcp.WithString("body_markdown", mcp.Description("Optional markdown body")),
			mcp.WithBoolean("requires_user_action", mcp.Description("Whether this item blocks on user action")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectID, err := req.RequireString("project_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			scopeType, err := req.RequireString("scope_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			scopeID, err := req.RequireString("scope_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			kind, err := req.RequireString("kind")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			summary, err := req.RequireString("summary")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			item, err := attention.RaiseAttentionItem(ctx, common.RaiseAttentionItemRequest{
				ProjectID:          projectID,
				ScopeType:          scopeType,
				ScopeID:            scopeID,
				Kind:               kind,
				Summary:            summary,
				BodyMarkdown:       req.GetString("body_markdown", ""),
				RequiresUserAction: req.GetBool("requires_user_action", false),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(item)
			if err != nil {
				return nil, fmt.Errorf("encode raise_attention_item result: %w", err)
			}
			return result, nil
		},
	)

	srv.AddTool(
		mcp.NewTool(
			"kan.resolve_attention_item",
			mcp.WithDescription("Resolve one attention item by id."),
			mcp.WithString("id", mcp.Required(), mcp.Description("Attention item id")),
			mcp.WithString("resolved_by", mcp.Description("Actor resolving the item")),
			mcp.WithString("reason", mcp.Description("Resolution reason")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			itemID, err := req.RequireString("id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			item, err := attention.ResolveAttentionItem(ctx, common.ResolveAttentionItemRequest{
				ID:         itemID,
				ResolvedBy: req.GetString("resolved_by", ""),
				Reason:     req.GetString("reason", ""),
			})
			if err != nil {
				return toolResultFromError(err), nil
			}
			result, err := mcp.NewToolResultJSON(item)
			if err != nil {
				return nil, fmt.Errorf("encode resolve_attention_item result: %w", err)
			}
			return result, nil
		},
	)
}

// toolResultFromError maps service errors into MCP-visible tool errors.
func toolResultFromError(err error) *mcp.CallToolResult {
	switch {
	case err == nil:
		return mcp.NewToolResultError("unknown error")
	case errors.Is(err, common.ErrInvalidCaptureStateRequest), errors.Is(err, common.ErrUnsupportedScope):
		return mcp.NewToolResultError("invalid_request: " + err.Error())
	case errors.Is(err, common.ErrNotFound):
		return mcp.NewToolResultError("not_found: " + err.Error())
	case errors.Is(err, common.ErrAttentionUnavailable):
		return mcp.NewToolResultError("not_implemented: " + err.Error())
	default:
		return mcp.NewToolResultError("internal_error: " + err.Error())
	}
}
