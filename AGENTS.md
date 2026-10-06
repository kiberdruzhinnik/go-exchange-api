# Repository Guidelines

## Project Structure & Module Organization

This is a Go 1.25 module (`github.com/kiberdruzhinnik/go-exchange-api`). HTTP exchange handlers and data retrieval live in `api/`; Redis and general helpers live in `utils/`; shared constants and errors are in `constants/` and `errors/`. Executables are under `cmd/`: `go-exchange-api` is the HTTP service, `healthcheck` probes it, and `cbr` is a standalone CBR data example. Deployment files are `Dockerfile` and `docker-compose.yml`; user-facing setup instructions and screenshots are in `README.md` and `images/`.

## Build, Test, and Development Commands

- `go build ./...` compiles all packages and executables.
- `go test ./...` runs the tests in `tests/`.
- `go vet ./...` checks for common Go mistakes.
- `go run ./cmd/go-exchange-api` starts the API on Gin's default `:8080` address. Set `EXCHANGE_API_REDIS` to a Redis URL to enable MOEX history caching; when set, Redis must be reachable at startup.
- `docker compose up -d` launches the configured API and Redis services.

## Coding Style & Naming Conventions

Use standard Go formatting: tabs for indentation and `gofmt` before submitting. Keep package names short and lowercase; exported identifiers use `MixedCaps`, while unexported helpers use `mixedCaps`. Keep exchange-specific behavior in `api/` and shared infrastructure in `utils/`. Run `go mod tidy` after dependency changes and include the resulting `go.mod` and `go.sum` updates.

## Testing Guidelines

Place all tests in the `tests/` folder, using package `tests` and `*_test.go` filenames; do not place test files beside production code. Group tests by feature in separate files, such as `moex_test.go`, `spbex_test.go`, and `cbr_test.go`; keep shared test helpers in `http_helpers_test.go`. Name test functions `TestFeatureOrBehavior`. Use Go's standard `testing` package and cover API parsing, error handling, and utility behavior where relevant. Run `go test ./...` before opening a change; avoid requiring live exchange services in unit tests by isolating external HTTP and Redis interactions.

## Commit & Pull Request Guidelines

Recent history uses short, descriptive subjects (for example, `fix image path`); dependency automation uses `Bump ...` subjects. Follow that pattern with an imperative, focused summary. Pull requests should explain the behavior change, note configuration or API effects, link related issues when available, and include the commands used to validate the change. Update `README.md` when deployment or endpoint usage changes.

## Configuration & Operations

The service reads `EXCHANGE_API_REDIS`; Compose supplies `redis://redis-cache:6379/0`. The API exposes `/moex/:ticker`, `/spbex/:ticker`, `/cbr/:ticker`, and `/healthcheck`. Do not commit credentials or local environment files; keep container and health-check behavior aligned when changing runtime configuration.
