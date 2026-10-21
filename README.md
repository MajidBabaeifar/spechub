# SpecHub

An index of app specs on GitHub that agents can find, compose and build from.
Specs don't depend on a stack; one spec, many implementations.

- Specs live in their own GitHub repos, tagged `spechub`, described by README front matter ([FMT](specs/format.md)).
- Names like `@spechub/scheduling` are claimed by pull request to the registry ([REG](specs/registry.md)).
- An hourly crawler builds `index.json` ([IDX](specs/index.md)).
- `spechub resolve` flattens a spec and everything it requires into one bundle for an agent ([CMP](specs/compose.md), [CLI](specs/cli.md)).
- Seed specs are extracted from open-source apps, then rebuilt to prove them ([EXT](specs/extract.md)).

Start at [specs/README.md](specs/README.md). Seeds in progress: [seeds/](seeds/).
