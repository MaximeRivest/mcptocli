---
title: Arguments
description: How mcp2cli maps tool schemas to CLI flags
---

`mcp2cli` reads the tool's JSON schema and generates CLI flags automatically.

## Named flags

```bash
mcp time get-current-time --timezone 'America/New_York'
```

## Positional arguments

Required scalar arguments can be passed positionally, in schema order:

```bash
mcp time get-current-time 'America/New_York'
```

## Booleans

```bash
mcp api update --dry-run
mcp api update --no-dry-run
```

## Arrays

Repeat the flag for multiple values:

```bash
mcp api search --tag cli --tag go --tag mcp
```

## Structured JSON

From a file:

```bash
mcp api create --payload @data.json
```

From stdin:

```bash
cat data.json | mcp api create --payload @-
```

## Raw JSON input

For complex schemas, bypass flag parsing entirely:

```bash
mcp api complex-tool --input '{"nested": {"key": "value"}}'
```

The `--input` flag accepts a raw JSON object and sends it directly as the tool's arguments.
