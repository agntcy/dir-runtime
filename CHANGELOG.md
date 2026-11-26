# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.7.0] - 2026-11-26

### Added
- **Discovery**: A2A card `supportedInterfaces` are promoted into workload locators, so an
  agent reachable over SLIM advertises that address in discovery output instead of only in
  its card

### Changed
- **Install**: The chart's A2A discovery paths now list both `/.well-known/agent-card.json`
  and `/.well-known/card.json`, matching the resolver's built-in default
- **Dependencies**: Bumped Kubernetes client libraries to `v0.37.2`
- **Dependencies**: Bumped `go.etcd.io/etcd/client/v3` to `v3.7.4`
- **Dependencies**: Bumped `github.com/moby/moby/api` to `v1.57.0`
- **Toolchain**: Updated Go to `v1.27.4`
- **CI/CD**: Updated GitHub Actions

### Fixed
- **CI/CD**: Renovate now runs `gomodTidyAll`, so modules linked by `replace` directives are
  tidied together instead of being left inconsistent
- **CI/CD**: `licensei` ignores the GitLab `client-go` v3 module path

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.6.0...v1.7.0)

## [v1.6.0] - 2026-11-11

### Changed
- **Store**: The `DiscoveredWorkload` CRD no longer carries a Workload's `addresses` and
  `ports`, following their removal from `agntcy.dir.runtime.v1.Workload`. The `Addresses`
  and `Ports` print columns are gone with them; use locators instead
- **Dependencies**: Pinned `github.com/agntcy/dir` modules to `v1.9.0`
- **Dependencies**: Bumped `google.golang.org/grpc` to `v1.85.0`
- **Dependencies** (security): Bumped `golang.org/x/net` to `v0.62.0`
- **Dependencies** (security): Bumped `github.com/klauspost/compress` to `v1.20.2`
- **Dependencies** (security): Bumped `github.com/go-jose/go-jose/v4` to `v4.1.6`
- **Dependencies** (security): Bumped `github.com/go-chi/chi/v5` to `v5.4.0`
- **Dependencies**: Bumped `golang.org/x/sys` to `v0.50.0`
- **Toolchain**: Updated Go to `v1.27.3`
- **CI/CD**: Updated GitHub Actions

### Fixed
- **CI/CD**: `dir-runtime-server` and `dir-runtime-discovery` are published as separate
  images. Both build targets previously shared one set of tags, so whichever finished last
  overwrote the other and every published `dir-runtime-server` tag in fact held the
  discovery image

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.5.0...v1.6.0)

## [v1.5.0] - 2026-10-20

### Added
- **Discovery**: Stale process descriptors are removed, and a descriptor whose PID has been
  reused by an unrelated process is no longer reported as that workload

### Changed
- **Dependencies**: Bumped `github.com/agntcy/dir` modules to `v1.8.0`
- **Dependencies**: Bumped `tonistiigi/xx` to `v1.10.0`
- **Dependencies**: Updated the `docker/dockerfile:1` digest
- **Toolchain**: Updated Go to `v1.27.2`
- **CI/CD**: Updated GitHub Actions

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.4.0...v1.5.0)

## [v1.4.0] - 2026-10-14

### Added
- **Discovery**: Process runtime adapter, which lists workloads from a descriptor directory
  and watches it for changes, alongside the existing Docker and Kubernetes adapters
- **Discovery**: Workload endpoints are described as locators
- **Server**: `ListWorkloads` filters on runtime type, reachability, locator protocol and
  agent attributes (names, skills, domains, authors, OASF schema versions) read from a
  workload's A2A card extensions, falling back to its OASF record. Adds
  `SERVER_OASF_SCHEMA_URLS` for the schema servers those extensions are read from
- **Utils**: `SetDefaultLogger` lets an embedding program, such as `dirctl daemon`, supply
  the logger dir-runtime packages write through, so output no longer mixes two formats

### Changed
- **Discovery**: Reconciliation is scoped to the instance's own workloads, so daemons
  sharing a store no longer reap each other's

### Fixed
- **Store**: Discovery and server share one SQLite database rather than each opening a
  random temporary file. The path is configurable (`store.sqlite.path`, default
  `~/.agntcy/dir-runtime/workloads.db`) and opened in WAL mode with a busy timeout so both
  can use it concurrently

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.6...v1.4.0)

## [v1.3.6] - 2026-09-22

### Changed
- **Toolchain**: Updated Go to `v1.27.1`
- **Dependencies** (security): Bumped `google.golang.org/grpc` to `v1.84.0`
- **Dependencies** (security): Bumped `golang.org/x/crypto` to `v0.56.0`
- **Dependencies** (security): Bumped `golang.org/x/mod` to `v0.40.0`
- **Dependencies** (security): Bumped `go.etcd.io/etcd/client/pkg/v3` to `v3.6.14`
- **Dependencies**: Bumped `github.com/moby/moby/client` to `v0.6.0`
- **Dependencies**: Bumped `github.com/stretchr/testify` to `v1.12.1`
- **CI/CD**: Updated GitHub Actions

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.5...v1.3.6)

## [v1.3.5] - 2026-08-18

### Changed
- **Dependencies**: Bumped Kubernetes client libraries to `v0.36.3`
- **Dependencies**: Bumped the `quay.io/coreos/etcd` image to `v3.6.11`
- **Dependencies**: Updated `gcr.io/distroless/static`, `golang:1.26.5-bookworm` and
  `docker/dockerfile:1` digests

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.4...v1.3.5)

## [v1.3.4] - 2026-07-29

### Changed
- **Dependencies**: Bumped `github.com/agntcy/dir/api`, `github.com/agntcy/dir/client`, and `github.com/agntcy/dir/utils` to `v1.6.2`
- **Dependencies** (security): Bumped `github.com/klauspost/compress` to `v1.18.7`
- **Dependencies** (security): Bumped `google.golang.org/grpc` to `v1.82.1`
- **Dependencies** (security): Bumped `golang.org/x/net` to `v0.56.0`
- **Dependencies** (security): Bumped `golang.org/x/text` to `v0.39.0`
- **CI/CD**: Updated GitHub Actions

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.3...v1.3.4)

## [v1.3.3] - 2026-07-14

### Changed
- **Toolchain**: Updated Go to `v1.26.5`
- **Dependencies**: Bumped `github.com/moby/moby/client` to `v0.5.0`
- **CI/CD**: Updated GitHub Actions
- **Docs**: Simplified README

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.2...v1.3.3)

## [v1.3.2] - 2026-06-17

### Fixed
- **Modules**: Tidied Go module dependencies

### Changed
- **Toolchain**: Updated Go to `v1.26.4`
- **CI/CD**: Updated GitHub Actions
- **Docs**: Moved documentation files and updated filenames and links

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.1...v1.3.2)

## [v1.3.1] - 2026-05-28

### Changed
- **Toolchain**: Updated Go to `v1.26.3`
- **Dependencies** (security): Bumped `golang.org/x/crypto` to `v0.52.0`
- **Dependencies** (security): Bumped `golang.org/x/net` to `v0.55.0`
- **Dependencies** (security): Bumped `github.com/in-toto/in-toto-golang` to `v0.11.0`
- **Dependencies**: Bumped `google.golang.org/grpc` to `v1.81.1`
- **Dependencies**: Bumped `go.etcd.io/etcd/client/v3` to `v3.6.11`
- **Dependencies**: Bumped `github.com/thalesgroup/crypto11` to `v1.6.1`
- **CI/CD**: Updated GitHub Actions
- **CI/CD**: Fixed Codecov upload

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.3.0...v1.3.1)

## [v1.3.0] - 2026-05-05

### Added
- **CI/CD**: Container image security scanning workflow with Trivy summary reporting
- **CI/CD**: Dependencies workflow with automated critical CVE issue creation
- **Tooling**: Unified Renovatebot configuration shared across `agntcy` repositories

### Changed
- **Dependencies**: Bumped Kubernetes client libraries to `v0.36.0`
- **Dependencies**: Bumped `actions/cache` to `v4.3.0`
- **Modules**: `discovery`, `server`, and `store` now consume `github.com/agntcy/dir/api`
  and `github.com/agntcy/dir/client` at `v1.3.0`

### Removed
- **API**: The local `api` module (Protobuf stubs and CRD types) has been removed.
  Consumers must now depend on `github.com/agntcy/dir/api` instead of
  `github.com/agntcy/dir-runtime/api`. Generated `proto/` definitions and the
  `api/crd` and `api/runtime` packages no longer ship from this repository.

[Full Changelog](https://github.com/agntcy/dir-runtime/compare/v1.2.1...v1.3.0)

## [v1.2.1] - 2026-04-27

Initial release of Directory Runtime as a standalone repository, migrated from
[agntcy/dir](https://github.com/agntcy/dir). Version starts at v1.2.1 to avoid
conflicts with existing container images published from the monorepo.

### Added
- **Discovery**: Event-based Docker container discovery with real-time monitoring
- **Discovery**: Containerd runtime support for container lifecycle tracking
- **Discovery**: Kubernetes workload discovery via CRD-based integration
- **Server**: gRPC API for querying discovered processes and workloads
- **Store**: ETCD-based storage backend for workload state persistence
- **API**: Protobuf definitions and generated Go stubs for runtime services
- **API**: CRD (CustomResourceDefinition) for DiscoveredWorkload resources
- **Install**: Helm chart for Kubernetes deployment
- **Install**: Docker Compose configuration for local development
- **CI/CD**: GitHub Actions workflows for CI, security scanning (CodeQL, Trivy), and release automation
- **CI/CD**: Post-release workflow for multi-module Go tag creation
- **Tooling**: Taskfile with build, test, lint, generate, and release tasks
- **Tooling**: golangci-lint, licensei, buf, and multimod configurations
- **Tooling**: Renovatebot and Dependabot for automated dependency updates
- **Tooling**: Pre-commit hooks with golangci-lint integration
- **Tooling**: Code coverage reporting with Codecov

[Full Changelog](https://github.com/agntcy/dir-runtime/releases/tag/v1.2.1)
