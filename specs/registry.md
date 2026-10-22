# Registry

**Status:** Draft · **Prefix:** REG

## Purpose

SpecHub owns the names. A name points at one GitHub repo, bound by IDs, claimed by pull request.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| REG-1 | Names live in `names.json` in the `spechub/registry` repo: `name → {repo_id, owner_id, repo, claimed_at, state}`; `state` is `active` or `frozen`. | test_registry_schema |
| REG-2 | A claim is a PR adding one entry. A GitHub Action merges it only if: the PR author has `admin` on the repo (`GET /repos/{o}/{r}/collaborators/{user}/permission`), the repo has topic `spechub`, its front matter `name` equals the claimed name, and the name is free. | test_claim_checks |
| REG-3 | The scope `@x` must equal the repo owner's GitHub login at claim time; `@spechub` is reserved for seeds. | test_claim_scope |
| REG-4 | Names bind to numeric `repo_id` and `owner_id`, never to the `owner/repo` URL. | test_bind_ids |
| REG-5 | If the crawler sees a different `repo_id` or `owner_id` behind a name (transfer, delete, recreate), the name becomes `frozen`: listed, not resolvable, until a new claim PR by an admin of the bound repo. | test_freeze_on_id_change |
| REG-6 | A rename of the bound repo (same `repo_id`) updates `repo` automatically. | test_rename_follows |
| REG-7 | Names are never reassigned to another repo without a manual review in the PR. | manual: PR has `reassign` label and a maintainer approval |

## Out of scope

- Accounts, sign-in, GitHub App claims: later; the PR flow needs no server.
- Verified org badges (DNS TXT).

## Decisions

- Registry as a git repo (Obsidian community plugins, Homebrew taps): free, auditable history, no database.
- IDs over URLs: blocks repojacking, where a freed username is re-registered and the repo recreated.
