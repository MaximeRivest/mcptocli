---
title: Exposed Commands
description: Use MCP servers as standalone commands
---

When you add a server, `mcp2cli` automatically creates a standalone command for it on your PATH.

## Automatic exposure

```bash
mcp add time 'uvx mcp-server-time'
# → creates mcp-time
```

Now you can use it directly:

```bash
mcp-time tools
mcp-time get-current-time 'America/New_York'
```

## Custom names

```bash
mcp expose time --as t
```

```bash
t tools
t get-current-time 'America/New_York'
```

## Remove an exposed command

```bash
mcp expose --remove time
```

## How it works

Exposed commands are small shims placed in `~/.local/share/mcp2cli/bin/`. When invoked, they call `mcp2cli` with the correct server binding.

Make sure the bin directory is on your PATH:

```bash
export PATH="$HOME/.local/share/mcp2cli/bin:$PATH"
```

The `add` command prints the path when it creates the shim.
