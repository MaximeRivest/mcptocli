#!/usr/bin/env bash
set -euo pipefail

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

bin="$tmpdir/mcp2cli"
fixture_dir="$(pwd)/testdata/servers/stdiofixture"

printf 'Building...\n'
go build -o "$bin" ./cmd/mcp2cli

printf 'Testing local stdio workflow...\n'
XDG_CONFIG_HOME="$tmpdir/config" XDG_DATA_HOME="$tmpdir/data" "$bin" add weather --command "go run $fixture_dir" >/dev/null
XDG_CONFIG_HOME="$tmpdir/config" XDG_DATA_HOME="$tmpdir/data" "$bin" tools weather >/dev/null
XDG_CONFIG_HOME="$tmpdir/config" XDG_DATA_HOME="$tmpdir/data" "$bin" tool weather get-forecast 1 2 >/dev/null
XDG_CONFIG_HOME="$tmpdir/config" XDG_DATA_HOME="$tmpdir/data" "$bin" resources weather >/dev/null
XDG_CONFIG_HOME="$tmpdir/config" XDG_DATA_HOME="$tmpdir/data" "$bin" prompts weather >/dev/null

printf 'Smoke test passed.\n'
