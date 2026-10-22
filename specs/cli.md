# CLI

**Status:** Draft · **Prefix:** CLI

## Purpose

`spechub`, for people and agents.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| CLI-1 | `spechub search <query>` prints matching names, summary, latest version, stars, implementation count; `--json` for machines. Matches name and summary, case-insensitive, all words. | test_search |
| CLI-2 | `spechub info <name>` prints versions, requires, required-by, implementations with live URLs, licence, attribution; `--json`. | test_info |
| CLI-3 | `spechub resolve <name>[@range] [--out dir]` writes the bundle ([CMP-7](compose.md)) and `spechub.lock` in the current directory; default out `./spechub-bundle`. | test_resolve |
| CLI-4 | `spechub lint [dir]` checks [FMT-1..5, FMT-9](format.md) on a local spec repo; exit 1 on errors; warnings don't fail. | test_lint |
| CLI-5 | The index is fetched from `https://majidbabaeifar.github.io/spechub/index.json` (overridable by `SPECHUB_INDEX`, a URL or file path), cached in the user cache dir for 10 minutes. | test_index_fetch |
| CLI-6 | Spec files are fetched from `raw.githubusercontent.com` at the resolved commit SHA; no API calls, no token needed. | test_fetch_by_sha |
| CLI-7 | Written in Go, no cgo; installed with `go install github.com/MajidBabaeifar/spechub/cmd/spechub@latest`. | ci build |
| CLI-8 | `spechub mcp` runs the MCP server on stdio ([MCP](mcp.md)). | test_mcp_tools |
| CLI-9 | Exit codes: 0 ok, 1 lint errors or not found, 2 usage error, 3 network or index error. | test_exit_codes |

## Out of scope

- `spechub claim` (opens the registry PR).

## Decisions

- Fetch by SHA, not tag: tags can be moved; SHAs can't.
- Go, not TypeScript: one static binary, installable from GitHub with no package-registry account; the crawler and MCP server share the same code.
