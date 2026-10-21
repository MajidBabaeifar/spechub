# MCP

**Status:** Draft · **Prefix:** MCP

## Purpose

Let any agent or app builder find and resolve specs without the CLI.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| MCP-1 | Remote MCP server (streamable HTTP) at `https://mcp.spechub.dev`, no auth. | test_mcp_handshake |
| MCP-2 | Tools: `search_specs(query, limit≤20)`, `get_spec(name, version?)` → front matter + area list, `resolve_spec(name, range?)` → bundle as one Markdown document with `BUNDLE.md` first. | test_mcp_tools |
| MCP-3 | Uses the same `index.json` and resolve code as the CLI. | test_shared_core |
| MCP-4 | Counts calls per tool and per spec name per day; no IPs or personal data stored. | test_usage_counters |
| MCP-5 | Runs on Cloudflare Workers; counters in D1. | deploy |

## Out of scope

- Write tools (publish, claim).

## Decisions

- Usage counters exist from day one: resolves per week is the number that matters.
