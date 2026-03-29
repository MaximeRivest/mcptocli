---
title: All Commands
description: Complete command reference
---

## Server Management

| Command | Description |
|---------|-------------|
| `mcp add <name> <command-or-url>` | Save a server under a name |
| `mcp ls` | List saved servers |
| `mcp rm <name>` | Remove a saved server |
| `mcp expose <server>` | Create a standalone command for a server |
| `mcp expose --remove <server>` | Remove an exposed command |

## Using a Server

| Command | Description |
|---------|-------------|
| `mcp <server> tools` | List all tools |
| `mcp <server> tools <tool>` | Inspect a tool's schema and usage |
| `mcp <server> <tool> [args...]` | Call a tool |
| `mcp <server> resources` | List all resources |
| `mcp <server> resource <name>` | Read a resource |
| `mcp <server> prompts` | List all prompts |
| `mcp <server> prompt <name> [args...]` | Render a prompt |
| `mcp <server> shell` | Open an interactive shell |
| `mcp <server> doctor` | Diagnose connection issues |
| `mcp <server> login` | Pre-authenticate (OAuth/bearer) |

## Background / Sharing

| Command | Description |
|---------|-------------|
| `mcp <server> up` | Start server in background (stdio, single client) |
| `mcp <server> up --share` | Start server in HTTP mode (multiple clients) |
| `mcp <server> down` | Stop background server |

## One-off Use

| Command | Description |
|---------|-------------|
| `mcp tools --command '<cmd>'` | List tools without registering |
| `mcp tool --command '<cmd>' <tool> [args...]` | Call a tool without registering |
| `mcp tools --url <url>` | List tools on a remote server |
| `mcp tool --url <url> <tool> [args...]` | Call a tool on a remote server |

## Global Flags

| Flag | Description |
|------|-------------|
| `-o, --output <format>` | Output format: auto, json, yaml, raw, table |
| `--timeout <duration>` | Request timeout (default: 30s) |
| `-h, --help` | Show help |

## Connection Flags

These are available on `tools`, `tool`, `resources`, `resource`, `prompts`, `prompt`, `shell`, `doctor`, and `login`:

| Flag | Description |
|------|-------------|
| `--command <cmd>` | Local server command (instead of registered name) |
| `--url <url>` | Remote server URL |
| `--auth <mode>` | Auth mode (oauth, bearer) |
| `--bearer-env <var>` | Environment variable holding a bearer token |
| `--header <key: value>` | Additional HTTP header (repeatable) |
| `--cwd <dir>` | Working directory for local commands |
| `--env <KEY=VALUE>` | Environment override (repeatable) |
