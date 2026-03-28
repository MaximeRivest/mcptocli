# mcp2cli: Turn any MCP server into a CLI

If [mcp2py](https://github.com/maximerivest/mcp2py) turns an MCP server into a Python module, `mcp2cli` turns an MCP server into a command-line tool you can use from any terminal.

## What is MCP?

MCP (Model Context Protocol) is an emerging standard that lets AI tools, APIs, and local apps describe what they can do in a machine-readable way. More and more services are exposing MCP interfaces. When they do, you don't have to learn their custom API — you just point `mcp2cli` at them and start using their tools from your terminal.

**You don't need to understand the protocol.** All you need to know is:

> If a service has an MCP server, `mcp2cli` lets you use it as if it were a normal command-line tool.

---

## Installation

`mcp2cli` is a single binary. No Go, Python, or Node.js runtime needed.

### macOS

Open **Terminal** and paste:

```bash
# Apple Silicon (M1/M2/M3/M4 — most modern Macs)
curl -L -o mcp2cli https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-darwin-arm64
chmod +x mcp2cli
sudo mv mcp2cli /usr/local/bin/
```

<details>
<summary>Intel Mac?</summary>

```bash
curl -L -o mcp2cli https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-darwin-amd64
chmod +x mcp2cli
sudo mv mcp2cli /usr/local/bin/
```

</details>

### Linux

Open a terminal and paste:

```bash
curl -L -o mcp2cli https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-linux-amd64
chmod +x mcp2cli
sudo mv mcp2cli /usr/local/bin/
```

<details>
<summary>ARM64 (Raspberry Pi, etc.)?</summary>

```bash
curl -L -o mcp2cli https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-linux-arm64
chmod +x mcp2cli
sudo mv mcp2cli /usr/local/bin/
```

</details>

### Windows

1. Download [`mcp2cli.exe`](https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-windows-amd64.exe)
2. Rename the file to `mcp2cli.exe`
3. Put it somewhere on your `PATH`, for example `C:\Users\YourName\bin\`

Or paste this into **PowerShell**:

```powershell
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\bin" | Out-Null
Invoke-WebRequest -Uri "https://github.com/MaximeRivest/mcp2cli/releases/latest/download/mcp2cli-windows-amd64.exe" -OutFile "$env:USERPROFILE\bin\mcp2cli.exe"
$env:PATH += ";$env:USERPROFILE\bin"
```

### Check that it worked

```bash
mcp2cli version
```

---

## Quick start

Two steps: **add** a server once, then **use** it by name.

### Step 1: Add a server

```bash
mcp2cli add weather 'npx -y @h1deya/mcp-server-weather'
```

That's it. The second argument is the command to start the server. URLs are detected automatically:

```bash
mcp2cli add notion https://mcp.notion.com/mcp --auth oauth
```

### Step 2: Use it

```bash
mcp2cli weather tools
```

```text
get-alerts      Get active weather alerts for a US state
get-forecast    Get forecast for a latitude/longitude pair
```

```bash
mcp2cli weather get-forecast --latitude 37.7 --longitude -122.4
```

The server name **is** the command. No `tool` vs `tools` to remember — just the server name followed by what you want to do.

---

## Daily use

Once a server is added, here's everything you need:

```bash
mcp2cli weather tools                                          # list tools
mcp2cli weather get-forecast --latitude 37.7 --longitude -122.4  # call a tool
mcp2cli weather get-forecast 37.7 -122.4                       # positional args work too
mcp2cli weather resources                                      # list resources
mcp2cli weather resource api-docs                              # read a resource
mcp2cli weather prompts                                        # list prompts
mcp2cli weather prompt review-code --code @main.go             # render a prompt
mcp2cli weather shell                                          # interactive mode
mcp2cli weather doctor                                         # diagnose problems
```

### Inspect a tool before calling it

```bash
mcp2cli weather tools get-forecast
```

```text
NAME
  get-forecast - Get weather forecast for a location

USAGE
  mcp2cli weather get-forecast --latitude <float> --longitude <float>

ARGS
  --latitude float   Required. Latitude of the location.
  --longitude float  Required. Longitude of the location.
```

### Interactive shell

```bash
mcp2cli weather shell
```

```text
weather> tools
weather> get-forecast 37.7 -122.4
weather> resources
weather> resource api-docs
weather> set output json
weather> get-forecast 37.7 -122.4
weather> exit
```

The shell keeps the connection open, supports history and tab completion, and lets you switch output formats on the fly.

---

## Remote servers

### With a bearer token

```bash
export ACME_TOKEN="your-api-key"
mcp2cli add acme https://api.acme.dev/mcp --bearer-env ACME_TOKEN
mcp2cli acme tools
mcp2cli acme search --query invoices
```

### With OAuth (browser login)

```bash
mcp2cli add notion https://mcp.notion.com/mcp --auth oauth
mcp2cli notion tools
# Browser opens automatically the first time
```

---

## Output formats

```bash
mcp2cli weather get-forecast 37.7 -122.4           # human-readable (default)
mcp2cli weather get-forecast 37.7 -122.4 -o json    # exact JSON for scripts
mcp2cli weather get-forecast 37.7 -122.4 -o yaml    # YAML
mcp2cli weather resource api-docs -o raw             # plain text
```

`-o json` is always script-safe: output goes to `stdout`, diagnostics go to `stderr`, exit codes are stable.

---

## Arguments

`mcp2cli` reads the tool's schema and generates CLI flags automatically.

```bash
# Named flags (always work)
mcp2cli weather get-forecast --latitude 37.7 --longitude -122.4

# Positional arguments (for required scalar args, in schema order)
mcp2cli weather get-forecast 37.7 -122.4

# Booleans
mcp2cli api update --dry-run
mcp2cli api update --no-dry-run

# Repeated values for arrays
mcp2cli api search --tag cli --tag go --tag mcp

# Structured JSON from a file
mcp2cli api create --payload @data.json

# Or from stdin
cat data.json | mcp2cli api create --payload @-
```

For complex schemas, you can always fall back to raw JSON:

```bash
mcp2cli api complex-tool --input '{"nested": {"key": "value"}}'
```

---

## One-off use (no registration)

You don't have to register a server to use it:

```bash
mcp2cli tools --command 'npx -y @h1deya/mcp-server-weather'
mcp2cli tool --command 'npx -y @h1deya/mcp-server-weather' get-forecast 37.7 -122.4
mcp2cli tool --url https://api.example.com/mcp --bearer-env TOKEN search --query test
```

---

## Exposed commands

When you add a server, `mcp2cli` automatically creates a standalone command for it:

```bash
mcp2cli add weather 'npx -y @h1deya/mcp-server-weather'
# → creates mcp-weather

mcp-weather tools
mcp-weather get-forecast 37.7 -122.4
```

Want a shorter name?

```bash
mcp2cli expose weather --as wea
wea tools
wea get-forecast 37.7 -122.4
```

These are real commands on your `PATH`, so `mcp-<TAB>` works in your shell.

---

## Managing servers

```bash
mcp2cli add weather 'npx -y @h1deya/mcp-server-weather'    # register
mcp2cli ls                                                   # list all
mcp2cli rm weather                                           # remove (cleans up exposed commands too)
```

`ls` output:

```text
weather  npx -y @h1deya/mcp-server-weather
notion   https://mcp.notion.com/mcp
```

Config is saved automatically:

- Global: `~/.config/mcp2cli/config.yaml`
- Per-project: `.mcp2cli.yaml` (use `--local` flag)

---

## Diagnosing problems

```bash
mcp2cli weather doctor
```

```text
CHECK    STATUS  DETAIL
resolve  ok      weather
command  ok      /usr/local/bin/npx
auth     ok      no auth required
connect  ok      initialize handshake succeeded
tools    ok      2 tool(s) available
```

---

## Shell completions

```bash
# bash
echo 'source <(mcp2cli completion bash)' >> ~/.bashrc

# zsh
echo 'source <(mcp2cli completion zsh)' >> ~/.zshrc

# fish
mcp2cli completion fish | source
```

---

## How `mcp2cli` relates to `mcp2py`

[mcp2py](https://github.com/maximerivest/mcp2py) turns MCP servers into **Python modules** — great for notebooks, scripts, and data analysis in Python.

`mcp2cli` turns MCP servers into **shell commands** — great for terminal users, shell scripts, CI pipelines, and anyone who prefers the command line.

They complement each other. If a service has an MCP server:
- use `mcp2py` to call it from Python
- use `mcp2cli` to call it from bash, zsh, fish, PowerShell, or any terminal

---

## Build from source

```bash
git clone https://github.com/MaximeRivest/mcp2cli.git
cd mcp2cli
go build -o mcp2cli ./cmd/mcp2cli
./mcp2cli version
```

---

## Current status

This is an alpha release. What works today:

- ✅ local stdio MCP servers
- ✅ remote HTTP JSON-RPC MCP servers
- ✅ bearer token and custom header auth
- ✅ OAuth login with browser flow and token persistence
- ✅ tools, resources, and prompts
- ✅ schema-driven CLI flags and positional arguments
- ✅ server name as implicit subcommand (`mcp2cli weather tools`)
- ✅ exposed standalone commands (`mcp-weather`, `wea`)
- ✅ interactive shell mode with history and completion
- ✅ terminal elicitation (server-initiated user prompts)
- ✅ metadata cache for fast completions
- ✅ `doctor` diagnostics

Still coming:

- ⬜ sampling (LLM-backed server requests)
- ⬜ SSE / streamable HTTP transport compatibility

---

If `mcp2py` makes MCP feel like a native Python library, `mcp2cli` makes MCP feel like it was built for the terminal from day one.
