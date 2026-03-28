package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	mcpjsonrpc "github.com/maximerivest/mcp2cli/internal/mcp/jsonrpc"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

// Session is the shared interface used by CLI commands.
type Session interface {
	ListTools(ctx context.Context) ([]types.Tool, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error)
	ListResources(ctx context.Context) ([]types.Resource, error)
	ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error)
	ListPrompts(ctx context.Context) ([]types.Prompt, error)
	GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error)
	Close() error
}

// HTTPClient is a minimal JSON-RPC-over-HTTP MCP client.
// Supports both plain JSON responses and Streamable HTTP (SSE) responses.
type HTTPClient struct {
	client    *http.Client
	url       string
	headers   map[string]string
	nextID    int64
	sessionID string // Mcp-Session-Id for Streamable HTTP
}

// DaemonChecker is an optional function that checks whether a local daemon
// is running for a server and returns an HTTP client + URL to reach it.
type DaemonChecker func(serverName string) (client *http.Client, url string, running bool)

// Connect selects the fastest available transport:
// 1. Running daemon (if DaemonCheck is set and finds one)
// 2. Remote HTTP (if server has a URL)
// 3. Local stdio (spawn process)
func Connect(ctx context.Context, server *config.Server, headers map[string]string, options ConnectOptions) (Session, error) {
	if server != nil && server.URL != "" {
		return ConnectHTTP(ctx, server, headers, options)
	}
	// Check for running daemon
	if options.DaemonCheck != nil && server != nil && server.Name != "" && server.Name != "(direct)" {
		if httpClient, url, running := options.DaemonCheck(server.Name); running {
			c := &HTTPClient{client: httpClient, url: url, headers: copyHeaders(headers)}
			// Daemon is already initialized — skip handshake
			return c, nil
		}
	}
	return ConnectStdio(ctx, server, options)
}

// ConnectHTTP connects to a remote MCP endpoint over HTTP and initializes it.
func ConnectHTTP(ctx context.Context, server *config.Server, headers map[string]string, options ConnectOptions) (*HTTPClient, error) {
	if server == nil {
		return nil, exitcode.New(exitcode.Internal, "server cannot be nil")
	}
	if server.URL == "" {
		return nil, exitcode.New(exitcode.Config, "http server url cannot be empty")
	}
	_ = options
	client := &HTTPClient{client: &http.Client{}, url: server.URL, headers: copyHeaders(headers)}
	if err := client.Initialize(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// Initialize performs the MCP initialize handshake.
func (c *HTTPClient) Initialize(ctx context.Context) error {
	request := types.InitializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]any{},
		ClientInfo:      types.Implementation{Name: "mcp2cli", Version: "dev"},
	}
	var result types.InitializeResult
	if err := c.call(ctx, "initialize", request, &result); err != nil {
		return wrapRPCError(exitcode.Protocol, err, "initialize MCP session")
	}
	if err := c.notify(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return exitcode.Wrap(exitcode.Protocol, err, "notify initialized")
	}
	return nil
}

// ListTools returns all tools exposed by the server.
func (c *HTTPClient) ListTools(ctx context.Context) ([]types.Tool, error) {
	var result types.ListToolsResult
	if err := c.call(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list tools")
	}
	return result.Tools, nil
}

// CallTool executes a tool with the provided arguments.
func (c *HTTPClient) CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error) {
	var result types.CallToolResult
	if err := c.call(ctx, "tools/call", types.CallToolParams{Name: name, Arguments: arguments}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("call tool %q", name))
	}
	return &result, nil
}

// ListResources returns all resources exposed by the server.
func (c *HTTPClient) ListResources(ctx context.Context) ([]types.Resource, error) {
	var result types.ListResourcesResult
	if err := c.call(ctx, "resources/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list resources")
	}
	return result.Resources, nil
}

// ReadResource reads one resource by URI.
func (c *HTTPClient) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error) {
	var result types.ReadResourceResult
	if err := c.call(ctx, "resources/read", types.ReadResourceParams{URI: uri}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("read resource %q", uri))
	}
	return &result, nil
}

// ListPrompts returns all prompts exposed by the server.
func (c *HTTPClient) ListPrompts(ctx context.Context) ([]types.Prompt, error) {
	var result types.ListPromptsResult
	if err := c.call(ctx, "prompts/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list prompts")
	}
	return result.Prompts, nil
}

// GetPrompt fetches a prompt with arguments.
func (c *HTTPClient) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	var result types.GetPromptResult
	if err := c.call(ctx, "prompts/get", types.GetPromptParams{Name: name, Arguments: arguments}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("get prompt %q", name))
	}
	return &result, nil
}

// Close closes the HTTP client session. It is a no-op for stateless HTTP transport.
func (c *HTTPClient) Close() error { return nil }

func (c *HTTPClient) call(ctx context.Context, method string, params any, result any) error {
	id := atomic.AddInt64(&c.nextID, 1)
	payload := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		payload["params"] = params
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "marshal request")
	}
	responsePayload, err := c.do(ctx, body)
	if err != nil {
		return err
	}
	var envelope struct {
		Result json.RawMessage      `json:"result"`
		Error  *mcpjsonrpc.RPCError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(responsePayload, &envelope); err != nil {
		return exitcode.Wrap(exitcode.Protocol, err, "decode response")
	}
	if envelope.Error != nil {
		return envelope.Error
	}
	if result == nil || len(envelope.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return exitcode.Wrap(exitcode.Protocol, err, "decode result")
	}
	return nil
}

func (c *HTTPClient) notify(ctx context.Context, method string, params any) error {
	payload := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		payload["params"] = params
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return exitcode.Wrap(exitcode.Internal, err, "marshal notification")
	}
	_, err = c.do(ctx, body)
	return err
}

func (c *HTTPClient) do(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Internal, err, "build HTTP request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "perform HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, exitcode.New(exitcode.Auth, "authentication required")
	}
	if resp.StatusCode/100 != 2 {
		return nil, exitcode.Newf(exitcode.Transport, "HTTP request failed with status %d", resp.StatusCode)
	}

	// Track session ID from Streamable HTTP servers
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/event-stream") {
		return c.readSSE(resp.Body)
	}

	responsePayload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "read HTTP response")
	}
	return responsePayload, nil
}

// readSSE parses a Server-Sent Events stream and returns the last JSON-RPC
// message data. Streamable HTTP servers send responses as SSE events.
func (c *HTTPClient) readSSE(body io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(body)
	var lastData []byte
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := []byte(strings.TrimPrefix(line, "data: "))
			// Check if this is a JSON-RPC response (has "id" or "result" or "error")
			if len(data) > 0 && data[0] == '{' {
				lastData = data
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "read SSE stream")
	}
	if lastData == nil {
		return nil, exitcode.New(exitcode.Protocol, "no JSON-RPC response in SSE stream")
	}
	return lastData, nil
}

func copyHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}
