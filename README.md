# mcp2cli: Turn any MCP server into a CLI

If [mcp2py](https://github.com/maximerivest/mcp2py) turns an MCP server into a Python module, `mcp2cli` turns an MCP server into a command-line tool you can use from any terminal.

## What is MCP?

MCP (Model Context Protocol) is an emerging standard that lets AI tools, APIs, and local apps describe what they can do in a machine-readable way. More and more services are exposing MCP interfaces. When they do, you don't have to learn their custom API — you just point `mcp2cli` at them and start using their tools from your terminal.

**You don't need to understand the protocol.** All you need to know is:

> If a service has an MCP server, `mcp2cli` lets you use it as if it were a normal command-line tool.

---

## Installation

### macOS and Linux

```bash
curl -fsSL https://raw.githubusercontent.com/MaximeRivest/mcp2cli/main/install.sh | sh
```

This downloads the right binary for your platform, installs it, and sets up shell completions — all in one step.

### Windows

```powershell
irm https://raw.githubusercontent.com/MaximeRivest/mcp2cli/main/install.ps1 | iex
```

### Verify

Open a **new** terminal and run:

```bash
mcp2cli version
```

<details>
<summary>Manual install (if you prefer)</summary>

`mcp2cli` is a single binary. Download the right one for your platform from the [releases page](https://github.com/MaximeRivest/mcp2cli/releases/latest), make it executable, and put it in your `PATH`.

</details>

---

## Quick start

Two steps: **add** a server once, then **use** it by name.

### Step 1: Add a server

```bash
mcp2cli add time 'npx -y @modelcontextprotocol/server-time'
```

That's it. The second argument is the command to start the server. URLs are detected automatically:

```bash
mcp2cli add notion https://mcp.notion.com/mcp --auth oauth
```

### Step 2: Use it

```bash
mcp2cli time tools
```

```text
Tools (2):

  convert-time      Convert time between timezones.
  get-current-time  Get current time in a specific timezone.

Inspect:  mcp2cli tools time <tool>
Invoke:   mcp2cli time <tool> [args...]
```

```bash
mcp2cli time get-current-time --timezone 'America/New_York'
```

The server name **is** the command. No `tool` vs `tools` to remember — just the server name followed by what you want to do.

---

## Daily use

Once a server is added, here's everything you need:

```bash
mcp2cli time tools                                              # list tools
mcp2cli time get-current-time --timezone 'America/New_York'      # call a tool
mcp2cli time get-current-time 'Europe/London'                    # positional args work too
mcp2cli time resources                                          # list resources
mcp2cli time prompts                                            # list prompts
mcp2cli time shell                                              # interactive mode
mcp2cli time doctor                                             # diagnose problems
```

### Inspect a tool before calling it

```bash
mcp2cli time tools get-current-time
```

```text
NAME
  get-current-time - Get current time in a specific timezone

USAGE
  time get-current-time --timezone <string>

ARGS
  --timezone string  Required. IANA timezone name (e.g., 'America/New_York', 'Europe/London').
```

### Keep the server running for instant responses

```bash
mcp2cli time up                                        # start in background
mcp2cli time get-current-time 'America/New_York'        # ~10ms instead of ~2s
mcp2cli time get-current-time 'Europe/London'            # instant
mcp2cli time down                                       # stop when done
```

### Share one server across multiple clients

```bash
mcp2cli time up --share                                # start in HTTP mode
mcp2cli time get-current-time 'America/New_York'        # mcp2cli uses it automatically
# Other MCP clients (Claude Desktop, notebooks) can connect to the same server
mcp2cli time down                                       # stop when done
```

### Interactive shell

```bash
mcp2cli time shell
```

```text
time> tools
time> get-current-time 'America/New_York'
time> convert-time --source-timezone 'America/New_York' --time '14:30' --target-timezone 'Europe/London'
time> set output json
time> get-current-time 'Europe/London'
time> exit
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
mcp2cli time get-current-time 'America/New_York'            # human-readable (default)
mcp2cli time get-current-time 'America/New_York' -o json     # exact JSON for scripts
mcp2cli time get-current-time 'America/New_York' -o yaml     # YAML
```

`-o json` is always script-safe: output goes to `stdout`, diagnostics go to `stderr`, exit codes are stable.

---

## Arguments

`mcp2cli` reads the tool's schema and generates CLI flags automatically.

```bash
# Named flags (always work)
mcp2cli time get-current-time --timezone 'America/New_York'

# Positional arguments (for required scalar args, in schema order)
mcp2cli time get-current-time 'America/New_York'

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
mcp2cli tools --command 'uvx mcp-server-time'
mcp2cli tool --command 'uvx mcp-server-time' get-current-time 'America/New_York'
mcp2cli tool --url https://api.example.com/mcp --bearer-env TOKEN search --query test
```

---

## Exposed commands

When you add a server, `mcp2cli` automatically creates a standalone command for it:

```bash
mcp2cli add time 'uvx mcp-server-time'
# → creates mcp-time

mcp-time tools
mcp-time get-current-time 'America/New_York'
```

Want a shorter name?

```bash
mcp2cli expose time --as t
t tools
t get-current-time 'America/New_York'
```

Remove an exposed command:

```bash
mcp2cli expose --remove time
```

These are real commands on your `PATH`, so `mcp-<TAB>` works in your shell.

---

## Managing servers

```bash
mcp2cli add time 'npx -y @modelcontextprotocol/server-time'   # save
mcp2cli ls                                                     # list all
mcp2cli rm time                                                # remove (cleans up exposed commands too)
```

`ls` output:

```text
time (up)  uvx mcp-server-time
notion     https://mcp.notion.com/mcp
```

Config is saved automatically:

- Global: `~/.config/mcp2cli/config.yaml`
- Per-project: `.mcp2cli.yaml` (use `--local` flag)

---

## Diagnosing problems

```bash
mcp2cli time doctor
```

```text
CHECK    STATUS  DETAIL
resolve  ok      time
command  ok      /home/maxime/.local/bin/uvx
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
- ✅ server name as implicit subcommand (`mcp2cli time tools`)
- ✅ background daemon for instant responses (`mcp2cli time up`)
- ✅ exposed standalone commands (`mcp-time`, `t`)
- ✅ interactive shell mode with history and completion
- ✅ terminal elicitation (server-initiated user prompts)
- ✅ Streamable HTTP (SSE) transport
- ✅ metadata cache for fast completions
- ✅ `doctor` diagnostics

Still coming:

- ✅ shared HTTP daemon mode (`up --share`)
- ⬜ sampling (LLM-backed server requests)

---

If `mcp2py` makes MCP feel like a native Python library, `mcp2cli` makes MCP feel like it was built for the terminal from day one.
