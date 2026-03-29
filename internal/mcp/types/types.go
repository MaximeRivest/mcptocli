// Package types provides MCP type aliases bridging to mcp-go.
//
// All MCP protocol types are now provided by github.com/mark3labs/mcp-go/mcp.
// This package re-exports the subset used by mcp2cli, so existing code
// compiles with minimal changes. New code should import mcp-go directly.
package types

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// ── Core types (re-exported from mcp-go) ────────────────────

type Implementation = mcp.Implementation
type Resource = mcp.Resource
type Prompt = mcp.Prompt
type PromptArgument = mcp.PromptArgument

// ── Tool ────────────────────────────────────────────────────

// Tool wraps mcp.Tool to provide InputSchema as json.RawMessage,
// which the schema inspection code needs for ordered JSON parsing.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// FromMCPTool converts an mcp.Tool to our Tool with raw schema.
func FromMCPTool(t mcp.Tool) Tool {
	raw, _ := json.Marshal(t.InputSchema)
	return Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: raw,
	}
}

// FromMCPTools converts a slice of mcp.Tool.
func FromMCPTools(tools []mcp.Tool) []Tool {
	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = FromMCPTool(t)
	}
	return out
}

// ── Content & Results ───────────────────────────────────────

// Content is a simplified content block for rendering.
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// CallToolResult wraps mcp.CallToolResult with simplified Content.
type CallToolResult struct {
	Content           []Content      `json:"content,omitempty"`
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
	Meta              map[string]any `json:"_meta,omitempty"`
}

// FromMCPCallToolResult converts from mcp-go's result type.
func FromMCPCallToolResult(r *mcp.CallToolResult) *CallToolResult {
	if r == nil {
		return nil
	}
	// Convert StructuredContent from any to map[string]any
	var sc map[string]any
	if m, ok := r.StructuredContent.(map[string]any); ok {
		sc = m
	}
	out := &CallToolResult{
		StructuredContent: sc,
		IsError:           r.IsError,
	}
	for _, c := range r.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			out.Content = append(out.Content, Content{Type: "text", Text: v.Text})
		case mcp.ImageContent:
			out.Content = append(out.Content, Content{Type: "image"})
		case mcp.AudioContent:
			out.Content = append(out.Content, Content{Type: "audio"})
		default:
			out.Content = append(out.Content, Content{Type: "unknown"})
		}
	}
	return out
}

// ── Resources ───────────────────────────────────────────────

type ListResourcesResult = mcp.ListResourcesResult

// ResourceContent is a simplified resource content block.
type ResourceContent struct {
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents,omitempty"`
}

// FromMCPReadResourceResult converts from mcp-go's result type.
func FromMCPReadResourceResult(r *mcp.ReadResourceResult) *ReadResourceResult {
	if r == nil {
		return nil
	}
	out := &ReadResourceResult{}
	for _, c := range r.Contents {
		switch v := c.(type) {
		case mcp.TextResourceContents:
			out.Contents = append(out.Contents, ResourceContent{
				URI:      v.URI,
				MimeType: v.MIMEType,
				Text:     v.Text,
			})
		case mcp.BlobResourceContents:
			out.Contents = append(out.Contents, ResourceContent{
				URI:      v.URI,
				MimeType: v.MIMEType,
				Blob:     v.Blob,
			})
		}
	}
	return out
}

// ── Prompts ─────────────────────────────────────────────────

type ListPromptsResult = mcp.ListPromptsResult

type PromptMessage struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages,omitempty"`
}

// FromMCPGetPromptResult converts from mcp-go's result type.
func FromMCPGetPromptResult(r *mcp.GetPromptResult) *GetPromptResult {
	if r == nil {
		return nil
	}
	out := &GetPromptResult{Description: r.Description}
	for _, m := range r.Messages {
		content := Content{}
		switch v := m.Content.(type) {
		case mcp.TextContent:
			content = Content{Type: "text", Text: v.Text}
		}
		out.Messages = append(out.Messages, PromptMessage{
			Role:    string(m.Role),
			Content: content,
		})
	}
	return out
}

// ── Initialize ──────────────────────────────────────────────

type InitializeParams = mcp.InitializeParams
type InitializeResult = mcp.InitializeResult

// ── Elicitation ─────────────────────────────────────────────

// ElicitRequestParams is sent by the server when it needs user input.
type ElicitRequestParams struct {
	Message         string         `json:"message"`
	RequestedSchema map[string]any `json:"requestedSchema"`
}

// ElicitResult is returned by the client in response to elicitation/create.
type ElicitResult struct {
	Action  string         `json:"action"`
	Content map[string]any `json:"content,omitempty"`
}
