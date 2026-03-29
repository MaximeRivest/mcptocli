---
title: Remote Servers
description: Connect to remote MCP servers with OAuth or bearer auth
---

## Bearer token

```bash
export ACME_TOKEN="your-api-key"
mcp add acme https://api.acme.dev/mcp --bearer-env ACME_TOKEN
mcp acme tools
mcp acme search --query invoices
```

The `--bearer-env` flag tells mcp2cli which environment variable holds the token. The token is never stored in config.

## OAuth (browser login)

```bash
mcp add notion https://mcp.notion.com/mcp --auth oauth
mcp notion tools
```

The browser opens automatically the first time. Tokens are persisted in your system keychain and refreshed automatically.

### Pre-authenticate

```bash
mcp login notion
```

This triggers the OAuth flow without calling any tool — useful for ensuring auth is ready before scripting.

## Custom headers

```bash
mcp tool --url https://api.example.com/mcp --header "X-API-Key: secret" list-items
```
