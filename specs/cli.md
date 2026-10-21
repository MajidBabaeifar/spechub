# CLI

**Status:** Draft · **Prefix:** CLI

## Purpose

`spechub`, run with `npx spechub`, for people and agents.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| CLI-1 | `spechub search <query>` prints matching names, summary, latest version, stars, implementation count; `--json` for machines. | test_search |
| CLI-2 | `spechub info <name>` prints versions, requires tree, implementations with live URLs, licence, attribution. | test_info |
| CLI-3 | `spechub resolve <name>[@range] [--out dir]` writes the bundle ([CMP-7](compose.md)) and `spechub.lock`; default out `./spechub-bundle`. | test_resolve |
| CLI-4 | `spechub lint [dir]` checks [FMT-1..5](format.md) on a local spec repo; exit 1 on errors. | test_lint |
| CLI-5 | The index is fetched from `https://spechub.github.io/index/index.json` (overridable by `SPECHUB_INDEX`), cached for 10 minutes. | test_index_fetch |
| CLI-6 | Spec files are fetched from GitHub at the resolved commit SHA, anonymously, or with `GITHUB_TOKEN` when set. | test_fetch_by_sha |
| CLI-7 | Written in TypeScript, Node ≥ 20, no native dependencies. | build |

## Out of scope

- `spechub claim` (opens the registry PR): after the registry Action exists.

## Decisions

- Fetch by SHA, not tag: tags can be moved; SHAs can't.
