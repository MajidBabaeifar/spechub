# CLI

**Status:** Live · **Prefix:** CLI

## Purpose

`specfork`, for people and agents.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| CLI-1 | `specfork search <query>` prints matching names, summary, latest version, stars, implementation count; `--json` for machines. Matches name and summary, case-insensitive, all words. | TestSearch |
| CLI-2 | `specfork info <name>` prints versions, requires, required-by, implementations with live URLs, licence, attribution; `--json`. | TestInfo |
| CLI-3 | `specfork resolve <name>[@range] [--out dir]` writes the bundle ([CMP-7](compose.md)) and `specfork.lock` in the current directory; default out `./specfork-bundle`. | TestResolve |
| CLI-4 | `specfork lint [dir]` checks [FMT-1..5, FMT-9](format.md) on a local spec repo; exit 1 on errors; warnings don't fail. | TestLint |
| CLI-5 | The index is fetched from `https://specfork.github.io/specfork/index.json` (overridable by `SPECFORK_INDEX`, a URL or file path), cached in the user cache dir for 10 minutes. | TestIndexFetch |
| CLI-6 | Spec files are fetched from `raw.githubusercontent.com` at the resolved commit SHA; no API calls, no token needed. | TestFetchBySHA |
| CLI-7 | Written in Go, no cgo; installed with `go install github.com/specfork/specfork/cmd/specfork@latest`. | CI: `go build ./...` with `CGO_ENABLED=0` |
| CLI-8 | `specfork mcp` runs the MCP server on stdio ([MCP](mcp.md)). | TestMCPTools |
| CLI-9 | Exit codes: 0 ok, 1 lint errors or not found, 2 usage error, 3 network or index error. | TestExitCodes |

## Out of scope

- `specfork claim` (opens the registry PR).

## Decisions

- Fetch by SHA, not tag: tags can be moved; SHAs can't.
- Go, not TypeScript: one static binary, installable from GitHub with no package-registry account; the crawler and MCP server share the same code.
