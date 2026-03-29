---
title: Using Tools
description: Discover, inspect, and call MCP tools
---

## List tools

```bash
mcp time tools
```

```text
Tools (2):

  convert-time      Convert time between timezones.
  get-current-time  Get current time in a specific timezone.

Inspect:  mcp tools time <tool>
Invoke:   mcp time <tool> [args...]
```

## Inspect a tool

```bash
mcp time tools get-current-time
```

```text
NAME
  get-current-time - Get current time in a specific timezone

USAGE
  time get-current-time --timezone <string>

ARGS
  --timezone string  Required. IANA timezone name (e.g., 'America/New_York').
```

## Call a tool

```bash
mcp time get-current-time --timezone 'America/New_York'
```

### Positional arguments

Required scalar arguments can be passed positionally:

```bash
mcp time get-current-time 'America/New_York'
```

### One-off use (no registration)

You don't have to register a server to use it:

```bash
mcp tools --command 'uvx mcp-server-time'
mcp tool --command 'uvx mcp-server-time' get-current-time 'America/New_York'
mcp tool --url https://api.example.com/mcp search --query test
```
