# mcp2cli Implementation Plan

## Overview

Build `mcp2cli` as a single Go binary that turns any MCP server into a delightful command-line tool.

If `mcp2py` makes MCP feel like a native Python library, `mcp2cli` should make MCP feel native to the shell:

- easy to discover
- easy to call
- pleasant for humans
- stable for scripts
- fast to install
- easy to debug

This plan translates the `README.md` UX into an implementation roadmap with concrete architecture, packages, phases, and acceptance criteria.

---

## Product goals

### Primary goals

1. **Zero-config direct use**
   - Local servers should work with `--command`
   - Remote servers should work with `--url`
   - Registration should be optional, not required

2. **Delightful schema-driven CLI**
   - MCP tools should feel like normal shell commands
   - Tool input schemas should become intuitive flags and positionals
   - Discovery should be great directly in the terminal

3. **Human-first, machine-always output**
   - Default output should be readable
   - `-o json` should always be script-safe and predictable
   - Diagnostics must go to `stderr`, never pollute `stdout`

4. **First-class auth and interactivity**
   - OAuth should be easy
   - Bearer tokens and custom headers should be simple
   - Elicitation and sampling should work when servers request them

5. **Single binary, cross-platform**
   - Linux, macOS, Windows
   - Homebrew / Scoop / release binaries
   - Minimal runtime dependencies beyond the target MCP server itself

### Secondary goals

1. Great shell completions
2. Interactive shell mode
3. Metadata caching for speed
4. Server-specific exposed commands that feel native
5. Excellent error messages and diagnostics
6. Clean internal architecture that can later power a reusable Go library

---

## Non-goals for v1

1. Generating standalone compiled CLIs per server
   - Lightweight exposed shims are in scope; fully separate generated binaries are not.
2. Full TUI application
3. Background daemon or always-on service
4. Plugin system
5. Supporting every obscure JSON Schema shape with beautiful flags
   - Complex schemas will have a raw JSON fallback

---

## Definition of done

`mcp2cli` v1 is done when all of the following work end-to-end:

```bash
# Direct local use
mcp2cli tools --command 'npx -y @h1deya/mcp-server-weather'
mcp2cli tool --command 'npx -y @h1deya/mcp-server-weather' get-alerts --state CA

# Registered local use
mcp2cli add weather --command 'npx -y @h1deya/mcp-server-weather'
mcp2cli tools weather
mcp2cli tool weather get-forecast --latitude 37.7749 --longitude -122.4194 -o json

# Exposed CLI use
mcp2cli expose weather
mcp-weather tools
mcp-weather get-forecast --latitude 37.7749 --longitude -122.4194
mcp2cli expose weather --as wea
wea get-alerts --state CA

# Remote use
mcp2cli add notion --url https://mcp.notion.com/mcp --auth oauth
mcp2cli login notion
mcp2cli tool notion notion-get-self

# Resources and prompts
mcp2cli resources weather
mcp2cli prompts weather
mcp2cli prompt weather some-prompt --topic mcp

# Shell mode
mcp2cli shell weather

# Scripting guarantees
mcp2cli tool weather get-alerts --state CA -o json | jq
```

And all of the following are true:

- `stdout` contains only command result output
- `stderr` contains warnings, progress, hints, and errors
- exit codes are stable by category
- exposed commands, including default `mcp-*` names and custom aliases, work on Linux, macOS, and Windows
- shell completion works for bash, zsh, and fish
- tests pass on Linux, macOS, and Windows

---

## Core product decisions

### 1. Runtime-first, not codegen-first

The main UX is dynamic:

- connect to server
- introspect metadata
- present tools/resources/prompts immediately
- invoke with schema-driven parsing

Code generation is explicitly deferred.

### 2. Multicall shims for exposed commands

`mcp2cli` should also support being invoked through lightweight shims such as:

- `mcp-weather`
- `wea`
- `mcp-notion`

These should resolve back to the main binary and bind a registered server automatically based on `argv[0]`.

That gives us:

- native-feeling per-server CLIs
- real shell discovery via `mcp-<TAB>`
- zero duplication of implementation
- one binary to install and update

### 3. Static top-level CLI, dynamic sub-arguments

Top-level commands are static:

- `add`
- `ls`
- `rm`
- `expose`
- `unexpose`
- `login`
- `tools`
- `tool`
- `resources`
- `resource`
- `prompts`
- `prompt`
- `shell`
- `completion`
- `doctor`
- `version`

But tool-specific arguments are dynamic and derived at runtime from MCP metadata.

### 4. Custom runtime argument parser for tool calls

Cobra will handle static commands and known flags.
Tool-specific arguments will be parsed by `mcp2cli` after it introspects the selected MCP tool.

This is necessary because:

- tool names are dynamic
- tool flags are dynamic
- one-off `--command` and `--url` flows cannot pre-register subcommands
- completions need schema-aware suggestions

### 5. Plain YAML config, minimal magic

Use a simple YAML config format with explicit merge rules.
Avoid Viper-style hidden precedence and implicit behavior.

### 6. Thin in-house MCP client layer

Implement a focused internal MCP client optimized for CLI UX:

- JSON-RPC framing
- stdio transport
- HTTP-based transport
- session lifecycle
- callbacks for elicitation/sampling

This gives full control over:

- connection lifecycle
- auth integration
- metadata caching
- progress/error surfacing
- future shell/repl support

---

## Technical stack

### Language and toolchain

- **Go**: `>= 1.24`
- **Build/release**: `goreleaser`
- **Linting**: `golangci-lint`
- **Testing**: `go test`, `testify`, golden tests, PTY tests for shell mode

### Proposed dependencies

Use as few as possible.

- **CLI**: `github.com/spf13/cobra`
- **YAML**: `gopkg.in/yaml.v3`
- **XDG paths**: `github.com/adrg/xdg`
- **Keyring**: `github.com/99designs/keyring`
- **Browser open**: `github.com/pkg/browser`
- **Tables**: `github.com/jedib0t/go-pretty/v6/table`
- **Shell/REPL**: `github.com/chzyer/readline`
- **JSON Schema validation**: `github.com/santhosh-tekuri/jsonschema/v6`
- **Color/styling**: `github.com/charmbracelet/lipgloss` (optional, subtle use only)
- **Testing helpers**: `github.com/stretchr/testify`, `github.com/creack/pty`

Prefer the Go standard library for:

- HTTP
- process management
- JSON
- context/cancellation
- TLS
- OAuth callback server plumbing

---

## Repository layout

```text
cmd/
  mcp2cli/
    main.go

internal/
  app/                 # app wiring, dependency container, shared runtime state
  cli/                 # cobra commands and static flag definitions
  config/              # global/local config loading, merging, writing
  serverref/           # resolve named servers, --command, --url into a normalized server config
  expose/              # create/remove/list exposed shims and alias bindings
  cache/               # metadata cache, token fallback file storage
  auth/                # bearer env, headers, OAuth flow, keyring integration
  mcp/
    jsonrpc/           # message framing, ids, request/response correlation
    transport/
      stdio/
      http/
    client/            # initialize, list tools/resources/prompts, call tool, get prompt, read resource
    session/           # high-level lifecycle for one command or shell session
    types/             # MCP request/response models used internally
  schema/
    inspect/           # derive ArgSpec from JSON Schema or prompt args
    validate/          # JSON Schema validation wrappers
    ordered/           # preserve property order for positional args
  invoke/              # parse runtime args, build input payloads, execute operations
  render/              # auto/json/yaml/raw/table renderers
  shell/               # REPL loop, history, completion bridge
  complete/            # dynamic completion helpers for cobra + shell mode
  doctor/              # diagnostics and health checks
  exitcode/            # stable exit code mapping
  ui/                  # stderr helpers, hints, warnings, subtle styling

testdata/
  servers/             # deterministic MCP fixture servers
  golden/              # expected CLI output snapshots

scripts/
  test-e2e.sh
  smoke.sh

.github/workflows/
  ci.yml
  release.yml
```

---

## CLI contract

## Server selection rules

Commands that operate on a server share the same selection model:

- use a registered name
- or provide `--command ...`
- or provide `--url ...`

### Resolution precedence

1. explicit CLI flags (`--command`, `--url`, `--header`, etc.)
2. local config (`.mcp2cli.yaml`)
3. global config (`~/.config/mcp2cli/config.yaml`)
4. built-in defaults

### Accepted patterns

```bash
# registered server
mcp2cli tools weather
mcp2cli tool weather get-alerts --state CA

# direct local
mcp2cli tools --command 'npx -y @h1deya/mcp-server-weather'
mcp2cli tool --command 'npx -y @h1deya/mcp-server-weather' get-alerts --state CA

# direct remote
mcp2cli tools --url https://mcp.example.com/mcp
mcp2cli tool --url https://mcp.example.com/mcp --auth oauth notion-get-self
```

### Common connection flags

These should be available consistently on `tools`, `tool`, `resources`, `resource`, `prompts`, `prompt`, `login`, `shell`, and `doctor`:

- `--command`
- `--url`
- `--transport auto|stdio|http|sse|streamable-http`
- `--header key:value` (repeatable)
- `--auth none|oauth|bearer`
- `--bearer-env NAME`
- `--timeout 30s`
- `--cwd PATH`
- `--env KEY=VALUE` (repeatable)
- `--root PATH` (repeatable)
- `--debug`
- `--trace-rpc`

## Exposed server CLI mode

When the binary is invoked as an exposed command such as `mcp-weather` or `wea`, it should enter a bound-server mode.

Examples:

```bash
mcp-weather tools
mcp-weather get-forecast --latitude 37.7749 --longitude -122.4194
mcp-weather tool get-forecast --latitude 37.7749 --longitude -122.4194
mcp-weather prompt review-code --code @main.go
```

### Resolution rules

1. `argv[0]` like `mcp-weather` or `wea` resolves to the full exposed command name
2. that exposed command name resolves to a registered server
3. remaining args are interpreted in bound-server mode

Alias rules:

- aliases are normalized to kebab-case
- aliases are unique across all registered servers
- multiple aliases may point to the same server
- `--as` means “use this exact command name”
- omitting `--as` means “use the default `mcp-<server>` command name”

### Reserved meta commands in shim mode

The following should remain reserved for metadata and management:

- `tools`
- `resources`
- `resource`
- `prompts`
- `prompt`
- `tool`
- `shell`
- `doctor`

### Tool shorthand in shim mode

If the first non-flag argument is not a reserved meta command, treat it as a tool name shorthand.

So these should be equivalent:

```bash
mcp-weather get-forecast --latitude 37.7749 --longitude -122.4194
mcp-weather tool get-forecast --latitude 37.7749 --longitude -122.4194
```

### Shim storage

Store user-managed shims in a directory like:

- `~/.local/share/mcp2cli/bin` on Linux/macOS
- platform-appropriate equivalent on Windows

Users can add that directory to `PATH` to make `mcp-<TAB>` work naturally.

---

## Config design

### Global and local files

- Global: `~/.config/mcp2cli/config.yaml`
- Local: `.mcp2cli.yaml`

### Example format

```yaml
version: 1

defaults:
  output: auto
  color: auto
  timeout: 30s

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

### Normalized internal model

```go
type ServerConfig struct {
    Name        string
    Source      ConfigSource // global, local, ephemeral
    Transport   string
    Command     string
    URL         string
    CWD         string
    Env         map[string]string
    Roots       []string
    Headers     map[string]string
    AuthMode    string // none, bearer, oauth
    BearerEnv   string
    ExposeAs    []string // full exposed command names, e.g. mcp-weather or wea
    Timeout     time.Duration
    Sampling    SamplingConfig
    Elicitation ElicitationConfig
}
```

### Config behavior

- local config overrides global config for matching server names
- `mcp2cli add --local` writes to `.mcp2cli.yaml`
- `mcp2cli add` defaults to global config
- `mcp2cli expose weather` adds the default exposed name `mcp-weather` and creates that shim
- `mcp2cli expose weather --as wea` adds exposed name `wea` and creates that shim
- `mcp2cli unexpose weather --as wea` removes that alias and shim
- alias uniqueness is validated before writing config or creating shims
- `mcp2cli ls` shows source column (`global`, `local`) plus exposed aliases
- writes preserve unknown fields when possible
- config write operations are atomic

### Command storage rule

For ergonomics, support both:

```yaml
command: npx -y @h1deya/mcp-server-weather
```

and an exact argv form later if needed:

```yaml
argv:
  - npx
  - -y
  - @h1deya/mcp-server-weather
```

Internally normalize to an executable plus argument slice.

---

## Dynamic argument parsing plan

This is the heart of the product.

## Supported schema shapes in v1

### First-class path

Beautiful flags/positionals are supported when the input schema is:

- a top-level object
- with named properties
- mostly scalar/array/object properties
- without deeply nested `oneOf`/`anyOf` ambiguity

### Escape hatch for complex schemas

For complex schemas, always support a raw JSON input path:

```bash
mcp2cli tool api complex-tool --input @payload.json
mcp2cli tool api complex-tool --input @-
```

This ensures every valid MCP tool remains callable even when schema-to-flags mapping is not delightful.

## Argument mapping rules

### Naming

- tool names → `kebab-case`
- argument names → `kebab-case`
- original MCP names retained internally for payload construction

### Positionals

Allow positionals only when all are true:

- argument is required
- argument is scalar (`string`, `number`, `integer`, `boolean` only when unambiguous)
- order is reliable
- schema is simple enough

If order is ambiguous, require named flags.

### Flags

- `--name value`
- `--name=value`
- `--flag` / `--no-flag` for booleans
- repeated flags for arrays: `--tag x --tag y`
- `@file.json` or `@-` for object/array/string payloads

### Property order preservation

Go maps do not preserve JSON object order.
To support intuitive positionals, parse tool schemas into an ordered representation before converting into Go structs.

Order precedence:

1. explicit future extension like `x-cli-order` if present
2. property declaration order preserved from raw schema
3. fallback: disable positional mode for that tool

### Validation flow

1. Parse CLI args into a JSON-like object
2. Apply defaults where appropriate
3. Validate against the original JSON Schema
4. Surface clear validation errors with field names and hints
5. Send validated payload to MCP server

---

## Output rendering plan

## Output modes

- `-o auto` (default)
- `-o json`
- `-o yaml`
- `-o raw`
- `-o table`

## Non-negotiable rules

1. `stdout` is reserved for result output only
2. diagnostics and progress go to `stderr`
3. `-o json` emits valid JSON with no extra decoration
4. `-o raw` passes plain text/bytes through when possible
5. `auto` must never silently drop data

## Auto mode heuristics

### Strings

- print directly
- preserve final newline behavior when sensible

### Flat object arrays

Render as table only if all are true:

- each item is an object
- keys are mostly scalar values
- column count is reasonable
- nested structures are minimal

Else pretty-print as indented JSON or YAML-like structured output.

### Objects

Pretty-print with stable key ordering when possible.

### Binary-like data

- if writing to TTY in `auto`, print a summary and suggest `-o raw > file`
- if writing to pipe/file, allow raw bytes when safe and explicitly requested

## Metadata commands

`tools`, `resources`, and `prompts` should also honor `-o json|yaml|table|auto`.

---

## Error handling and exit codes

## Error design rules

Errors should be:

- short
- actionable
- kind
- specific
- script-safe

### Example style

```text
error: missing required argument: state
hint: run `mcp2cli tools weather get-alerts`
```

```text
error: authentication required
hint: run `mcp2cli login notion`
```

## Exit codes

Proposed stable mapping:

- `0` success
- `2` usage or validation error
- `3` config or server resolution error
- `4` auth error
- `5` connection or transport error
- `6` protocol error
- `7` tool/resource/prompt server-side error
- `8` interactive input required but unavailable
- `10` internal error

Map all internal errors into these categories centrally in `internal/exitcode`.

---

## MCP client architecture

## Core responsibilities

The internal MCP client layer must support:

- initialize handshake
- list tools
- call tool
- list resources
- read resource
- list prompts
- get prompt
- client callbacks for elicitation and sampling
- notifications/progress/logging when useful

## Session model

### One-shot session

Used by normal commands:

1. resolve server
2. connect
3. initialize
4. perform one operation
5. clean shutdown

### Persistent session

Used by `shell`:

1. resolve server
2. connect once
3. initialize once
4. reuse connection for many operations
5. close on exit

## JSON-RPC layer

Implement a transport-agnostic JSON-RPC core with:

- request id generation
- request/response correlation
- notification handling
- cancellation support
- timeout support
- structured tracing with secret redaction

## Stdio transport

### Requirements

- spawn process with context cancellation
- support cwd/env overrides
- support content-length framed messages
- capture stderr separately for debug display
- graceful shutdown on command completion or Ctrl-C

### Acceptance criteria

- works against deterministic local fixture servers
- cleans up child processes reliably
- surfaces executable-not-found errors clearly

## HTTP transport

### Requirements

- support current MCP HTTP transport modes required by target servers
- custom headers
- bearer auth
- OAuth-initialized sessions
- request timeout and retry policy only where safe
- streaming/progress support where applicable

### Compatibility strategy

Implement the modern HTTP transport first, then add compatibility mode(s) if needed for specific servers.
Keep the transport interface small so additional HTTP variants do not affect CLI UX.

**Implementation note:** The current implementation supports the straightforward JSON-RPC-over-HTTP POST path for remote MCP servers. Compatibility layers for SSE and streamable HTTP style transports are still future work.

---

## Auth plan

## Bearer tokens and headers

Support:

- `--header key:value`
- `--bearer-env NAME`
- config-defined headers
- config-defined auth mode

Behavior:

- secrets never appear in `ps` output when avoidable
- secrets are redacted in logs/traces
- env lookup errors include the missing variable name

## OAuth

### Goals

- `mcp2cli login notion`
- browser opens automatically when possible
- fallback to printing URL if browser launch fails
- token stored securely
- subsequent commands reuse cached token

### Implementation

1. start OAuth flow requested by server
2. open system browser using loopback redirect when supported
3. run temporary localhost callback server on random available port
4. exchange code for token
5. store token in OS keyring
6. fall back to `0600` file storage only if keyring is unavailable, with a warning

### Token cache key

Use a stable key derived from:

- server name if registered
- otherwise URL or normalized command fingerprint
- auth mode

## Commands

- `login` → fetch/store auth
- future `logout` can be added later without breaking model

---

## Resources and prompts plan

## Resources

Support:

- `mcp2cli resources <server>`
- `mcp2cli resources <server> <resource>` for inspection
- `mcp2cli resource <server> <resource>` to read contents
- `-o raw|json|yaml|auto`

Implementation notes:

- expose a friendly alias when available
- preserve original URI in machine-readable output
- support resource templates in a second pass once base resources are working

## Prompts

Support:

- `mcp2cli prompts <server>`
- `mcp2cli prompts <server> <prompt>` for inspection
- `mcp2cli prompt <server> <prompt> [args...]`

Prompt arguments are generally simpler than tool schemas and can use the same runtime parser with a smaller adapter.

By default:

- rendered text prints to stdout
- `-o json` returns full structured prompt result

---

## Completion plan

## Shell completions

Generate static completion scripts:

- `mcp2cli completion bash`
- `mcp2cli completion zsh`
- `mcp2cli completion fish`

## Dynamic completion behavior

Completion should know about:

- registered server names
- exposed commands, including default `mcp-*` names and custom aliases
- tool names
- prompt names
- resource names
- tool flags derived from schemas
- enum values when present

For exposed shims, the desired experience is:

```bash
mcp-<TAB>
mcp-weather get-forecast --<TAB>
wea get-forecast --<TAB>
```

## Performance strategy

Dynamic completion cannot connect to remote servers on every tab press.
Use metadata cache with TTL.

### Metadata cache

Store under:

- `~/.cache/mcp2cli/metadata/...`

Cache contents:

- tool list
- tool schemas
- prompt list
- resource list
- fetch timestamp
- server fingerprint

Completion policy:

1. use fresh cache when available
2. refresh in background or on explicit command where feasible
3. fall back gracefully when metadata is unavailable

---

## Shell mode plan

## Goals

`mcp2cli shell <server>` should provide:

- persistent connection
- history
- completion
- inline help
- output mode switching
- pleasant exploration

## Architecture

Reuse the same runtime parser and invoker used by normal CLI commands.
The shell should not invent a second semantics.

## Commands inside shell

At minimum:

- `tools`
- `resources`
- `prompts`
- direct tool invocation by name
- `set output json|auto|yaml|raw|table`
- `help`
- `exit`

## Testing

Use PTY-based tests for:

- history
- prompt rendering
- completion hooks
- interactive elicitation

---

## Sampling and elicitation plan

## Elicitation

### v1 behavior

- if interactive TTY is available, prompt the user
- if not interactive, fail with exit code `8` and a clear message
- later add non-interactive answer sources (`--answers file.yaml`) if needed

Implementation:

- render the server message clearly on `stderr`
- prompt for each requested field
- validate response shape before sending back

## Sampling

### v1 goal

Support server requests for model-generated responses when practical, without making the feature mandatory for ordinary tool use.

### Behavior

- auto-detect configured provider from env/config when enabled
- print a short notice to `stderr` when sampling is invoked
- allow `--no-sampling` / config disable
- if sampling is requested but not configured, fail clearly with hint

### Sampler interface

```go
type Sampler interface {
    CreateMessage(ctx context.Context, req SamplingRequest) (SamplingResponse, error)
}
```

Start with:

- OpenAI-compatible implementation
- Anthropic implementation
- auto-selector based on env/config

---

## Doctor plan

`mcp2cli doctor <server>` should answer: “why doesn’t this work?”

Checks:

1. config resolution
2. local executable presence or URL parse validity
3. environment variables for auth
4. token cache presence/state
5. transport connection
6. initialize handshake
7. metadata listing
8. optional single tool smoke call when explicitly requested

Output should be readable in `auto` mode and scriptable in `-o json`.

---

## Testing strategy

## Test layers

### 1. Unit tests

Cover:

- config merge rules
- name conversion
- ordered schema parsing
- runtime arg parser
- renderer heuristics
- exit code mapping
- auth helpers

### 2. Integration tests with fixture servers

Build deterministic MCP fixture servers covering:

- basic stdio tools
- remote HTTP tools
- resources
- prompts
- OAuth-like auth handshake stubs
- elicitation
- sampling
- error cases

These should live in `testdata/servers` and be runnable in CI.

### 3. Golden tests

Snapshot:

- `tools` output
- error messages
- table output
- YAML/JSON output
- `doctor` output
- shell help text

### 4. End-to-end CLI tests

Spawn the built binary and assert:

- stdout content
- stderr hints
- exit codes
- child process cleanup

### 5. Cross-platform CI

Matrix:

- Ubuntu
- macOS
- Windows

At minimum:

- `go test ./...`
- `go test -race ./...` on Linux
- `golangci-lint run`
- binary build smoke test

### 6. Optional nightly real-world smoke tests

Run against known public or local MCP servers when credentials/environment allow.
Keep these separate from required CI.

---

## Performance and UX targets

These are guiding targets, not hard SLAs.

- CLI startup overhead should feel instant
- cached completions should return in under 100ms locally
- one-shot commands should not leave orphaned processes
- large JSON output should stream without excessive buffering
- normal success cases should not print noisy banners

---

## Security and privacy rules

1. redact secrets in logs and traces
2. prefer keyring over plain-text token storage
3. if file fallback is used, require `0600` permissions
4. never write auth tokens to stdout
5. keep debug/trace output on stderr
6. make browser-opening behavior explicit
7. do not execute shell strings through `/bin/sh` unless the user explicitly requested shell execution semantics
8. normalize command execution to executable + argv when possible

---

## Phase-by-phase roadmap

## Phase 0: Bootstrap and project hygiene

### Deliverables

- Go module initialization
- Cobra CLI skeleton
- basic `version` command
- config loader skeleton
- Makefile or task runner
- CI workflow
- goreleaser config
- lint/test tooling

### Files

- `go.mod`
- `cmd/mcp2cli/main.go`
- `internal/cli/root.go`
- `.github/workflows/ci.yml`
- `.goreleaser.yaml`
- `Makefile`

### Exit criteria

- `mcp2cli version` works
- CI passes on all supported platforms

---

## Phase 1: Config, registry, and server resolution

### Deliverables

- config load/merge/write
- `add`, `ls`, `rm`
- `expose`, `unexpose`
- shim directory management
- support global and local config
- resolve ephemeral `--command` and `--url`
- standard common flags

### Commands completed

- `add`
- `ls`
- `rm`
- `expose`
- `unexpose`
- shared server resolution used by all future commands

### Exit criteria

- can add/list/remove servers
- can expose a server as the default `mcp-<server>` command or a custom alias like `wea`
- local overrides global
- ephemeral and registered server refs normalize identically

---

## Phase 2: MCP core and stdio transport

### Deliverables

- JSON-RPC core
- stdio transport
- initialize handshake
- list tools
- call tool
- graceful process shutdown

### Commands completed

- `tools <server>`
- basic `tool <server> <tool> --input @file.json`

### Exit criteria

- can list tools from deterministic stdio fixture server
- can call tools with raw JSON input
- child processes are cleaned up reliably

---

## Phase 3: Delightful tool invocation and renderers

### Deliverables

- ordered schema parsing
- runtime tool arg parser
- flags + positionals for simple schemas
- `-o auto|json|yaml|raw|table`
- friendly tool inspection output
- stable error messages and exit codes

### Commands completed

- `tools <server> <tool>` detailed inspection
- `tool <server> <tool> [args...]`

### Exit criteria

- README tool examples work for local servers
- complex schema tools still callable via `--input`
- `-o json` is script-safe

---

## Phase 4: Remote transport and auth

### Deliverables

- HTTP transport
- headers and bearer env support
- OAuth login flow
- keyring-backed token cache
- `login`
- `doctor` basic checks

### Commands completed

- `login`
- `doctor`
- direct and registered remote `tools` / `tool`

### Exit criteria

- remote server with bearer auth works
- OAuth flow completes and token is reused
- `doctor` explains common failures

---

## Phase 5: Resources, prompts, and metadata caching

### Deliverables

- list/read resources
- list/render prompts
- metadata cache for completions and speed
- output renderers applied consistently to metadata commands

### Commands completed

- `resources`
- `resource`
- `prompts`
- `prompt`

### Exit criteria

- README resource/prompt examples work
- metadata cache is created and reused safely

---

## Phase 6: Completions and shell mode

### Deliverables

- bash/zsh/fish completion generation
- dynamic completion backed by cache
- completion support for exposed commands, including default `mcp-*` names and custom aliases
- interactive shell with persistent session
- history and inline help

### Commands completed

- `completion`
- `shell`

### Exit criteria

- server names and tool names complete correctly
- exposed commands complete correctly
- shell mode supports repeated tool calls without reconnecting

---

## Phase 7: Elicitation and sampling

### Deliverables

- elicitation callback support
- TTY prompts
- non-interactive failure behavior
- sampler interface and initial providers
- `--no-sampling` support

### Exit criteria

- fixture server can request user input interactively
- fixture server can request sampling when configured
- failure mode is clear when sampling is unavailable

---

## Phase 8: Polish, docs, packaging, and 1.0 hardening

### Deliverables

- README aligned with actual behavior
- installation docs
- release artifacts
- Homebrew/Scoop packaging
- smoke test scripts
- performance and UX polish
- manual QA pass across major examples

### Exit criteria

- all README examples verified
- first tagged release published

---

## Risks and mitigations

## Risk 1: Dynamic schema parsing gets messy

### Mitigation

- explicitly support a beautiful subset
- keep `--input @file.json` as universal fallback
- centralize mapping logic in `internal/schema` and `internal/invoke`

## Risk 2: Transport spec differences across servers

### Mitigation

- keep transport interface small
- implement one modern HTTP transport first
- add compatibility layer only when needed
- use fixture servers plus optional smoke tests against real servers

## Risk 3: OAuth edge cases are painful cross-platform

### Mitigation

- implement browser flow first
- support URL print fallback
- isolate token storage and browser/callback logic in `internal/auth`

## Risk 4: Completion becomes slow

### Mitigation

- use metadata cache with TTL
- avoid live network calls during completion whenever possible

## Risk 5: Pretty output hides data or breaks scripts

### Mitigation

- make `-o json` exact and boring
- reserve pretty heuristics for `auto`
- keep stdout/stderr separation strict

---

## Suggested implementation order inside each phase

Within each phase, prefer this order:

1. internal models and interfaces
2. deterministic tests
3. implementation
4. golden output snapshots
5. command wiring
6. README/example verification

This keeps the delightful API grounded in tests, not just intention.

---

## Immediate next tasks

1. initialize Go module and Cobra skeleton
2. add config package and `add/ls/rm`
3. create deterministic stdio fixture server
4. implement JSON-RPC framing and stdio session
5. make `tools --command ...` work before anything else
6. then build the delightful tool-argument parser

That path gets to visible value quickly while protecting the hardest part of the product: the runtime CLI UX.


# Round 1: 

## AI response

```text
 Started implementation.

 What’s in now:
 - Go module/bootstrap
 - root CLI
 - config storage
 - server registry commands
 - expose/unexpose commands
 - real exposed-command shims
 - multicall mode for mcp-weather / wea
 - tests + CI scaffold

 Implemented commands:
 - mcp2cli version
 - mcp2cli add
 - mcp2cli ls
 - mcp2cli rm
 - mcp2cli expose
 - mcp2cli unexpose
 - mcp2cli completion

 Implemented behavior:
 - mcp2cli expose weather creates default mcp-weather
 - mcp2cli expose weather --as wea creates wea
 - --as is treated as the full exposed command name
 - exposed commands route shorthand tool calls:
     - wea get-forecast ... rewrites to wea tool get-forecast ...

 Files added:
 - go.mod
 - cmd/mcp2cli/main.go
 - internal/app/invocation.go
 - internal/config/config.go
 - internal/expose/expose.go
 - internal/cli/root.go
 - internal/cli/manage.go
 - internal/cli/version.go
 - internal/cli/completion.go
 - internal/cli/stubs.go
 - tests in:
     - internal/app/invocation_test.go
     - internal/config/config_test.go
 - bootstrap files:
     - .gitignore
     - Makefile
     - .github/workflows/ci.yml
     - .goreleaser.yaml

 Current status:
 - config/registry/expose layer works
 - MCP transport/tool execution commands are still stubbed

 Validation:
 - go test ./... passes

 Next best step:
 1. implement Phase 2 stdio MCP client
 2. make mcp2cli tools --command ... work for real
 3. then make exposed commands like wea tools work on top of that

 If you want, I can continue immediately with the stdio transport and real tools support.
```

Commit:
- bf4d360 — feat: bootstrap CLI and exposed command support

# Round 2:

## AI response

```text
  Continued implementation, following IMPLEMENTATION_PLAN.md Phase 2.

 What’s now working:
 - real stdio MCP client
 - JSON-RPC framing + request/response handling
 - MCP initialize handshake
 - tools/list
 - tools/call
 - mcp2cli tools <server>
 - mcp2cli tools --command '...'
 - mcp2cli tool <server> <tool> --input ...
 - mcp2cli tool --command '...' <tool> --input ...
 - exposed commands also work:
     - wea tools
     - wea echo --input '{"message":"hi"}'

 Implemented files:
 - internal/serverref/resolve.go
 - internal/mcp/types/types.go
 - internal/mcp/jsonrpc/framing.go
 - internal/mcp/jsonrpc/client.go
 - internal/mcp/transport/stdio/stdio.go
 - internal/mcp/client/client.go
 - internal/cli/tools.go
 - testdata/servers/stdiofixture/main.go
 - tests:
     - internal/mcp/client/client_test.go
     - internal/cli/tools_test.go

 Also updated:
 - go.mod
 - go.sum
 - internal/cli/stubs.go

 Validation:
 - go test ./... passes
 - manual smoke tests passed for:
     - registered server
     - direct --command
     - exposed command shorthand

 Current scope/caveat:
 - tool currently supports the Phase 2 raw JSON path via --input
 - schema-driven flags/positionals like --latitude / --longitude are not implemented yet
 - HTTP transport is still not implemented

 Example now:

 bash
   mcp2cli add weather --command 'go run /path/to/testdata/servers/stdiofixture'
   mcp2cli tools weather
   mcp2cli tool weather echo --input '{"message":"hello"}'

   mcp2cli expose weather --as wea
   wea tools
   wea echo --input '{"message":"hi"}'


 Uncommitted changes are in your working tree.

 Next best step from the plan:
 - move into Phase 3
 - derive CLI args from tool schemas
 - implement delightful tool inspection/usage rendering
 - support:
   bash
     mcp-weather get-forecast --latitude 37.7 --longitude -122.4
   
```

Commit:
- 2868f12 — feat: add stdio MCP client and tool commands

# Round 3:

## AI response

```text
 Continued implementation, following IMPLEMENTATION_PLAN.md Phase 3.

 What’s now added

 - internal/naming/kebab.go
     - camelCase / snake_case → kebab-case conversion
 - internal/schema/inspect/inspect.go
     - ordered JSON Schema parsing
     - tool/arg inspection model
 - internal/invoke/toolargs.go
     - runtime parsing of tool args from:
           - flags
           - positionals
           - repeated array flags
           - @file / @-
 - upgraded internal/cli/tools.go
     - friendly tool inspection output
     - real schema-driven tool invocation
     - output modes:
           - auto
           - json
           - yaml
           - raw
           - table where applicable
 - tests for naming, schema inspection, arg parsing, and CLI flows

 What works now

 Tool inspection:

 ```bash
   mcp2cli tools weather get-forecast
 ```

 Now renders like:

 ```text
   NAME
     get-forecast - Get weather forecast for a location

   USAGE
     mcp2cli tool weather get-forecast --latitude <float> --longitude <float>

   ARGS
     --latitude float   Required. Latitude of the location
     --longitude float  Required. Longitude of the location
 ```

 Schema-driven invocation:

 ```bash
   mcp2cli tool weather get-forecast --latitude 37.7 --longitude -122.4
   mcp2cli tool weather get-forecast 37.7 -122.4
   wea get-forecast --latitude 1 --longitude 2
 ```

 Raw JSON fallback still works:

 ```bash
   mcp2cli tool weather echo --input '{"message":"hello"}'
 ```

 Output modes now supported on current tool flows:

 ```bash
   mcp2cli tools weather -o json
   mcp2cli tools weather get-forecast -o yaml
   mcp2cli tool weather get-forecast 1 2 -o json
   mcp2cli tool weather get-forecast 1 2 -o raw
 ```

 Validation

 - go test ./... passes

 Current status vs plan

 Done / started from Phase 3:
 - ordered schema parsing
 - runtime tool arg parser
 - flags + positionals for simple schemas
 - friendlier tool inspection output
 - more output modes

 Still not done from later phases:
 - centralized stable exit code mapping
 - HTTP transport
 - resources/prompts
 - shell mode
 - auth flows
```

Commit: 7d09ee94eb2ebbe2e47243074d35f7882721aa00

# Round 4:

## AI response
 Done — I committed the Phase 3 schema-driven work, then continued.

 Commit:
 - 7d09ee9 — feat: add schema-driven tool invocation

 Then I implemented the next Phase 3 piece from IMPLEMENTATION_PLAN.md:
 - stable error categories and exit codes
 - kind CLI error formatting with hints
 - better wrapping for:
     - usage errors
     - config/server resolution errors
     - stdio transport errors
     - protocol/server MCP errors

 Added:
 - internal/exitcode/exitcode.go
 - internal/exitcode/exitcode_test.go

 Updated:
 - cmd/mcp2cli/main.go
 - internal/cli/state.go
 - internal/cli/tools.go
 - internal/invoke/toolargs.go
 - internal/mcp/client/client.go
 - internal/mcp/jsonrpc/client.go
 - internal/mcp/transport/stdio/stdio.go
 - internal/serverref/resolve.go

 What works now:
 - missing args return exit code 2
 - server/config issues return categorized errors
 - stderr formatting looks like:

 ```text
   error: missing required argument: longitude
   hint: run `mcp2cli tools weather get-forecast`
 ```

 and:

 ```text
   error: tool "nope" not found
   hint: run `mcp2cli tools weather`
 ```

 Validation:
 - go test ./... passes

 Current working tree:
 - code changes are not committed yet
 - IMPLEMENTATION_PLAN.md is still modified in the repo and remains uncommitted

 Next good step from the plan:
 - continue polishing Phase 3 by improving renderers further and/or
 - start Phase 5-style metadata commands (resources, prompts) once you want to widen
 scope
