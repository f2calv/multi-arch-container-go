---
description: 'Go coding conventions - style, package layout, error handling, logging, configuration and concurrency.'
applyTo: '**/*.go'
---

# Go

## Style (enforced by `gofmt` and `go vet`)

- **`gofmt` is authoritative.** Never hand-format Go source; run `gofmt -l -w .` before committing. CI fails when `gofmt -l .` reports anything. Tabs for indentation are non-negotiable - that is gofmt's output.
- **`go vet ./...` and `staticcheck ./...` must pass.** Treat every finding as an error.
- **Follow [Effective Go](https://go.dev/doc/effective_go) and the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).** Where this document is silent, those documents win.
- **Naming**: `MixedCaps`, never underscores. Exported identifiers start with a capital; unexported with a lower case letter. Initialisms keep their case (`ID`, `URL`, `HTTP`, `JSON`) - `GitHubRunID`, not `GitHubRunId`.
- **Short names for short scopes.** `i`, `k`, `v`, `err` inside a tight loop are idiomatic; `settings`, `logger`, `interval` for anything that lives longer. Do not stutter: `config.Load`, not `config.LoadConfig`.
- **File-per-concern**: one focused responsibility per file (`config.go`, `telemetry.go`, `worker.go`). This is the Go analogue of the sibling .NET repository's one-type-per-file rule - do not accumulate unrelated types in `main.go`.
- **`main` is wiring only**: load configuration, install the logger, register signal handling, hand off to a worker. Business logic belongs in another file or package.
- **Accept interfaces, return structs.** Define the interface in the *consuming* package, keep it small (one or two methods), and return concrete types from constructors.
- **Zero values should be useful.** A struct that works without an explicit constructor is better than one that requires `New` plus five setters.
- **Early return over nesting.** Handle the error and `return`; keep the happy path at the left margin.
- **No `else` after a `return`** in the `if` branch.
- **Group related declarations** in a single `const (...)` / `var (...)` block rather than a run of individual statements.

## Error Handling

- **Errors are values - handle them, never discard them.** `_ = f()` requires a comment justifying why the error is genuinely uninteresting.
- **Wrap with context on the way up**: `fmt.Errorf("loading %s: %w", path, err)`. The `%w` verb preserves the chain for `errors.Is`/`errors.As`.
- **Error strings are lower case and unpunctuated** (`"loading appsettings.json: file not found"`), because they are usually embedded in a larger message.
- **Sentinel errors** (`var ErrNotFound = errors.New("not found")`) for conditions callers branch on; custom error types when callers need structured detail.
- **`panic` is for unrecoverable programmer errors only.** Never panic on bad input, missing configuration or I/O failure - return an error.
- **`main` is the only place that calls `os.Exit`**, and it does so after logging the error. `os.Exit` skips deferred functions, so it must never be called deeper in the call stack.

## Logging

- **`log/slog` is the logging framework** - standard library, structured, zero dependencies. It is the Go analogue of Serilog in the sibling .NET repository. Never use `fmt.Println` or the legacy `log` package for application output.
- **Structured attributes, not formatted strings.** Write `slog.Info("processed batch", "user_id", id, "count", len(items))` - never `slog.Info(fmt.Sprintf("processed %d items for %s", len(items), id))`. Attributes are queryable in Loki/Elasticsearch; interpolated text is not.
- **Attribute keys are `snake_case`** and match the sibling repositories' field names where the concept is shared (`git_repository`, `interval_seconds`, `process_architecture`).
- **The message is a constant.** Keep the first argument a static string so events group correctly; put every varying part in an attribute.
- **Handler setup lives in one place** (`telemetry.go`) and is installed exactly once from `main` via `slog.SetDefault`. Application code calls the package-level `slog.Info`/`slog.Error` functions or an injected `*slog.Logger`.
- **Pass `context.Context` to `slog.InfoContext`** on paths that already carry a context, so trace correlation works.
- **Verbosity via a `LOG_LEVEL` environment variable**, defaulting to `info`.

## Configuration

- **Layered, file-then-environment**, using [koanf](https://github.com/knadh/koanf) - the Go analogue of `Microsoft.Extensions.Configuration`. (Viper is the better-known alternative but pulls in a far heavier dependency tree.)
- **Every setting has a default** in a `defaultSettings()` function, so the binary runs with no file and no environment variables present.
- **Unmarshal into a typed struct**; never scatter `os.Getenv` calls through the code base.
- **Keys are `snake_case`** in both the file and the environment. koanf lower-cases environment keys but preserves file keys verbatim, so `snake_case` is the only casing where both sources resolve to the same key.
- **Section separator is `__`** in environment variables (`APP__INTERVAL_SECONDS`), matching the sibling .NET and Rust repositories.
- **Validate at startup.** A configuration error must abort the process immediately with a clear message, never surface later as a runtime surprise.

## Concurrency

- **`context.Context` is the first parameter** of any function that blocks, does I/O or spawns goroutines - `func Run(ctx context.Context, ...)`. Never store a context in a struct.
- **Every goroutine has a defined exit.** If you cannot say what stops it, do not start it.
- **Long-running loops `select` over their work and `ctx.Done()`** so SIGINT/SIGTERM stop them promptly.
- **Use `signal.NotifyContext`** for shutdown rather than a bare channel - it produces a context the whole call tree already understands.
- **Share memory by communicating.** Prefer channels for hand-off; use `sync.Mutex` for genuinely shared state, keeping the critical section as small as possible.
- **`defer` the release immediately after acquisition** (`mu.Lock(); defer mu.Unlock()`), and remember `defer` runs at *function* exit, not block exit.
- **Run tests with `-race` in CI.**

## Testing

- **Table-driven tests** are the default shape for anything with more than one interesting input.
- **Test names describe the behaviour**, not the function - `TestLoadFallsBackToDefaultsWhenFileMissing`, not `TestLoad`.
- **Arrange/Act/Assert**, separated by blank lines. One behaviour per test.
- **Failure messages carry the actual and expected values**: `t.Errorf("greeting = %q, want %q", got, want)`. Use `t.Errorf` to keep going, `t.Fatalf` only when the rest of the test cannot proceed.
- **`t.Setenv`, `t.TempDir` and `t.Cleanup`** instead of manual setup/teardown - they restore state automatically and mark the test as non-parallel where required.
- **No shared mutable package-level state between tests.** Each test must be independently repeatable.
- **Test the exported behaviour**, not private helpers, unless the helper carries genuinely tricky logic.

## Documentation

- **Doc comments on every exported identifier**, starting with the identifier's own name: `// AppConfig is bound from the "app" section of appsettings.json.`
- **A package comment on one file per package** (`// Package main is a multi-architecture container demonstrator.`).
- **First sentence is a complete summary** - it is what `pkg.go.dev` and editor tooling display.
- **Do not delete hyperlinks** to blog posts, issues or StackOverflow answers when refactoring a comment.
- **Comment the "why", not the "what".** Restating the code in prose is noise; explaining a non-obvious constraint is valuable.

## Performance

- **Measure before optimising.** `go test -bench` and `pprof` before any micro-optimisation.
- **Preallocate when the size is known**: `make([]T, 0, n)`, `make(map[K]V, n)`.
- **Avoid allocation in hot loops**: reuse buffers, use `strings.Builder` over `+=`, and consider `sync.Pool` only when profiling justifies it.
- **Pass small structs by value; large ones by pointer.** Do not reflexively use pointers - value semantics are cheaper and safer for small types.
- **`CGO_ENABLED=0`** for container builds, producing a fully static binary suitable for a `distroless/static` or `scratch` base image.

## Dependencies

- **Prefer the standard library.** Every module is compile time, binary size and supply-chain surface; add one only when it earns its place.
- **`go.mod` and `go.sum` are committed**, and CI runs `go mod verify` so the module graph cannot silently drift.
- **The toolchain version is pinned in `go.mod`** and CI reads it via `actions/setup-go` with `go-version-file: go.mod` - never hard-code the version in two places.
- **Run `go mod tidy`** whenever imports change, and commit the result.
