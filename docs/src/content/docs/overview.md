---
title: mcp2cli
description: Turn any MCP server into a CLI
---

# mcp2cli

**Turn any MCP server into a command-line tool.**

MCP (Model Context Protocol) is an open standard that lets services describe their capabilities in a machine-readable way. More and more services are exposing MCP interfaces — databases, APIs, AI tools, local apps.

`mcp2cli` lets you use any of them from your terminal, as if they were native shell commands.

```bash
mcp add time 'uvx mcp-server-time'
mcp time get-current-time --timezone America/New_York
```

You don't need to understand the protocol. If a service has an MCP server, `mcp2cli` turns it into a CLI.

## Why mcp2cli?

- **Schema-driven flags** — tool parameters become `--flags` automatically
- **Tab completion** — tools, arguments, server names all complete in your shell
- **Background mode** — keep a server running for instant responses (`mcp time up`)
- **Shared mode** — one server, multiple clients (`mcp time up --share`)
- **Exposed commands** — `mcp-time` works like a native command on your PATH
- **Interactive shell** — explore tools with history and completion
- **Works with any MCP server** — local (stdio) or remote (HTTP), with OAuth or bearer auth

## How it relates to mcp2py

[mcp2py](https://github.com/maximerivest/mcp2py) turns MCP servers into **Python modules** — great for notebooks and scripts.

`mcp2cli` turns MCP servers into **shell commands** — great for terminals, shell scripts, and CI.

They complement each other. Use whichever fits your workflow.
