# Copilot Instructions

## Shared Instructions

Shared Copilot instruction files are maintained centrally in the [.github](https://github.com/f2calv/.github) repository under `instructions/`, and are applied to every workspace from the VS Code user profile via `~/.copilot/instructions`. They are deliberately not copied into this repository, so a change there takes effect everywhere without a pull request here.

Everything below is specific to this repository.

## Repository Purpose

This repository is a Go application that demonstrates how to build multi-architecture container images (amd64, arm64, arm/v7) from a single `Dockerfile` using `docker buildx`. It is a reference implementation, not a production workload.

## Sibling Repositories (alignment is a hard requirement)

Four repositories implement the *same* trivial worker application in four languages:

- [multi-arch-container-dotnet](https://github.com/f2calv/multi-arch-container-dotnet)
- [multi-arch-container-go](https://github.com/f2calv/multi-arch-container-go) (this one)
- [multi-arch-container-rust](https://github.com/f2calv/multi-arch-container-rust)
- [multi-arch-container-python](https://github.com/f2calv/multi-arch-container-python)

Their premise is that a developer fluent in one language can learn another language's containerisation story by diffing two repositories. **Any change made here must be considered for the other two.** Keep the following as close to identical as possible:

- Repository layout and file names.
- `Dockerfile` stage names (`build`, `final`), section comment banners and ordering.
- The `ARG`/`ENV` provenance block and OCI `LABEL` block.
- Environment variable names consumed by the application — both the flat `GIT_*`/`GITHUB_*` provenance variables and the `APP__*` configuration overrides.
- Application file responsibilities: configuration model, logging setup, worker loop, entry-point wiring.
- `.github/workflows/ci.yml` job names and structure.
- `.editorconfig` common section, `.pre-commit-config.yaml`, `.devcontainer/`, `.vscode/extensions.json`.
- `build.sh` / `build.ps1` are byte-identical (all values are derived from git).
- `README.md` section headings.

## No Helm Charts

These repositories are **application code only**. Kubernetes packaging lives in the standalone [f2calv/helm-charts](https://github.com/f2calv/helm-charts) repository, which provides a single multi-purpose chart used by all deployments. Do not reintroduce a `charts/` directory or a `chart` job in `ci.yml`.

## Linting is manual, never an auto-installed git hook

Do **not** wire `pre-commit install` into `.devcontainer/postCreateCommand.sh`, `postStartCommand.sh` or the README. The hook cost is fixed interpreter start-up per hook rather than per file, so a one-file commit pays the same price as a full run — noticeable on slower hardware. Linting is run manually with `pre-commit run --all-files`, and the `lint` job in `ci.yml` is the authoritative gate. A once-per-push hook (`pre-commit install --hook-type pre-push`) is an acceptable opt-in, never a default.

## Project Structure

- `src/multi-arch-container-go/` – application source.
  - `main.go` – entry point; configuration, logging and signal wiring only.
  - `config.go` – `AppConfig` / `Settings` types and the layered loader.
  - `telemetry.go` – `log/slog` handler installation.
  - `worker.go` – the worker loop.
  - `config_test.go` – unit tests.
- `appsettings.json` – base configuration.
- `go.mod` / `go.sum` – module definition, pinned toolchain version and dependency checksums.
- `Dockerfile` – two-stage, cross-compiling, multi-architecture build.
- `.github/workflows/ci.yml` – CI/CD using reusable workflows from [f2calv/gha-workflows](https://github.com/f2calv/gha-workflows).
- `.devcontainer/` – VS Code devcontainer (Go toolchain + Docker-outside-of-Docker). All Go tooling runs in the container; nothing is installed on the host.
- `build.sh` / `build.ps1` – local build scripts for manual testing.

> Note on layout: Go convention would normally place the entry point under `cmd/<name>/`. `src/<name>/` is used deliberately to mirror the sibling .NET repository (`src/multi-arch-container-dotnet/`) and the f2calv repository-structure rule above. Do not "fix" it.

## Technology Stack

- **Language**: Go (toolchain version pinned in `go.mod`)
- **Logging**: `log/slog` (standard library), with a text or JSON handler selected by configuration
- **Configuration**: [koanf](https://github.com/knadh/koanf) (appsettings.json → environment variables)
- **Container**: Docker (multi-stage, distroless static final image, non-root)
- **CI/CD**: GitHub Actions (reusable workflows from `f2calv/gha-workflows`)
- **Versioning**: GitVersion (MainLine mode)

## Configuration Keys

Configuration keys are **snake_case** in both `appsettings.json` and the environment. This is deliberate: koanf lower-cases environment keys but preserves file keys verbatim, so snake_case is the only casing where both sources resolve to the same key — and it is what the sibling .NET and Rust repositories use.

| Key | Environment variable | Default |
| --- | --- | --- |
| `app.greeting` | `APP__GREETING` | `Hello from a multi-architecture container` |
| `app.interval_seconds` | `APP__INTERVAL_SECONDS` | `3` |
| `app.log_format` | `APP__LOG_FORMAT` | `text` |

The flat provenance variables (`GIT_REPOSITORY`, `GIT_BRANCH`, `GIT_COMMIT`, `GIT_TAG`, `GITHUB_WORKFLOW`, `GITHUB_RUN_ID`, `GITHUB_RUN_NUMBER`) are baked into the image by the `ARG`/`ENV` block of the `Dockerfile` and unmarshalled onto `Settings`.

Log verbosity is controlled separately by `LOG_LEVEL` (`debug`|`info`|`warn`|`error`), defaulting to `info`.

## Target Platforms

The Dockerfile maps `TARGETARCH`+`TARGETVARIANT` onto `GOARCH` (plus `GOARM` for 32-bit ARM):

- `linux/amd64` → `GOARCH=amd64`
- `linux/arm64` → `GOARCH=arm64`
- `linux/arm/v7` → `GOARCH=arm GOARM=7`

## Container Conventions

- Keep the `Dockerfile` single-file with multi-stage builds; never add per-architecture Dockerfiles.
- The final image is `gcr.io/distroless/static-debian12:nonroot`, which requires `CGO_ENABLED=0` so the binary is fully static.
- Heredoc `RUN <<EOF` blocks require **LF line endings**. `.gitattributes` enforces this; a CRLF `Dockerfile` fails at build time with `/bin/sh: set: Illegal option -`.
