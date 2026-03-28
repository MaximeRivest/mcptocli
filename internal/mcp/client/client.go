package client

import (
	"context"
	"fmt"
	"time"

	"github.com/maximerivest/mcp2cli/internal/config"
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
		return nil, fmt.Errorf("server cannot be nil")
	}
	if server.Command == "" {
		return nil, fmt.Errorf("stdio server command cannot be empty")
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
		return fmt.Errorf("initialize MCP session: %w", err)
	}
	if err := c.rpc.Notify(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return fmt.Errorf("notify initialized: %w", err)
	}
	return nil
}

// ListTools returns all tools exposed by the server.
func (c *Client) ListTools(ctx context.Context) ([]types.Tool, error) {
	var result types.ListToolsResult
	if err := c.rpc.Call(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	return result.Tools, nil
}

// CallTool executes a tool with the provided arguments.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (*types.CallToolResult, error) {
	request := types.CallToolParams{Name: name, Arguments: arguments}
	var result types.CallToolResult
	if err := c.rpc.Call(ctx, "tools/call", request, &result); err != nil {
		return nil, fmt.Errorf("call tool %q: %w", name, err)
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
