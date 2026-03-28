package client

import (
	"context"
	"errors"
	"fmt"
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
	rpc       *mcpjsonrpc.Client
	transport *stdio.Transport
	server    *config.Server
}

// ConnectStdio starts a stdio MCP session and performs the initialize handshake.
func ConnectStdio(ctx context.Context, server *config.Server) (*Client, error) {
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
		rpc:       mcpjsonrpc.NewClient(transport.Reader(), transport.Writer(), transport),
		transport: transport,
		server:    server,
	}
	if err := client.Initialize(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

// Initialize performs the MCP initialize handshake.
func (c *Client) Initialize(ctx context.Context) error {
	request := types.InitializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]any{},
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
		return nil, wrapRPCError(exitcode.Server, err, fmt.Sprintf("call tool %q", name))
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

func wrapRPCError(category exitcode.Category, err error, message string) error {
	var rpcErr *mcpjsonrpc.RPCError
	if errors.As(err, &rpcErr) {
		return exitcode.Wrap(category, rpcErr, message)
	}
	return exitcode.Wrap(category, err, message)
}
