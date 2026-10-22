# MCP

**Status:** Live · **Prefix:** MCP

## Purpose

Let any agent find and resolve specs through MCP.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| MCP-1 | `specfork mcp` serves MCP over stdio. Claude Code: `claude mcp add specfork -- specfork mcp`. | TestMCPHandshake |
| MCP-2 | Tools: `search_specs(query, limit≤20)`, `get_spec(name, version?)` → metadata + area list, `resolve_spec(name, range?)` → bundle as one Markdown document with `BUNDLE.md` first. | TestMCPTools |
| MCP-3 | Uses the same index and resolve code as the CLI. | TestMCPTools, TestInfo |
| MCP-4 | Removed: usage counters move out of scope until the server is hosted. | — |
| MCP-5 | Removed: superseded by MCP-1 (stdio first). | — |

## Out of scope

- Write tools (publish, claim).
- Hosted server (`https://mcp.<domain>`) with per-tool, per-spec daily counters and no personal data: needs a domain and a hosting account.

## Decisions

- Stdio first: works today with no hosting; the hosted server reuses the same tools.
