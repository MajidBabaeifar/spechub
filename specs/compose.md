# Compose

**Status:** Draft · **Prefix:** CMP

## Purpose

Turn a spec plus everything it requires into one flat bundle an agent can build from.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| CMP-1 | `requires` ranges: exact `1.2.3`, caret `^1.2` (same major; same minor when major is 0), or `head`. | test_parse_ranges |
| CMP-2 | Version selection is minimal version selection: per name, the highest of the minimum versions required anywhere in the tree. One version per name. | test_mvs |
| CMP-3 | A cycle in `requires` or an unsatisfiable range → error naming the path. | test_cycle_error |
| CMP-4 | `overrides` may target only global IDs of specs in the spec's own transitive `requires`; value `removed` or replacement text. Any other target → error. | test_override_targets |
| CMP-5 | Two specs overriding the same ID differently → error, unless the root spec overrides it too; the root wins. | test_override_conflict |
| CMP-6 | `spechub.lock` pins every resolved name to `{version, sha}`; resolve with a lock uses the pinned SHAs. | test_lockfile |
| CMP-7 | Bundle layout: `spechub-bundle/BUNDLE.md` (order, versions, SHAs, overrides applied, licences, attributions) and one folder per spec in dependency order, root last. | test_bundle_layout |
| CMP-8 | In the bundle, every ID is rewritten to its global ID; overridden rows show the new text and `(override by <name>)`; removed rows show `Removed (by <name>)`. | test_bundle_rewrite |
| CMP-9 | In-repo links (`[AUTH-3](auth.md)`) and global ID references ([FMT-9](format.md)) are rewritten to bundle paths. | test_bundle_links |

## Out of scope

- `provides`/`expects` interface checks (e.g. "something provides `User`"): after real compositions show the need.
- Merging two specs' areas into one file.

## Decisions

- Minimal version selection (Go modules): deterministic, no solver, no surprise upgrades.
- Overrides hold only IDs and new text, so a public spec can override a paid one without leaking it.
