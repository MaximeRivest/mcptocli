---
title: Installation
description: Install mcp2cli on macOS, Linux, or Windows
---

## macOS and Linux

```bash
curl -fsSL https://raw.githubusercontent.com/MaximeRivest/mcp2cli/main/install.sh | sh
```

This downloads the right binary, installs it, sets up shell completions, and creates an `mcp` alias if available.

## Windows

```powershell
irm https://raw.githubusercontent.com/MaximeRivest/mcp2cli/main/install.ps1 | iex
```

## Verify

Open a **new** terminal and run:

```bash
mcp2cli version
```

If the `mcp` alias was installed, you can also use:

```bash
mcp version
```

:::tip
All examples in this documentation use `mcp` for brevity. If `mcp` is taken on your system, substitute `mcp2cli`.
:::

## Build from source

```bash
git clone https://github.com/MaximeRivest/mcp2cli.git
cd mcp2cli
go build -o mcp2cli ./cmd/mcp2cli
./mcp2cli version
```
