# syntax=docker/dockerfile:1
#
# Multi-architecture container image built from a SINGLE Dockerfile.
#
# This file deliberately mirrors its sibling repositories stage-for-stage and
# comment-for-comment, so that a developer fluent in one language can learn the
# containerisation story of another by diffing the two files:
#
#   https://github.com/f2calv/multi-arch-container-dotnet
#   https://github.com/f2calv/multi-arch-container-go       <- you are here
#   https://github.com/f2calv/multi-arch-container-rust
#   https://github.com/f2calv/multi-arch-container-python
#
# ------------------------------------------------------------------------------
# Stage 1 of 2: build
#
# Pinned to $BUILDPLATFORM (the native architecture of the machine running the
# build) and CROSS-COMPILES to $TARGETPLATFORM. The alternative - emulating the
# target architecture under QEMU - is typically 10-50x slower.
#
# Go has the easiest compiled cross-platform story: the toolchain ships
# every target out of the box, so it is purely a matter of setting GOOS/GOARCH.
# ------------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1-bookworm AS build
WORKDIR /src

ARG APP_NAME=multi-arch-container-go

# -- Dependency layer ----------------------------------------------------------
# Copy ONLY the files that influence module resolution so that editing a .go file
# reuses the cached download. `go mod download` is platform-agnostic, so it is
# performed BEFORE TARGETARCH is introduced and is shared by every architecture.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    go mod download

# -- Compile layer -------------------------------------------------------------
COPY . .

# buildx injects TARGETARCH/TARGETVARIANT automatically:
#   linux/amd64  -> TARGETARCH=amd64  TARGETVARIANT=
#   linux/arm64  -> TARGETARCH=arm64  TARGETVARIANT=
#   linux/arm/v7 -> TARGETARCH=arm    TARGETVARIANT=v7
# Concatenating the two gives a single flat token to switch on: amd64|arm64|armv7.
ARG TARGETARCH
ARG TARGETVARIANT
RUN --mount=type=cache,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,target=/root/.cache/go-build,id=go-build-${TARGETARCH}${TARGETVARIANT},sharing=locked <<EOF
set -eux
# Map the Docker platform onto GOARCH (+ GOARM for 32-bit ARM).
# https://go.dev/doc/install/source#environment
case "${TARGETARCH}${TARGETVARIANT}" in
    amd64) export GOARCH=amd64                ;;
    arm64) export GOARCH=arm64                ;;
    armv7) export GOARCH=arm    GOARM=7       ;;
    *) echo "unsupported platform: linux/${TARGETARCH}/${TARGETVARIANT}" >&2; exit 1 ;;
esac
# CGO_ENABLED=0 produces a fully static binary, which is what allows the tiny
# `distroless/static` final image below. -trimpath and -s -w strip absolute paths
# and debug symbols so the build is reproducible and the binary is small.
export CGO_ENABLED=0 GOOS=linux
go build -trimpath -ldflags "-s -w" -o "/out/${APP_NAME}" "./src/${APP_NAME}"
EOF

# ------------------------------------------------------------------------------
# Stage 2 of 2: final
#
# No --platform override here, so buildx resolves the base image for
# $TARGETPLATFORM and the resulting image is genuinely native to the target.
#
# Alternatives, smallest to largest:
#   scratch                                   ~0MB, but no CA certs, no /etc/passwd, no tzdata
#   gcr.io/distroless/static-debian12:nonroot ~2MB, CA certs + tzdata + non-root user (used here)
#   gcr.io/distroless/base-debian12:nonroot   ~20MB, adds glibc for CGO_ENABLED=1 builds
# ------------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS final
WORKDIR /app
COPY --from=build /out/multi-arch-container-go .
# Base configuration; every value can be overridden by an environment variable at runtime.
COPY appsettings.json .

# -- Provenance ----------------------------------------------------------------
# Supplied by the CI workflow (.github/workflows/ci.yml) or by build.sh/build.ps1.
ARG GIT_REPOSITORY=n/a
ENV GIT_REPOSITORY=$GIT_REPOSITORY
ARG GIT_BRANCH=n/a
ENV GIT_BRANCH=$GIT_BRANCH
ARG GIT_COMMIT=n/a
ENV GIT_COMMIT=$GIT_COMMIT
ARG GIT_TAG=n/a
ENV GIT_TAG=$GIT_TAG

ARG GITHUB_WORKFLOW=n/a
ENV GITHUB_WORKFLOW=$GITHUB_WORKFLOW
ARG GITHUB_RUN_ID=0
ENV GITHUB_RUN_ID=$GITHUB_RUN_ID
ARG GITHUB_RUN_NUMBER=0
ENV GITHUB_RUN_NUMBER=$GITHUB_RUN_NUMBER

# https://github.com/opencontainers/image-spec/blob/main/annotations.md
LABEL org.opencontainers.image.title="multi-arch-container-go" \
    org.opencontainers.image.description="Multi-architecture container build (amd64/arm64/armv7) w/Go" \
    org.opencontainers.image.source="https://github.com/f2calv/multi-arch-container-go" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.version="$GIT_TAG" \
    org.opencontainers.image.revision="$GIT_COMMIT"

# The :nonroot distroless tag already runs as uid/gid 65532 - setting it
# explicitly documents the intent and keeps the four sibling repos consistent.
USER nonroot:nonroot

ENTRYPOINT ["/app/multi-arch-container-go"]
