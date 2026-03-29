---
title: Background Mode
description: Keep servers running for instant responses
---

Local MCP servers are normally started on every command and shut down after. For repeated use, keep the server running in the background.

## Single-client mode

```bash
mcp time up
mcp time get-current-time 'America/New_York'   # instant
mcp time get-current-time 'Europe/London'       # instant
mcp time down
```

The server runs via stdio in the background. One client at a time, lowest latency.

## Shared mode

```bash
mcp time up --share
```

This starts the server in HTTP mode on a local port. Multiple clients can connect to the same server simultaneously:

- Another terminal running `mcp time ...`
- Claude Desktop
- A notebook
- Any MCP client that speaks HTTP

```bash
mcp time get-current-time 'America/New_York'   # auto-detects the running server
mcp time down                                   # stop when done
```

:::tip
Shared mode is ideal when you want one persistent server session shared across tools — for example, a database server where you want to keep the connection alive.
:::

## Checking status

```bash
mcp ls
```

```text
time (up)  uvx mcp-server-time
notion     https://mcp.notion.com/mcp
```

Servers that are running show `(up)` in the listing.
