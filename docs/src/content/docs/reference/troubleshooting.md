---
title: Troubleshooting
description: Diagnose and fix common issues
---

## Doctor

The `doctor` command runs a series of checks against a server:

```bash
mcp time doctor
```

```text
CHECK    STATUS  DETAIL
resolve  ok      time
command  ok      /home/user/.local/bin/uvx
auth     ok      no auth required
connect  ok      initialize handshake succeeded
tools    ok      2 tool(s) available
```

## Common errors

### Command not found

```text
error: command "uvx" not found: executable file not found in $PATH
hint: make sure "uvx" is installed and on your PATH
```

The server command's executable isn't installed. Install it and make sure it's on your PATH.

### Server crashes on startup

```text
error: server exited (code 1) before completing MCP handshake
hint: npm error 404 Not Found - @foo/bar is not in this registry.
```

The server command starts but crashes before it can respond. The hint shows the server's stderr output. Try running the command directly to debug:

```bash
uvx mcp-server-time
```

### Server doesn't speak MCP

```text
error: server started but did not respond to MCP handshake
hint: the command may not be an MCP server, or it may need arguments.
```

The command runs but doesn't produce MCP-compatible JSON-RPC output on stdout. Make sure the command is actually an MCP server.

### Connection timeout

```text
error: context deadline exceeded
```

The server took too long to respond. Try increasing the timeout:

```bash
mcp time tools --timeout 60s
```

### OAuth errors

If OAuth login fails or tokens expire:

```bash
mcp login notion
```

This triggers a fresh login flow.

## Shell completions not working

Make sure you've sourced the completions in your shell config:

```bash
# bash
source <(mcp2cli completion bash)

# zsh
source <(mcp2cli completion zsh)

# fish
mcp2cli completion fish | source
```

Then open a new terminal.
