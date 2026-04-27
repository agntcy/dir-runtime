# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0] - 2026-04-27

Initial release of Directory Runtime as a standalone repository, migrated from
[agntcy/dir](https://github.com/agntcy/dir).

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

[Full Changelog](https://github.com/agntcy/dir-runtime/releases/tag/v1.0.0)
