# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
