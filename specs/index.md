# Index

**Status:** Draft · **Prefix:** IDX

## Purpose

Find spec and implementation repos on GitHub and publish one `index.json` plus a static site. No mirroring.

## Requirements

| ID | Requirement | Verified by |
|---|---|---|
| IDX-1 | The `index` workflow runs hourly, on pushes to `main` and on demand: `spechub crawl` searches `topic:spechub fork:true` and `topic:spechub-impl fork:true`, and looks up every repo in `names.json` by `repo_id`. | test_crawl_sources |
| IDX-2 | Search results over 1,000 (GitHub's cap per query) are split by `created:` date ranges until each range is under the cap. | test_search_partition |
| IDX-3 | Per spec repo it reads: `repo_id`, `owner_id`, current `owner/repo`, `stargazers_count`, `pushed_at`, topics; default-branch head SHA and every `v*` semver tag with its commit SHA; at each of those commits the README front matter and the list of `specs/*.md` paths. No other file contents. | test_fields_read |
| IDX-4 | Only names `active` in the registry, bound to the repo's IDs and not frozen ([REG-5](registry.md)) are resolvable; other spec repos appear under `unclaimed`. | test_unclaimed |
| IDX-5 | Change detection: the previous `index.json` is fetched from the site; a spec whose head SHA and tag set are unchanged is copied from it without further reads. | test_skip_unchanged |
| IDX-6 | `index.json`: `{generated_at, specs: [{name, repo, repo_id, owner_id, state, summary, license, stars, pushed_at, derived_from, base, head_sha, head: {requires, overrides, files}, versions: [{version, sha, requires, overrides, files}], implementations: [{repo, stack, url, version}]}], unclaimed: [repo]}`; versions sorted newest first. | test_index_schema |
| IDX-7 | Implementations attach to a spec by their `implements` name; unknown names are dropped. | test_impl_attach |
| IDX-8 | The crawler uses one token, honours `X-RateLimit-Remaining` (stops at < 100, keeping the previous entries for unread specs), and never scrapes HTML. | test_rate_limit_stop |
| IDX-9 | The workflow publishes `index.json` and a static site to GitHub Pages at `https://majidbabaeifar.github.io/spechub/`: a searchable list, and a page per spec with versions, requires, required-by, implementations, areas (linked to GitHub at the commit) and the `resolve` command. | test_site_pages; manual: open a seed page |

## Out of scope

- Webhooks via a GitHub App, GH Archive: when hourly polling stops fitting the rate limit.
- Build verification results.

## Decisions

- Published to Pages, not committed: no bot commits in `main`; the previous index on the site is the crawler's memory.
