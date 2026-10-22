# Compose

**Status:** Live · **Prefix:** CMP

## Purpose

Turn a spec plus everything it requires into one flat bundle an agent can build from.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| CMP-1 | `requires` ranges: exact `1.2.3`, caret `^1.2` (same major; same minor when major is 0), or `head` (the default branch head; wins over any version). | TestParseRanges |
| CMP-2 | Minimal version selection: each range picks the lowest published version satisfying it; per name, the highest pick across the tree (including the requirements of every picked version) is selected. One version per name; it must satisfy every range ([CMP-3](compose.md)). | TestMVS |
| CMP-3 | A cycle in `requires` or an unsatisfiable range → error naming the path. | TestCycleError |
| CMP-4 | `overrides` may target only global IDs of specs in the spec's own transitive `requires`; value `removed` or replacement text. Any other target → error. | TestOverrideTargets |
| CMP-5 | Two specs overriding the same ID differently → error, unless the root spec overrides it too; the root wins. | TestOverrideConflict |
| CMP-6 | `spechub.lock` (JSON `{root, specs: {name: {version, sha, repo}}}`) pins every resolved name; resolving the same root with a lock present uses the pinned SHAs; `--update` ignores it. | TestLockfile |
| CMP-7 | Bundle layout: `spechub-bundle/BUNDLE.md` (order, versions, SHAs, overrides applied, licences, attributions) and one folder per spec, `<scope>/<name>/` with its `README.md` and `specs/`; `BUNDLE.md` lists them in dependency order, root last, and tells the agent to build the root and treat dependencies as part of it. | TestBundleLayoutAndRewrite |
| CMP-8 | In the bundle, every ID is rewritten to its global ID; overridden rows show the new text and `(override by <name>)`; removed rows show `Removed (by <name>)`. | TestBundleLayoutAndRewrite |
| CMP-9 | In-repo links (`[AUTH-3](auth.md)`) and global ID references ([FMT-9](format.md)) are rewritten to bundle paths. | TestBundleLayoutAndRewrite |

## Out of scope

- `provides`/`expects` interface checks (e.g. "something provides `User`"): after real compositions show the need.
- Merging two specs' areas into one file.

## Decisions

- Minimal version selection (Go modules): deterministic, no solver, no surprise upgrades.
- Overrides hold only IDs and new text, so a public spec can override a paid one without leaking it.
