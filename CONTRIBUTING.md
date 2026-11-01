# Contributing

Thanks for helping hunt ghosts. Issues and pull requests are welcome.

- Open an issue before large changes so we agree on direction first.
- Small fixes (typos, tests, docs) can come straight in as PRs.
- Keep the tool dependency-free (Go standard library only) and local-first.
- Run `go test ./...` before pushing. New behavior needs tests.
- Claims about what ectocheck catches must be backed by a real run. Do not
  claim detection rates we have not measured.

## Good first contributions

- Better sentence splitting in `internal/claims` (markdown links, abbreviations).
- Smarter retrieval scoring in `internal/retrieve` (stemming, synonyms).
- A `--no-model` mode that reports retrieval-only overlap signals.
