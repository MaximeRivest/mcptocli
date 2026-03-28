package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
	"github.com/maximerivest/mcp2cli/internal/exitcode"
	mcpjsonrpc "github.com/maximerivest/mcp2cli/internal/mcp/jsonrpc"
	"github.com/maximerivest/mcp2cli/internal/mcp/transport/stdio"
	"github.com/maximerivest/mcp2cli/internal/mcp/types"
)

const protocolVersion = "2024-11-05"

// Client is a minimal MCP client for stdio transports.
type Client struct {
	rpc                *mcpjsonrpc.Client
	transport          *stdio.Transport
	server             *config.Server
	elicitationHandler ElicitationHandler
	callbackErrMu      sync.Mutex
	callbackErr        error
}

// ElicitationHandler handles server requests for user input.
type ElicitationHandler func(ctx context.Context, params types.ElicitRequestParams) (*types.ElicitResult, error)

// ConnectOptions configures optional client behaviors.
type ConnectOptions struct {
	ElicitationHandler ElicitationHandler
}

// ConnectStdio starts a stdio MCP session and performs the initialize handshake.
func ConnectStdio(ctx context.Context, server *config.Server, options ConnectOptions) (*Client, error) {
	if server == nil {
		return nil, exitcode.New(exitcode.Internal, "server cannot be nil")
	}
	if server.Command == "" {
		return nil, exitcode.New(exitcode.Config, "stdio server command cannot be empty")
	}

	transport, err := stdio.Start(ctx, server.Command, server.CWD, server.Env)
	if err != nil {
		return nil, err
	}

	client := &Client{
		transport:          transport,
		server:             server,
		elicitationHandler: options.ElicitationHandler,
	}
	client.rpc = mcpjsonrpc.NewClient(transport.Reader(), transport.Writer(), transport, client.handleRequest)
	if err := client.Initialize(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

// Initialize performs the MCP initialize handshake.
func (c *Client) Initialize(ctx context.Context) error {
	capabilities := map[string]any{}
	if c.elicitationHandler != nil {
		capabilities["elicitation"] = map[string]any{}
	}
	request := types.InitializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities,
		ClientInfo: types.Implementation{
			Name:    "mcp2cli",
			Version: "dev",
		},
	}
	var result types.InitializeResult
	if err := c.rpc.Call(ctx, "initialize", request, &result); err != nil {
		return wrapRPCError(exitcode.Protocol, err, "initialize MCP session")
	}
	if err := c.rpc.Notify(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return exitcode.Wrap(exitcode.Protocol, err, "notify initialized")
	}
	return nil
}

// ListTools returns all tools exposed by the server.
func (c *Client) ListTools(ctx context.Context) ([]types.Tool, error) {
	var result types.ListToolsResult
	if err := c.rpc.Call(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list tools")
	}
	return result.Tools, nil
}

// CallTool executes a tool with the provided arguments.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error) {
	request := types.CallToolParams{Name: name, Arguments: arguments}
	var result types.CallToolResult
	if err := c.rpc.Call(ctx, "tools/call", request, &result); err != nil {
		if callbackErr := c.consumeCallbackErr(); callbackErr != nil {
			return nil, callbackErr
		}
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("call tool %q", name))
	}
	return &result, nil
}

// ListResources returns all resources exposed by the server.
func (c *Client) ListResources(ctx context.Context) ([]types.Resource, error) {
	var result types.ListResourcesResult
	if err := c.rpc.Call(ctx, "resources/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list resources")
	}
	return result.Resources, nil
}

// ReadResource reads one resource by URI.
func (c *Client) ReadResource(ctx context.Context, uri string) (*types.ReadResourceResult, error) {
	var result types.ReadResourceResult
	if err := c.rpc.Call(ctx, "resources/read", types.ReadResourceParams{URI: uri}, &result); err != nil {
		if callbackErr := c.consumeCallbackErr(); callbackErr != nil {
			return nil, callbackErr
		}
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("read resource %q", uri))
	}
	return &result, nil
}

// ListPrompts returns all prompts exposed by the server.
func (c *Client) ListPrompts(ctx context.Context) ([]types.Prompt, error) {
	var result types.ListPromptsResult
	if err := c.rpc.Call(ctx, "prompts/list", map[string]any{}, &result); err != nil {
		return nil, wrapRPCError(exitcode.Protocol, err, "list prompts")
	}
	return result.Prompts, nil
}

// GetPrompt fetches a prompt with arguments.
func (c *Client) GetPrompt(ctx context.Context, name string, arguments map[string]string) (*types.GetPromptResult, error) {
	var result types.GetPromptResult
	if err := c.rpc.Call(ctx, "prompts/get", types.GetPromptParams{Name: name, Arguments: arguments}, &result); err != nil {
		if callbackErr := c.consumeCallbackErr(); callbackErr != nil {
			return nil, callbackErr
		}
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("get prompt %q", name))
	}
	return &result, nil
}

// Close closes the underlying transport.
func (c *Client) Close() error {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.Close()
}

// DefaultTimeout returns the default timeout for one-shot MCP operations.
func DefaultTimeout() time.Duration {
	return 30 * time.Second
}

func (c *Client) handleRequest(ctx context.Context, method string, params json.RawMessage) (any, *mcpjsonrpc.RPCError, bool) {
	switch method {
	case "elicitation/create":
		if c.elicitationHandler == nil {
			return nil, &mcpjsonrpc.RPCError{Code: -32601, Message: "elicitation not supported"}, true
		}
		var request types.ElicitRequestParams
		if err := json.Unmarshal(params, &request); err != nil {
			return nil, &mcpjsonrpc.RPCError{Code: -32602, Message: err.Error()}, true
		}
		result, err := c.elicitationHandler(ctx, request)
		if err != nil {
			c.setCallbackErr(err)
			return nil, &mcpjsonrpc.RPCError{Code: -32000, Message: err.Error()}, true
		}
		return result, nil, true
	default:
		return nil, nil, false
	}
}

func (c *Client) setCallbackErr(err error) {
	if err == nil {
		return
	}
	c.callbackErrMu.Lock()
	defer c.callbackErrMu.Unlock()
	c.callbackErr = err
}

func (c *Client) consumeCallbackErr() error {
	c.callbackErrMu.Lock()
	defer c.callbackErrMu.Unlock()
	err := c.callbackErr
	c.callbackErr = nil
	return err
}

func wrapRPCError(category exitcode.Category, err error, message string) error {
	var rpcErr *mcpjsonrpc.RPCError
	if errors.As(err, &rpcErr) {
		return exitcode.Wrap(category, rpcErr, message)
	}
	return exitcode.Wrap(category, err, message)
}
