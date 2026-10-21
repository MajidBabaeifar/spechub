# Format

**Status:** Draft · **Prefix:** FMT

## Purpose

What makes a GitHub repo a SpecHub spec, and how it describes itself.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| FMT-1 | A spec repo has the GitHub topic `spechub` and a root `README.md` whose YAML front matter has `spechub: 1`. | test_detect_spec_repo |
| FMT-2 | Front matter fields: `name` (required, `@scope/name`, `[a-z0-9-]`, ≤ 64 chars), `summary` (required, ≤ 200 chars), `license` (required, SPDX id), `requires` (map name → range), `base` (`{spec: name@version, relation: fork\|extends}`), `derived_from` (`{repo, commit, url, license}`), `overrides` (map global ID → `removed` or replacement text). Unknown fields → lint warning. | test_front_matter_schema |
| FMT-3 | Requirements live in `specs/*.md`, one area per file, each with `**Status:**`, `**Prefix:**`, and a table `ID \| Requirement \| Verified by`. | test_lint_areas |
| FMT-4 | IDs are `PREFIX-n`; prefixes unique within a repo; IDs unique within a repo; never reused; a dropped row says `Removed`. | test_lint_ids |
| FMT-5 | `Verified by` is a stack-free acceptance check: inputs → expected outcome, a scenario name, or `manual: <what to look at>`. Never empty. | test_lint_verified_by |
| FMT-6 | A version is a git tag `vMAJOR.MINOR.PATCH` (semver); the index stores tag + commit SHA. Untagged repos resolve only as `@head`. | test_versions_from_tags |
| FMT-7 | Global requirement ID = `name:ID`, e.g. `@spechub/auth:AUTH-3`. | test_global_id |
| FMT-8 | An implementation repo has topic `spechub-impl` and README front matter `implements: name@version`, `stack: [..]`, optional `url` (live deployment). The spec never lists its implementations. | test_detect_impl_repo |

## Out of scope

- Mandatory depth rules (data model, permissions sections): decided after the first seeds build.
- Non-GitHub hosts.

## Decisions

- Front matter in README, not a separate manifest: one file fewer, and people read the README (Hugging Face model cards).
- Children declare parents (`base`, `implements`); the index inverts the links. No author maintains a list.
- Areas reuse this repo's own spec format, so a spec repo looks like any `specs/` folder.
