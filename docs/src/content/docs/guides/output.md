---
title: Output Formats
description: Control how results are displayed
---

## Formats

```bash
mcp time get-current-time 'America/New_York'            # human-readable (default)
mcp time get-current-time 'America/New_York' -o json     # JSON for scripts
mcp time get-current-time 'America/New_York' -o yaml     # YAML
mcp time get-current-time 'America/New_York' -o raw      # raw text, no formatting
```

## Scripting

`-o json` is always script-safe:

- Tool output goes to `stdout`
- Diagnostics go to `stderr`
- Exit codes are stable and meaningful

```bash
# Use with jq
mcp time get-current-time 'UTC' -o json | jq -r '.content[0].text'

# Use in a script
if mcp acme check-status -o raw; then
  echo "OK"
fi
```

## Listing formats

The `tools`, `resources`, and `prompts` commands also support `-o`:

```bash
mcp time tools -o json     # full tool schemas as JSON
mcp time tools -o yaml     # YAML
mcp time tools -o raw      # just tool names, one per line
```

`-o raw` is useful for piping into other commands:

```bash
mcp time tools -o raw | xargs -I{} mcp time tools {}
```
