# Format

**Status:** Draft · **Prefix:** FMT

## Purpose

What makes a GitHub repo a SpecHub spec, and how it describes itself.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| FMT-1 | A spec repo has the GitHub topic `spechub` and a root `README.md` whose YAML front matter has `spechub: 1`. | TestFrontMatterSchema, TestUnclaimed |
| FMT-2 | Front matter fields: `name` (required, `@scope/name`, `[a-z0-9-]`, ≤ 64 chars), `summary` (required, ≤ 200 chars), `license` (required, SPDX id), `requires` (map name → range), `base` (`{spec: name@version, relation: fork\|extends}`), `derived_from` (`{repo, commit, url, license}`), `overrides` (map global ID → `removed` or replacement text). Unknown fields → lint warning. | TestFrontMatterSchema, TestLintAreas |
| FMT-3 | Requirements live in `specs/*.md`, one area per file, each with `**Status:**`, `**Prefix:**`, and a table `ID \| Requirement \| Verified by`. | TestLintAreas |
| FMT-4 | IDs are `PREFIX-n`; prefixes unique within a repo; IDs unique within a repo; never reused; a dropped row says `Removed`. | TestLintAreas |
| FMT-5 | `Verified by` is a stack-free acceptance check: inputs → expected outcome, a scenario name, or `manual: <what to look at>`. Never empty. | TestLintAreas, TestLintVerifiedByRequired |
| FMT-6 | A version is a git tag `vMAJOR.MINOR.PATCH` (semver); the index stores tag + commit SHA. Untagged repos resolve only as `@head`. | TestFieldsRead |
| FMT-7 | Global requirement ID = `name:ID`, e.g. `@spechub/auth:AUTH-3`. | TestBundleLayoutAndRewrite |
| FMT-8 | An implementation repo has topic `spechub-impl` and README front matter `implements: name@version`, `stack: [..]`, optional `url` (live deployment). The spec never lists its implementations. | TestDetectImplRepo, TestImplAttach |
| FMT-9 | References to another spec are global IDs in backticks (`` `@spechub/teams:TEAM-5` ``) or the bare name; never relative paths across repos. | TestLintAreas |

## Out of scope

- Mandatory depth rules (data model, permissions sections): decided after the first seeds build.
- Non-GitHub hosts.

## Decisions

- Front matter in README, not a separate manifest: one file fewer, and people read the README (Hugging Face model cards).
- Children declare parents (`base`, `implements`); the index inverts the links. No author maintains a list.
- Areas reuse this repo's own spec format, so a spec repo looks like any `specs/` folder.
