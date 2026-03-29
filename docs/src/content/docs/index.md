---
title: mcp2cli
description: Turn any MCP server into a CLI
template: splash
---

<div class="landing">
  <h1 class="landing-title">mcp2cli</h1>
  <p class="landing-tagline">Turn any MCP server into a command-line tool.</p>
</div>

If a service has an MCP server, `mcp2cli` lets you use it from your terminal — no SDK, no boilerplate, just commands.

```bash
mcp add time 'uvx mcp-server-time'
mcp time get-current-time --timezone America/New_York
```

## What you get

- **Add once, use forever** — save a server by name, call its tools like shell commands
- **Everything auto-generated** — flags, help text, and tab completion come from the server's schema
- **Works everywhere** — local servers, remote APIs, OAuth, bearer tokens
- **Fast when you need it** — keep a server running in the background with `mcp time up`
- **Share across clients** — one server, multiple terminals, Claude Desktop, notebooks (`mcp time up --share`)
