# Toolchain

- Go 1.26.9+ (`tool` directive pins mockery in `go.mod`)
- `go tool mockery` generates mocks
- `golangci-lint` for linting and formatting (v2.12.2 config)
- `goreleaser` for cross-platform release builds (v2 config in `.goreleaser.yaml`)

## Release workflow

A GitHub Actions workflow (`.github/workflows/release.yml`) runs on every `v*` tag push and via `workflow_dispatch`.
It uses `goreleaser/goreleaser-action@v7` with `release --clean` to build and publish cross-platform archives
(linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) with checksums and grouped changelogs.
