# Registry

**Status:** Draft · **Prefix:** REG

## Purpose

SpecHub owns the names. A name points at one GitHub repo, bound by IDs, claimed by pull request.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| REG-1 | Names live in `registry/names.json` in `MajidBabaeifar/spechub`: `{"names": {name: {repo_id, owner_id, repo, claimed_at, state}}}`; `state` is `active` or `frozen`. | TestRegistrySchema |
| REG-2 | A claim is a PR that only adds entries to `names.json`. The `claim` workflow (`pull_request_target`, never runs PR code) checks each entry: the name is free and well-formed; `repo_id`/`owner_id` match the repo; the repo has topic `spechub`; its README front matter `name` equals the claimed name; the PR author is the repo owner (user repos) or has `admin` on it (org repos). All pass → squash-merged; else a comment lists the failures. | TestClaimChecks |
| REG-3 | The scope `@x` must equal the repo owner's GitHub login, lower-cased, at claim time. `@spechub` is reserved for seeds: those claims are never auto-merged. | TestClaimScope |
| REG-4 | Names bind to numeric `repo_id` and `owner_id`, never to the `owner/repo` URL. | TestBindIDs |
| REG-5 | The crawler looks repos up by `repo_id`; if the repo is gone or its `owner_id` differs (transfer, delete, recreate), the index marks the name `frozen`: listed, not resolvable, until a new claim PR. | TestFreezeOnIDChange |
| REG-6 | A renamed repo (same `repo_id`) is followed automatically: the index shows its current `owner/repo`. | TestRenameFollows |
| REG-7 | Names are never reassigned to another repo without a manual review in the PR. | manual: PR has `reassign` label and a maintainer approval |

## Out of scope

- Accounts, sign-in, GitHub App claims: later; the PR flow needs no server.
- Verified org badges (DNS TXT).

## Decisions

- Registry as a file in a git repo (Obsidian community plugins, Homebrew taps): free, auditable history, no database.
- IDs over URLs: blocks repojacking, where a freed username is re-registered and the repo recreated.
- The GitHub name `spechub` belongs to an unrelated organisation, so SpecHub lives under `MajidBabaeifar` until it has its own org; the `@spechub` scope is SpecHub's own and independent of GitHub.
- No bot commits: freezes and renames are computed by the crawler into the index, not written back to `names.json`.
