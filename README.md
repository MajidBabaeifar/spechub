# SpecHub

App specs that agents can find, compose and build from. Specs don't depend on a stack: one spec, many implementations.

**Browse:** https://majidbabaeifar.github.io/spechub/

## Install

```bash
go install github.com/MajidBabaeifar/spechub/cmd/spechub@latest
```

## Use

```bash
spechub search booking
spechub info @spechub/scheduling
spechub resolve @spechub/scheduling
```

`resolve` writes `spechub-bundle/` (the spec and everything it requires, IDs made global, overrides applied) and `spechub.lock`. Point your agent at `spechub-bundle/BUNDLE.md`.

Let the agent do it itself over MCP:

```bash
claude mcp add spechub -- spechub mcp
```

Tools: `search_specs`, `get_spec`, `resolve_spec`.

## Publish a spec

1. Make a public GitHub repo with a `README.md` that starts with front matter, and one `specs/*.md` file per area:

   ```yaml
   ---
   spechub: 1
   name: "@your-login/booking"
   summary: One line, ≤ 200 chars.
   license: CC-BY-4.0
   requires:
     "@spechub/auth": "^0.1"
   overrides:
     "@spechub/auth:AUTH-8": removed
   ---
   ```

   Each area has `**Status:**`, `**Prefix:**` and a table `| ID | Requirement | Verified by |`. Format: [specs/format.md](specs/format.md).
2. `spechub lint .` until it passes.
3. Add the topic `spechub` and tag a release: `git tag v0.1.0 && git push --tags`.
4. [Claim the name](#claim-a-name).

## Claim a name

Open a PR that adds one entry to [registry/names.json](registry/names.json):

```json
"@your-login/booking": {"repo_id": 123, "owner_id": 456, "repo": "your-login/booking", "claimed_at": "2026-10-22", "state": "active"}
```

Get the IDs with `gh api repos/your-login/booking --jq '{repo_id: .id, owner_id: .owner.id}'`. The scope must be your GitHub login (lower-cased). The `claim` workflow checks the PR and merges it when everything matches ([REG](specs/registry.md)); the index picks it up within the hour.

## Publish an implementation

Built something from a spec? Add the topic `spechub-impl` and front matter to its README:

```yaml
---
implements: "@spechub/scheduling@0.1.0"
stack: [go, sqlite]
url: https://your-live-app.example
---
```

It shows up on the spec's page.

## How it works

- Specs live in their own repos; SpecHub stores no copies ([IDX](specs/index.md)).
- Names are bound to GitHub repo and owner IDs, so a renamed repo is followed and a transferred one is frozen ([REG](specs/registry.md)).
- An hourly workflow crawls GitHub and publishes `index.json` and the site to Pages.
- `resolve` uses minimal version selection, pins commits in `spechub.lock`, and fetches files by SHA ([CMP](specs/compose.md)).
- Seed specs are extracted from open-source apps and proven by rebuilding them in another stack ([EXT](specs/extract.md)).

Specs: [specs/README.md](specs/README.md).
