// Package client wraps mcp-go's client for mcptocli.
//
// All MCP protocol handling is delegated to github.com/mark3labs/mcp-go/client.
// This package provides the Session interface and Connect functions that
// mcptocli's CLI code expects, adapting mcp-go's types as needed.
package client

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/kballard/go-shellquote"
	"encoding/json"

	mcpgo "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/maximerivest/mcptocli/internal/config"
	"github.com/maximerivest/mcptocli/internal/exitcode"
	"github.com/maximerivest/mcptocli/internal/mcp/types"
)

// Session is the interface used by CLI commands to talk to MCP servers.
type Session interface {
	ListTools(ctx context.Context) ([]types.Tool, error)
	CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error)
	ListResources(ctx context.Context) ([]mcp.Resource, error)
	ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error)
	ListPrompts(ctx context.Context) ([]mcp.Prompt, error)
	GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error)
	Close() error
}

// ElicitationHandler handles server requests for user input.
type ElicitationHandler func(ctx context.Context, params types.ElicitRequestParams) (*types.ElicitResult, error)

// ConnectOptions configures optional client behaviors.
type ConnectOptions struct {
	ElicitationHandler ElicitationHandler
	DaemonCheck        DaemonChecker
}

// DaemonChecker checks whether a local daemon is running.
// Returns an HTTP client configured for the daemon, the URL, and whether it's running.
type DaemonChecker func(serverName string) (client *http.Client, url string, running bool)

// DefaultTimeout returns the default timeout for one-shot MCP operations.
func DefaultTimeout() time.Duration {
	return 30 * time.Second
}

// Connect selects the best available transport and returns a Session.
func Connect(ctx context.Context, server *config.Server, headers map[string]string, options ConnectOptions) (Session, error) {
	if server == nil {
		return nil, exitcode.New(exitcode.Internal, "server cannot be nil")
	}

	// Remote HTTP
	if server.URL != "" {
		return ConnectHTTP(ctx, server, headers, options)
	}

	// Check for running daemon
	if options.DaemonCheck != nil && server.Name != "" && server.Name != "(direct)" {
		if httpClient, url, running := options.DaemonCheck(server.Name); running {
			return newDaemonSession(httpClient, url), nil
		}
	}

	// Stdio
	return ConnectStdio(ctx, server, options)
}

// ConnectHTTP connects to a remote MCP endpoint over HTTP.
func ConnectHTTP(ctx context.Context, server *config.Server, headers map[string]string, options ConnectOptions) (Session, error) {
	if server.URL == "" {
		return nil, exitcode.New(exitcode.Config, "http server url cannot be empty")
	}

	var httpOpts []transport.StreamableHTTPCOption
	if len(headers) > 0 {
		httpOpts = append(httpOpts, transport.WithHTTPHeaders(headers))
	}

	c, err := mcpgo.NewStreamableHttpClient(server.URL, httpOpts...)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "create HTTP client")
	}

	s := &session{client: c}
	if err := s.initialize(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return s, nil
}

// ConnectStdio starts a stdio MCP session.
func ConnectStdio(ctx context.Context, server *config.Server, options ConnectOptions) (Session, error) {
	if server.Command == "" {
		return nil, exitcode.New(exitcode.Config, "stdio server command cannot be empty")
	}

	// Validate command exists
	args := splitCommand(server.Command)
	if len(args) == 0 {
		return nil, exitcode.New(exitcode.Config, "command cannot be empty")
	}
	if _, err := exec.LookPath(args[0]); err != nil {
		return nil, exitcode.WithHint(
			exitcode.Wrapf(exitcode.Config, err, "command %q not found", args[0]),
			fmt.Sprintf("make sure %q is installed and on your PATH", args[0]),
		)
	}

	var env []string
	for k, v := range server.Env {
		env = append(env, k+"="+v)
	}

	// Create transport (don't start yet — client.Start wires handlers)
	stdioTransport := transport.NewStdio(args[0], env, args[1:]...)

	// Build client options
	var clientOpts []mcpgo.ClientOption
	if options.ElicitationHandler != nil {
		clientOpts = append(clientOpts, mcpgo.WithElicitationHandler(
			&elicitAdapter{handler: options.ElicitationHandler},
		))
	}

	c := mcpgo.NewClient(stdioTransport, clientOpts...)

	// Start transport + wire notification/request handlers
	if err := c.Start(ctx); err != nil {
		return nil, exitcode.Wrap(exitcode.Transport, err, "start stdio server")
	}

	s := &session{client: c}
	if err := s.initialize(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return s, nil
}

// session wraps mcp-go's Client and implements Session.
type session struct {
	client *mcpgo.Client
}

func (s *session) initialize(ctx context.Context) error {
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "mcptocli",
		Version: "dev",
	}

	if _, err := s.client.Initialize(ctx, initReq); err != nil {
		return exitcode.Wrap(exitcode.Protocol, err, "initialize MCP session")
	}
	return nil
}

func (s *session) ListTools(ctx context.Context) ([]types.Tool, error) {
	result, err := s.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Protocol, err, "list tools")
	}
	return types.FromMCPTools(result.Tools), nil
}

func (s *session) CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = arguments

	result, err := s.client.CallTool(ctx, req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Server, err, fmt.Sprintf("call tool %q", name))
	}
	return types.FromMCPCallToolResult(result), nil
}

func (s *session) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	result, err := s.client.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		// Servers that don't support resources return "method not found" — treat as empty.
		if isMethodNotFound(err) {
			return nil, nil
		}
		return nil, exitcode.Wrap(exitcode.Protocol, err, "list resources")
	}
	return result.Resources, nil
}

func (s *session) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error) {
	req := mcp.ReadResourceRequest{}
	req.Params.URI = uri

	result, err := s.client.ReadResource(ctx, req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Server, err, fmt.Sprintf("read resource %q", uri))
	}
	return types.FromMCPReadResourceResult(result), nil
}

func (s *session) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	result, err := s.client.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		// Servers that don't support prompts return "method not found" — treat as empty.
		if isMethodNotFound(err) {
			return nil, nil
		}
		return nil, exitcode.Wrap(exitcode.Protocol, err, "list prompts")
	}
	return result.Prompts, nil
}

func (s *session) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	req := mcp.GetPromptRequest{}
	req.Params.Name = name
	req.Params.Arguments = arguments

	result, err := s.client.GetPrompt(ctx, req)
	if err != nil {
		return nil, exitcode.Wrap(exitcode.Server, err, fmt.Sprintf("get prompt %q", name))
	}
	return types.FromMCPGetPromptResult(result), nil
}

func (s *session) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// elicitAdapter adapts our ElicitationHandler to mcp-go's ElicitationHandler interface.
type elicitAdapter struct {
	handler ElicitationHandler
}

func (a *elicitAdapter) Elicit(ctx context.Context, request mcp.ElicitationRequest) (*mcp.ElicitationResult, error) {
	// Convert mcp-go's request to our types
	schemaMap := map[string]any{}
	if request.Params.RequestedSchema != nil {
		raw, _ := json.Marshal(request.Params.RequestedSchema)
		json.Unmarshal(raw, &schemaMap)
	}

	params := types.ElicitRequestParams{
		Message:         request.Params.Message,
		RequestedSchema: schemaMap,
	}

	result, err := a.handler(ctx, params)
	if err != nil {
		return nil, err
	}

	mcpResult := &mcp.ElicitationResult{}
	mcpResult.Action = mcp.ElicitationResponseAction(result.Action)
	mcpResult.Content = result.Content
	return mcpResult, nil
}

// daemonSession talks plain JSON-RPC over HTTP to our local proxy daemon.
// Unlike StreamableHTTP, it doesn't do session negotiation — just POST.
type daemonSession struct {
	httpClient *http.Client
	url        string
}

func newDaemonSession(httpClient *http.Client, url string) *daemonSession {
	return &daemonSession{httpClient: httpClient, url: url}
}

func (d *daemonSession) rpc(ctx context.Context, method string, params any) (json.RawMessage, error) {
	reqBody, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("daemon request: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("%s", rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}

func (d *daemonSession) ListTools(ctx context.Context) ([]types.Tool, error) {
	raw, err := d.rpc(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var result mcp.ListToolsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return types.FromMCPTools(result.Tools), nil
}

func (d *daemonSession) CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error) {
	raw, err := d.rpc(ctx, "tools/call", map[string]any{"name": name, "arguments": arguments})
	if err != nil {
		return nil, err
	}
	var result mcp.CallToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return types.FromMCPCallToolResult(&result), nil
}

func (d *daemonSession) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	raw, err := d.rpc(ctx, "resources/list", map[string]any{})
	if err != nil {
		if isMethodNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var result mcp.ListResourcesResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Resources, nil
}

func (d *daemonSession) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error) {
	raw, err := d.rpc(ctx, "resources/read", map[string]any{"uri": uri})
	if err != nil {
		return nil, err
	}
	var result mcp.ReadResourceResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return types.FromMCPReadResourceResult(&result), nil
}

func (d *daemonSession) ListPrompts(ctx context.Context) ([]mcp.Prompt, error) {
	raw, err := d.rpc(ctx, "prompts/list", map[string]any{})
	if err != nil {
		if isMethodNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	var result mcp.ListPromptsResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Prompts, nil
}

func (d *daemonSession) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	raw, err := d.rpc(ctx, "prompts/get", map[string]any{"name": name, "arguments": arguments})
	if err != nil {
		return nil, err
	}
	var result mcp.GetPromptResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return types.FromMCPGetPromptResult(&result), nil
}

func (d *daemonSession) Close() error {
	return nil
}

// isMethodNotFound returns true if the error indicates the server doesn't support the method.
func isMethodNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Method not found") || strings.Contains(s, "method not found")
}

// splitCommand splits a shell command string into args using proper shell quoting.
func splitCommand(command string) []string {
	args, err := shellquote.Split(command)
	if err != nil {
		// Fallback: treat as single command
		return []string{command}
	}
	return args
}
