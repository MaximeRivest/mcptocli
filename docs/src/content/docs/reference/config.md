---
title: Configuration
description: Config files and how servers are stored
---

## Config files

mcp2cli stores server definitions in YAML config files:

- **Global:** `~/.config/mcp2cli/config.yaml`
- **Per-project:** `.mcp2cli.yaml` in any directory (use `--local` flag with `add`)

Per-project config is found by walking up from the current directory.

## Example config

```yaml
servers:
  time:
    command: uvx mcp-server-time
    expose_as:
      - mcp-time
  notion:
    url: https://mcp.notion.com/mcp
    auth: oauth
  acme:
    url: https://api.acme.dev/mcp
    bearer_env: ACME_TOKEN
```

## Server fields

| Field | Description |
|-------|-------------|
| `command` | Shell command to start a local stdio server |
| `url` | URL for a remote HTTP server |
| `auth` | Auth mode: `oauth` or `bearer` |
| `bearer_env` | Environment variable holding a bearer token |
| `expose_as` | List of exposed command names |
| `roots` | Root paths to advertise to the server |

## Data directory

mcp2cli stores runtime data in `~/.local/share/mcp2cli/`:

- `bin/` — exposed command shims
- `cache/` — metadata cache (tools, resources, prompts) for fast completions
- `daemons/` — PID files and sockets for background servers
- `history/` — shell history per server
