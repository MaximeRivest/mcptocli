---
title: Quick Start
description: Get up and running in 30 seconds
---

Two steps: **add** a server once, then **use** it by name.

## Step 1: Add a server

```bash
mcp add time 'uvx mcp-server-time'
```

The second argument is the command to start the server. URLs are detected automatically:

```bash
mcp add notion https://mcp.notion.com/mcp --auth oauth
```

## Step 2: Use it

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

```bash
mcp time get-current-time --timezone 'America/New_York'
```

The server name **is** the command. Just the server name followed by what you want to do.

## What next?

- [Using Tools](/mcp2cli/guides/tools/) — inspect and call tools
- [Remote Servers](/mcp2cli/guides/remote/) — OAuth and bearer auth
- [Background Mode](/mcp2cli/guides/background/) — keep servers running for instant responses
- [Interactive Shell](/mcp2cli/guides/shell/) — explore interactively
