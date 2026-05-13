# Contributing

## Local checks

CI runs `golangci-lint`, `go test -race`, and a `linux/mips/softfloat`
cross-compile on every PR. Reproduce locally with:

```
golangci-lint run ./...                  # lint
go test -race -count=1 ./...             # tests
task build:mips                          # cross-compile sanity check
```

On macOS the `netstats` package won't build under your host arch because
`safchain/ethtool` uses Linux-only `unix.ETHTOOL_*` constants. Run the
lint/cross-compile steps with `GOOS=linux` if you want a local preview of
what CI will see:

```
GOOS=linux golangci-lint run ./...
GOOS=linux GOARCH=mips GOMIPS=softfloat go vet ./...
```

The rest of the test suite runs on any host — fixtures under `testdata/`
cover the parsers.

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/).
Release Please reads commits on `main` to decide the next version, so the
prefix matters:

- `feat:` → minor bump (`0.1.0` → `0.2.0`)
- `fix:` → patch bump (`0.1.0` → `0.1.1`)
- `feat!:` or `BREAKING CHANGE:` footer → major bump
- `chore:`, `docs:`, `refactor:`, `test:`, `ci:` → no version bump, still
  shown in the changelog

Examples:

```
feat(ploam): expose previous-state timestamps
fix(pontop): handle empty Allocation Counters page
chore(deps): bump safchain/ethtool to v0.8.0
```

If you squash PRs, write the squash-merge commit in this format — that's
what release-please sees.

## Releasing

Releases are fully automated. On every push to `main`, the
[release.yml](.github/workflows/release.yml) workflow:

1. Runs Release Please, which either opens/updates a release PR (when there
   are unreleased commits) or cuts a release (when the release PR is merged).
2. On a cut release, cross-compiles the MIPS binary, builds the IPK via the
   OpenWrt SDK in Docker, generates `SHA256SUMS`, and uploads everything to
   the GitHub release.

Don't tag manually — let Release Please do it. The next version is decided
from the commit history since the previous tag.
