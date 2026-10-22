# Index

**Status:** Draft · **Prefix:** IDX

## Purpose

Find spec and implementation repos on GitHub and publish one `index.json`. No mirroring.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| IDX-1 | A scheduled GitHub Action runs hourly: searches `topic:spechub fork:true` and `topic:spechub-impl fork:true`, plus every repo in `names.json`. | test_crawl_sources |
| IDX-2 | Search results over 1,000 (GitHub's cap per query) are split by `created:` date ranges until each range is under the cap. | test_search_partition |
| IDX-3 | Per spec repo it reads: front matter at the default branch head and at every `v*` tag, tag → commit SHA, `stargazers_count`, `pushed_at`, `repo_id`, `owner_id`. Nothing else; no file contents beyond `README.md` are stored. | test_fields_read |
| IDX-4 | Only names `active` in the registry and bound to the repo's IDs are resolvable; other spec repos appear under `unclaimed`. | test_unclaimed |
| IDX-5 | Change detection: one GraphQL query per ≤ 100 repos fetches default-branch head SHA and tags; a repo is re-read only when they changed. | test_skip_unchanged |
| IDX-6 | `index.json`: `{generated_at, specs: [{name, repo, repo_id, summary, license, stars, pushed_at, derived_from, base, versions: [{version, sha, requires, overrides}], head_sha, implementations: [{repo, stack, url, version}]}], unclaimed: [repo]}`. | test_index_schema |
| IDX-7 | Implementations attach to a spec by their `implements` name; unknown names are dropped. | test_impl_attach |
| IDX-8 | The crawler uses one token, honours `X-RateLimit-Remaining` (stops at < 100), and never scrapes HTML. | test_rate_limit_stop |
| IDX-9 | A static site (GitHub Pages) renders `index.json`: search, spec page with versions, requires graph, implementations and live links. | manual: open a seed spec page |

## Out of scope

- Webhooks via a GitHub App, GH Archive: when hourly polling stops fitting the rate limit.
- Build verification results.

## Decisions

- Git is the database: `index.json` is committed by the Action, so history and rollback are free.
