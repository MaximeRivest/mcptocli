package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	Close() error
}

// HTTPClient is a minimal JSON-RPC-over-HTTP MCP client.
type HTTPClient struct {
	client  *http.Client
	url     string
	headers map[string]string
	nextID  int64
}

// Connect selects stdio or HTTP transport based on the resolved server.
func Connect(ctx context.Context, server *config.Server, headers map[string]string) (Session, error) {
	if server != nil && server.URL != "" {
		return ConnectHTTP(ctx, server, headers)
	}
	return ConnectStdio(ctx, server)
}

// ConnectHTTP connects to a remote MCP endpoint over HTTP and initializes it.
func ConnectHTTP(ctx context.Context, server *config.Server, headers map[string]string) (*HTTPClient, error) {
	if server == nil {
		return nil, exitcode.New(exitcode.Internal, "server cannot be nil")
	}
	if server.URL == "" {
		return nil, exitcode.New(exitcode.Config, "http server url cannot be empty")
	}
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
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "perform HTTP request")
	}
	defer resp.Body.Close()
	responsePayload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "read HTTP response")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, exitcode.New(exitcode.Auth, "authentication required")
	}
	if resp.StatusCode/100 != 2 {
		return nil, exitcode.Newf(exitcode.Transport, "HTTP request failed with status %d", resp.StatusCode)
	}
	return responsePayload, nil
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
