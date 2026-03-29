---
title: Interactive Shell
description: Explore MCP servers interactively
---

The shell keeps a persistent connection open and gives you tab completion, history, and the ability to switch output formats on the fly.

## Start a shell

```bash
mcp time shell
```

```text
time> tools
  convert-time      Convert time between timezones.
  get-current-time  Get current time in a specific timezone.

time> get-current-time 'America/New_York'
2025-03-15T14:30:00-04:00

time> set output json
output = json

time> get-current-time 'Europe/London'
{
  "content": [
    {
      "type": "text",
      "text": "2025-03-15T18:30:00+00:00"
    }
  ]
}

time> exit
```

## Shell commands

| Command | Description |
|---------|-------------|
| `tools` | List all tools |
| `resources` | List all resources |
| `prompts` | List all prompts |
| `<tool> [args...]` | Call a tool |
| `tool <name> [args...]` | Call a tool (explicit) |
| `resource <name>` | Read a resource |
| `prompt <name> [args...]` | Render a prompt |
| `set output <mode>` | Switch output format (auto, json, yaml, raw, table) |
| `help` | Show available commands |
| `exit` / `quit` | Exit the shell |

## Tab completion

Press `Tab` to complete:
- Tool names
- `--flag` names for the current tool
- Shell commands (`tools`, `resources`, `set output`, etc.)
