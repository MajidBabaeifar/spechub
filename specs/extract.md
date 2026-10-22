# Extract

**Status:** Draft · **Prefix:** EXT

## Purpose

Produce seed specs from open-source apps, and prove them by rebuilding.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| EXT-1 | Sources are shallow-cloned into `.sources/<repo>` (gitignored), read only; nothing in them is run. | manual: `.gitignore` has `.sources/` |
| EXT-2 | Before extracting, the source licence is recorded in `derived_from.license`; non-OSI licences (BSL, ELv2, commercial `ee/` folders) are described only from public docs or skipped. | manual: seed front matter |
| EXT-3 | Specs describe behaviour, data and rules; never code, identifiers or text copied from the source. Names are generic (`scheduling`, not the product name). | manual: review |
| EXT-4 | Generic parts (auth, teams, webhooks, …) go into `@spechub/<base>` specs and are `require`d; an app spec holds only its domain. | manual: review |
| EXT-5 | Every requirement has a stack-free `Verified by` check ([FMT-5](format.md)). | `spechub lint` |
| EXT-6 | Each seed is staged in `seeds/<name>/` with the full spec repo layout, then moved to its own repo `MajidBabaeifar/spechub-<name>` (topic `spechub`), tagged `v0.1.0`, and claimed in the registry. | manual |
| EXT-7 | Proof: an agent builds the resolved bundle in a stack different from the source's; failures are fixed in the spec, not the build. The rebuild is published as an implementation repo. | manual: rebuild passes its checks |
| EXT-8 | The pipeline is the Claude Code skill `/extract-spec <repo-url>`. | manual: run on Cal.com |

## Out of scope

- Reverse-engineering closed-source apps.

## Decisions

- Quality over count: 20 seeds that build beat 1,000 generated ones.
