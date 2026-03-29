# mcptocli: Turn any MCP server into a CLI

📖 **[Documentation](https://maximerivest.github.io/mcptocli/)**

If [mcp2py](https://github.com/maximerivest/mcp2py) turns an MCP server into a Python module, `mcptocli` turns an MCP server into a command-line tool you can use from any terminal.

## What is MCP?

MCP (Model Context Protocol) is an emerging standard that lets AI tools, APIs, and local apps describe what they can do in a machine-readable way. More and more services are exposing MCP interfaces. When they do, you don't have to learn their custom API — you just point `mcptocli` at them and start using their tools from your terminal.

**You don't need to understand the protocol.** All you need to know is:

> If a service has an MCP server, `mcptocli` lets you use it as if it were a normal command-line tool.

---

## Installation

### macOS and Linux

```bash
curl -fsSL https://raw.githubusercontent.com/MaximeRivest/mcptocli/main/install.sh | sh
```

This downloads the right binary for your platform, installs it, and sets up shell completions — all in one step.

### Windows

```powershell
irm https://raw.githubusercontent.com/MaximeRivest/mcptocli/main/install.ps1 | iex
```

### Verify

Open a **new** terminal and run:

```bash
mcptocli version
```

<details>
<summary>Manual install (if you prefer)</summary>

`mcptocli` is a single binary. Download the right one for your platform from the [releases page](https://github.com/MaximeRivest/mcptocli/releases/latest), make it executable, and put it in your `PATH`.

</details>

---

## Quick start

Two steps: **add** a server once, then **use** it by name.

### Step 1: Add a server

```bash
mcptocli add time 'uvx mcp-server-time'
```

That's it. The second argument is the command to start the server. URLs are detected automatically:

```bash
mcptocli add notion https://mcp.notion.com/mcp --auth oauth
```

### Step 2: Use it

```bash
mcptocli time tools
```

```text
Tools (2):

  convert-time      Convert time between timezones.
  get-current-time  Get current time in a specific timezone.

Inspect:  mcptocli tools time <tool>
Invoke:   mcptocli time <tool> [args...]
```

```bash
mcptocli time get-current-time --timezone 'America/New_York'
```

The server name **is** the command. No `tool` vs `tools` to remember — just the server name followed by what you want to do.

---

## Daily use

Once a server is added, here's everything you need:

```bash
mcptocli time tools                                              # list tools
mcptocli time get-current-time --timezone 'America/New_York'      # call a tool
mcptocli time get-current-time 'Europe/London'                    # positional args work too
mcptocli time resources                                          # list resources
mcptocli time prompts                                            # list prompts
mcptocli time shell                                              # interactive mode
mcptocli time doctor                                             # diagnose problems
```

### Inspect a tool before calling it

```bash
mcptocli time tools get-current-time
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
mcptocli time up                                        # start in background
mcptocli time get-current-time 'America/New_York'        # ~10ms instead of ~2s
mcptocli time get-current-time 'Europe/London'            # instant
mcptocli time down                                       # stop when done
```

### Share one server across multiple clients

```bash
mcptocli time up --share                                # start in HTTP mode
mcptocli time get-current-time 'America/New_York'        # mcptocli uses it automatically
# Other MCP clients (Claude Desktop, notebooks) can connect to the same server
mcptocli time down                                       # stop when done
```

### Interactive shell

```bash
mcptocli time shell
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
mcptocli add acme https://api.acme.dev/mcp --bearer-env ACME_TOKEN
mcptocli acme tools
mcptocli acme search --query invoices
```

### With OAuth (browser login)

```bash
mcptocli add notion https://mcp.notion.com/mcp --auth oauth
mcptocli notion tools
# Browser opens automatically the first time
```

---

## Output formats

```bash
mcptocli time get-current-time 'America/New_York'            # human-readable (default)
mcptocli time get-current-time 'America/New_York' -o json     # exact JSON for scripts
mcptocli time get-current-time 'America/New_York' -o yaml     # YAML
```

`-o json` is always script-safe: output goes to `stdout`, diagnostics go to `stderr`, exit codes are stable.

---

## Arguments

`mcptocli` reads the tool's schema and generates CLI flags automatically.

```bash
# Named flags (always work)
mcptocli time get-current-time --timezone 'America/New_York'

# Positional arguments (for required scalar args, in schema order)
mcptocli time get-current-time 'America/New_York'

# Booleans
mcptocli api update --dry-run
mcptocli api update --no-dry-run

# Repeated values for arrays
mcptocli api search --tag cli --tag go --tag mcp

# Structured JSON from a file
mcptocli api create --payload @data.json

# Or from stdin
cat data.json | mcptocli api create --payload @-
```

For complex schemas, you can always fall back to raw JSON:

```bash
mcptocli api complex-tool --input '{"nested": {"key": "value"}}'
```

---

## One-off use (no registration)

You don't have to register a server to use it:

```bash
mcptocli tools --command 'uvx mcp-server-time'
mcptocli tool --command 'uvx mcp-server-time' get-current-time 'America/New_York'
mcptocli tool --url https://api.example.com/mcp --bearer-env TOKEN search --query test
```

---

## Exposed commands

When you add a server, `mcptocli` automatically creates a standalone command for it:

```bash
mcptocli add time 'uvx mcp-server-time'
# → creates mcp-time

mcp-time tools
mcp-time get-current-time 'America/New_York'
```

Want a shorter name?

```bash
mcptocli expose time --as t
t tools
t get-current-time 'America/New_York'
```

Remove an exposed command:

```bash
mcptocli expose --remove time
```

These are real commands on your `PATH`, so `mcp-<TAB>` works in your shell.

---

## Managing servers

```bash
mcptocli add time 'uvx mcp-server-time'                        # save
mcptocli ls                                                     # list all
mcptocli rm time                                                # remove (cleans up exposed commands too)
```

`ls` output:

```text
time (up)  uvx mcp-server-time
notion     https://mcp.notion.com/mcp
```

Config is saved automatically:

- Global: `~/.config/mcptocli/config.yaml`
- Per-project: `.mcptocli.yaml` (use `--local` flag)

---

## Diagnosing problems

```bash
mcptocli time doctor
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
echo 'source <(mcptocli completion bash)' >> ~/.bashrc

# zsh
echo 'source <(mcptocli completion zsh)' >> ~/.zshrc

# fish
mcptocli completion fish | source
```

---

## How `mcptocli` relates to `mcp2py`

[mcp2py](https://github.com/maximerivest/mcp2py) turns MCP servers into **Python modules** — great for notebooks, scripts, and data analysis in Python.

`mcptocli` turns MCP servers into **shell commands** — great for terminal users, shell scripts, CI pipelines, and anyone who prefers the command line.

They complement each other. If a service has an MCP server:
- use `mcp2py` to call it from Python
- use `mcptocli` to call it from bash, zsh, fish, PowerShell, or any terminal

---

## Build from source

```bash
git clone https://github.com/MaximeRivest/mcptocli.git
cd mcptocli
go build -o mcptocli ./cmd/mcptocli
./mcptocli version
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
- ✅ server name as implicit subcommand (`mcptocli time tools`)
- ✅ background daemon for instant responses (`mcptocli time up`)
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

If `mcp2py` makes MCP feel like a native Python library, `mcptocli` makes MCP feel like it was built for the terminal from day one.
