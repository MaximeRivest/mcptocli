# mcp2cli: Turn any MCP server into a CLI

> Status: implemented alpha. `mcp2cli` is written in Go and already supports local stdio servers, remote HTTP JSON-RPC servers, bearer auth, OAuth login, tools, resources, prompts, shell mode, exposed commands, and cache-backed completions. Sampling is still pending.

If `mcp2py` turns an MCP server into a Python module, `mcp2cli` should turn an MCP server into a delightful command-line tool.

MCP servers expose **tools**, **resources**, and **prompts**. `mcp2cli` should map them into shell-native primitives:

- 🔧 **Tools** → commands
- 📦 **Resources** → inspectable values
- 📝 **Prompts** → rendered text
- 🔐 **Auth** → browser login or token-based flows when needed
- 💬 **Elicitation** → terminal prompts when the server asks the user something
- 🤖 **Sampling** → optional LLM-backed responses when a server requests them

The goal is simple: if a server speaks MCP, you should be able to use it from bash, zsh, fish, CI, Makefiles, and shell pipelines without writing a custom client.

## Current status

Implemented today:

- local stdio MCP servers
- remote HTTP JSON-RPC MCP servers
- bearer env auth and custom headers
- OAuth login with token persistence
- tools, resources, and prompts
- schema-driven tool and prompt arguments
- exposed commands like `mcp-weather` and `wea`
- interactive shell mode
- metadata-backed completions
- terminal elicitation

Still pending:

- sampling
- additional remote compatibility layers like SSE / streamable HTTP

## Install / Build

```bash
go build -o bin/mcp2cli ./cmd/mcp2cli
./bin/mcp2cli version
```

Or during development:

```bash
go test ./...
go run ./cmd/mcp2cli version
```

## Why this exists

Today, using an MCP server from the shell often means one of these:

- writing JSON-RPC by hand
- learning transport details before doing anything useful
- manually handling OAuth or bearer tokens
- reading JSON Schema just to know which flags to pass
- getting output that is either too raw for humans or too pretty for scripts

`mcp2cli` should feel like the CLI that server authors would have built themselves if they cared deeply about terminal UX.

## Quick Start

### 1. Use a server immediately

No registration should be required for the happy path.

```bash
mcp2cli tool --command 'npx -y @h1deya/mcp-server-weather' get-alerts --state CA
```

And for a remote server:

```bash
mcp2cli tool --url https://mcp.notion.com/mcp --auth oauth notion-get-self
```

### 2. Register a server you use often

```bash
mcp2cli add weather --command 'npx -y @h1deya/mcp-server-weather'
```

### 3. Expose it as its own CLI

```bash
mcp2cli expose weather
mcp-weather tools
mcp-weather get-alerts --state CA
```

Optional shorter alias:

```bash
mcp2cli expose weather --as wea
wea get-forecast --latitude 37.7749 --longitude -122.4194
```

Aliases should be normalized to kebab-case, unique across all exposed servers, and `--as` should be treated as the full exposed command name. If `--as` is omitted, the default exposed command should be `mcp-<server>`.

### 4. Discover what it can do

```bash
mcp2cli tools weather
```

Example output:

```text
get-alerts      Get active weather alerts for a US state
get-forecast    Get forecast for a latitude/longitude pair
```

### 5. Call a tool

```bash
mcp2cli tool weather get-alerts --state CA
mcp2cli tool weather get-forecast --latitude 37.7749 --longitude -122.4194 -o json | jq
```

### 6. Use a remote server with OAuth

```bash
mcp2cli add notion --url https://mcp.notion.com/mcp --auth oauth
mcp2cli login notion
mcp2cli tool notion notion-get-self
```

### 6.5 Use resources and prompts

```bash
mcp2cli resources weather
mcp2cli resource weather api-docs
mcp2cli prompts weather
mcp2cli prompt weather review-code --code 'x <- 1' --focus api
```

### 7. Explore interactively

```bash
mcp2cli shell weather
```

```text
weather> tools
weather> get-alerts --state CA
weather> get-forecast --latitude 37.7749 --longitude -122.4194
weather> exit
```

## Philosophy

### Delightful defaults

The first successful command should be easy:

- local stdio servers and remote HTTP servers both work
- transport is mostly invisible
- OAuth can open a browser automatically
- terminal prompts appear if the server needs user input
- sensible human-readable output is the default

### Human-first, machine-always

By default, output should be nice to read.

When you need scripting, `-o json` should always give stable machine-readable output.

```bash
mcp2cli tool weather get-alerts --state CA -o json | jq '.[].headline'
```

### Progressive disclosure

Beginners should be able to start with one command.

Power users should still get full control:

- custom headers
- bearer tokens from env
- explicit transport settings
- raw JSON output
- debug logs
- project-local config

### Runtime, not generator-first

The primary experience should be dynamic and immediate.

Point `mcp2cli` at an MCP server and use it right away.
No code generation should be required for the common path.

## Mental model

Every command operates on a **server reference**. A server reference can be:

| Type | Example | Use case |
| --- | --- | --- |
| Registered name | `weather` | Daily use inside `mcp2cli` |
| Exposed CLI | `mcp-weather`, `wea` | Feels like a native CLI for one server |
| Local command | `--command 'npx -y @modelcontextprotocol/server-filesystem /home/maxime'` | One-off local server |
| Remote URL | `--url https://mcp.example.com/mcp` | One-off or registered remote server |

This means the same UX should work whether the server is:

- a local Node/Python/Rust process over stdio
- a remote MCP server over HTTP JSON-RPC today, with SSE/streamable HTTP compatibility to come
- a registered alias in config
- an exposed command on your `PATH`

## Core commands

The CLI should revolve around a small, memorable set of commands:

| Command | Purpose |
| --- | --- |
| `mcp2cli add` | Register a server |
| `mcp2cli ls` | List registered servers |
| `mcp2cli rm` | Remove a registered server |
| `mcp2cli expose` | Create an exposed command for a server |
| `mcp2cli unexpose` | Remove an exposed command |
| `mcp2cli login` | Trigger auth ahead of time |
| `mcp2cli tools` | List tools or inspect one tool |
| `mcp2cli tool` | Invoke a tool |
| `mcp2cli resources` | List resources or inspect one resource |
| `mcp2cli resource` | Read a resource |
| `mcp2cli prompts` | List prompts or inspect one prompt |
| `mcp2cli prompt` | Render a prompt |
| `mcp2cli shell` | Open an interactive MCP shell |
| `mcp2cli completion` | Generate shell completions |
| `mcp2cli doctor` | Diagnose connection/auth/config issues |

## Server-specific CLIs

A registered server should be exposable as a real command on your `PATH`.

```bash
mcp2cli expose weather
```

That should create a command like:

```bash
mcp-weather
```

And optional aliases should also be supported:

```bash
mcp2cli expose weather --as wea
wea
```

This is what makes `mcp-<TAB>` feel magical: the commands are real, so your shell can discover them naturally.

Once exposed, the server should feel like its own CLI:

```bash
mcp-weather tools
mcp-weather get-forecast --latitude 37.7749 --longitude -122.4194
mcp-weather prompt review-code --code @main.go
```

Implementation-wise, these should be lightweight shims, symlinks, or wrappers back to the main `mcp2cli` binary rather than generated standalone CLIs.

Alias rules should be simple:

- `mcp2cli expose weather` → `mcp-weather`
- `mcp2cli expose weather --as wea` → `wea`
- `--as` takes the full exposed command name
- omitting `--as` uses the default `mcp-<server>` name
- aliases are unique across all servers
- multiple aliases may point to the same server

A few meta commands should still be reserved in shim mode:

- `tools`
- `resources`
- `prompts`
- `prompt`
- `tool`
- `shell`
- `doctor`

So daily use can be short:

```bash
mcp-weather get-forecast --latitude 37.7749 --longitude -122.4194
```

And the explicit escape hatch remains available when a tool name is ambiguous:

```bash
mcp-weather tool get-forecast --latitude 37.7749 --longitude -122.4194
```

## The delightful API

This is the most important part of the project.

### 1. Tools → commands

A tool should feel like a normal shell command.

If the MCP server exposes:

```json
{
  "name": "searchFiles",
  "description": "Search for files matching a pattern",
  "inputSchema": {
    "type": "object",
    "properties": {
      "pattern": {"type": "string", "description": "Glob pattern"},
      "maxResults": {"type": "integer", "default": 100}
    },
    "required": ["pattern"]
  }
}
```

Then `mcp2cli` should make that feel like any of these:

```bash
mcp2cli tool files search-files '*.go' --max-results 50
mcp2cli tool files search-files --pattern '*.go' --max-results 50
mcp-files search-files '*.go' --max-results 50
```

### 2. Names → kebab-case

CLI names should follow shell conventions.

- MCP `searchFiles` → CLI `search-files`
- MCP `getWeather` → CLI `get-weather`
- MCP `notion_get_self` → CLI `notion-get-self`

This should apply consistently to:

- tool names
- prompt names
- argument names
- generated completion entries

### 3. Arguments → positionals when obvious, flags always available

Rules for a delightful tool-call syntax:

- required scalar arguments may be positional, in schema order
- every argument must also be available as a named flag
- optional arguments should be flags
- booleans should support `--flag` / `--no-flag`
- arrays should support repeated flags
- objects should accept JSON strings or `@file.json`
- `@-` should mean “read JSON/text from stdin”

Examples:

```bash
# Positional + flag
mcp2cli tool files read-file README.md
mcp2cli tool files read-file --path README.md

# Repeated flags for arrays
mcp2cli tool api search --tag cli --tag go --tag mcp

# Structured JSON from file
mcp2cli tool api create-issue --payload @issue.json

# Structured JSON from stdin
cat issue.json | mcp2cli tool api create-issue --payload @-
```

### 4. Discovery should be first-class

You should not need to leave the terminal to understand a server.

```bash
mcp2cli tools weather
mcp2cli tools weather get-forecast
mcp2cli resources notion
mcp2cli prompts writer
```

Inspecting a single tool should show:

- human description
- usage line
- argument list
- which arguments are required
- defaults
- output hints when known

Example:

```text
$ mcp2cli tools weather get-forecast

NAME
  get-forecast - Get weather forecast for a location

USAGE
  mcp2cli tool weather get-forecast --latitude <float> --longitude <float>

ARGS
  --latitude float     Required. Latitude of the location.
  --longitude float    Required. Longitude of the location.
```

### 5. Output → pretty by default, exact on demand

Default output should be optimized for humans.

But scripts must always be able to opt into exact data.

Proposed output modes:

- `-o auto` → default, human-friendly
- `-o json` → exact JSON payload
- `-o yaml` → readable structured output
- `-o raw` → plain text/bytes when possible
- `-o table` → tabular rendering when shape allows it

Examples:

```bash
mcp2cli tool weather get-alerts --state CA
mcp2cli tool weather get-alerts --state CA -o json
mcp2cli resource api api-docs -o raw
```

Suggested default behavior:

- strings print as strings
- arrays of uniform objects render nicely as tables when safe
- nested objects pretty-print cleanly
- raw resource contents can pass straight through

### 6. Resources → easy to inspect, easy to read

Resources are not tools, so they should not feel like tools.

They should have dedicated commands:

```bash
mcp2cli resources api
mcp2cli resource api api-docs -o raw
```

If resources have opaque URIs, `mcp2cli` should still present a friendly alias when possible, while preserving the original URI in machine-readable output.

### 7. Prompts → render text, don’t overcomplicate it

Prompts should feel like text templates.

```bash
mcp2cli prompts writer
mcp2cli prompt writer review-code --code @main.go --focus api-design
```

By default, `prompt` should print the rendered text.
If the server returns prompt metadata, `-o json` should expose it.

### 8. Auth should mostly disappear

For remote servers:

- `--auth oauth` should open a browser when needed
- tokens should be cached securely
- `--bearer-env TOKEN_NAME` should pull bearer tokens from environment variables
- `--header 'X-API-Key: ...'` should support custom headers

Examples:

```bash
mcp2cli add notion --url https://mcp.notion.com/mcp --auth oauth
mcp2cli add acme --url https://api.acme.dev/mcp --bearer-env ACME_API_TOKEN
```

And then just:

```bash
mcp2cli tool notion notion-get-self
mcp2cli tool acme search --query invoices
```

### 9. Errors should be short, actionable, and kind

Error messages should help users recover immediately.

```text
$ mcp2cli tool weather get-alerts
error: missing required argument: state
hint: run `mcp2cli tools weather get-alerts`
```

```text
$ mcp2cli tool notion notion-search --query roadmap
error: authentication required
hint: run `mcp2cli login notion`
```

For scripts, exit codes should be stable by category:

- usage/validation error
- auth error
- tool/server error
- transport/connectivity error
- internal error

### 10. Shell mode should be joyful

A lot of MCP exploration is interactive. A shell mode should keep the connection open and provide a lightweight REPL:

```bash
mcp2cli shell weather
```

```text
weather> tools
weather> get-alerts --state CA
weather> get-forecast --latitude 37.7749 --longitude -122.4194
```

This mode should support:

- history
- completion
- inline help
- persistent connection/auth during the session
- easy switching between `auto` and `json` output

## Example workflows

### Local filesystem server

```bash
mcp2cli add files --command 'npx -y @modelcontextprotocol/server-filesystem /home/maxime'
mcp2cli tools files
mcp2cli tool files list-directory /home/maxime
mcp2cli tool files read-file /home/maxime/todo.txt -o raw
```

### Remote server with OAuth

```bash
mcp2cli add notion --url https://mcp.notion.com/mcp --auth oauth
mcp2cli login notion
mcp2cli tool notion notion-get-self
```

### Shell pipeline

```bash
mcp2cli tool weather get-alerts --state CA -o json | jq 'length'
```

### Prompt rendering

```bash
mcp2cli prompt writer summarize-article --title 'MCP for CLI users' --text @article.md
```

### Project automation

```bash
#!/usr/bin/env bash
set -euo pipefail

alerts_json=$(mcp2cli tool weather get-alerts --state CA -o json)
count=$(jq 'length' <<< "$alerts_json")

echo "Found $count alerts"
```

## Config

`mcp2cli` should support both global and project-local config.

Suggested locations:

- global: `~/.config/mcp2cli/config.yaml`
- local: `.mcp2cli.yaml`

Example:

```yaml
version: 1

defaults:
  output: auto

servers:
  weather:
    command: npx -y @h1deya/mcp-server-weather
    expose:
      - mcp-weather
      - wea

  files:
    command: npx -y @modelcontextprotocol/server-filesystem /home/maxime
    roots:
      - /home/maxime
    expose:
      - mcp-files

  notion:
    url: https://mcp.notion.com/mcp
    auth: oauth
    expose:
      - mcp-notion

  acme:
    url: https://api.acme.dev/mcp
    bearer_env: ACME_API_TOKEN
    headers:
      X-Team: platform
```

Nice-to-have behavior:

- local config overrides global config
- `mcp2cli add --local` writes to `.mcp2cli.yaml`
- `mcp2cli add` defaults to global config
- `mcp2cli expose weather` creates `mcp-weather`
- `mcp2cli expose weather --as wea` creates `wea`
- `mcp2cli unexpose weather --as wea` removes that alias
- `mcp2cli ls` shows where each server is defined and which exposed command names point to it

For auth storage, `mcp2cli` uses reliable file-backed token storage by default. System keyring support is available by setting:

```bash
export MCP2CLI_USE_SYSTEM_KEYRING=1
```

## Shell integration

Tab completion is a big part of the delight.

Current completion behavior is backed by the metadata cache, so registered and exposed servers feel fast once they have been inspected at least once.

```bash
mcp2cli completion bash
mcp2cli completion zsh
mcp2cli completion fish
```

And exposed server CLIs should live in a directory like:

```text
~/.local/share/mcp2cli/bin
```

so users can add it to `PATH` and get:

```bash
mcp-<TAB>
```

Completion should eventually know about:

- registered server names
- exposed commands, including default `mcp-*` names and custom aliases like `wea`
- tool names
- tool flags derived from JSON Schema
- enum values when available

That means all of these should feel great:

```bash
mcp2cli tool weather <TAB>
mcp2cli tool weather get-forecast --<TAB>
mcp-<TAB>
mcp-weather get-forecast --<TAB>
wea get-forecast --<TAB>
```

## Sampling and elicitation

Some MCP servers ask the client to do more than plain tool calls.

`mcp2cli` currently handles:

- **elicitation** → ask the user in the terminal

Still pending:

- **sampling** → optionally call a configured LLM and return the answer to the server

Current elicitation behavior:

- interactive terminals are prompted directly
- non-interactive stdin fails clearly with an interactive-input error
- prompts are printed to `stderr`, keeping `stdout` clean for command output
- current support is focused on local stdio sessions and other transports that can carry server-initiated requests in-band; the simple HTTP JSON-RPC path does not yet provide full elicitation support

## Why Go

Go is a strong fit for this project because we want:

- a single fast binary
- excellent cross-platform distribution
- clean concurrency for long-lived sessions and streaming transports
- strong JSON and HTTP support
- easy installation for shell users

The implementation language should disappear behind the UX.
The product should feel simple, immediate, and dependable.

## Initial scope

The first release should focus on:

1. stdio and HTTP-based MCP transports
2. tool discovery and invocation
3. resources and prompts
4. auth support for OAuth and bearer tokens
5. human + JSON output modes
6. config, server registry, and exposed server CLIs
7. shell mode
8. completion and schema-driven help
9. excellent error messages

## Design principles

1. **Delightful defaults**
2. **No ceiling for power users**
3. **Human first, machine always**
4. **Progressive disclosure**
5. **Fast path for discovery**
6. **Stable path for automation**
7. **Shell-native naming and output**
8. **Kind, actionable errors**
9. **One binary, many servers**
10. **Feel dynamic, not ceremony-heavy**

---

If `mcp2py` makes MCP feel like a native Python library, `mcp2cli` should make MCP feel like it was built for the terminal from day one.
