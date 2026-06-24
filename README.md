# Directory Runtime

![GitHub Release (latest by date)](https://img.shields.io/github/v/release/agntcy/dir-runtime)
[![Coverage](https://codecov.io/gh/agntcy/dir-runtime/branch/main/graph/badge.svg)](https://codecov.io/gh/agntcy/dir-runtime)
[![License](https://img.shields.io/github/license/agntcy/dir-runtime)](./LICENSE.md)
[![Contributor-Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-fbab2c.svg)](CODE_OF_CONDUCT.md)

The Directory Runtime watches container runtimes (Docker, Kubernetes) for labeled workloads and exposes them through a gRPC API. Resolvers enrich workloads with capability metadata (A2A agent cards, OASF records from the [Directory](https://github.com/agntcy/dir)) for discovery-based routing.

## Quick Start

```bash
# Build images
IMAGE_TAG=latest task build

# Deploy discovery + server
docker compose -f install/docker/docker-compose.yml up -d

# Query discovered workloads
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads
```

See the [documentation](docs/dir-component-runtime-discovery.md) for example workloads, Kubernetes deployment, workload labels, configuration, and the full gRPC API reference.

## Documentation

- [Runtime Discovery](docs/dir-component-runtime-discovery.md) — architecture, setup, labels, configuration, and API

## Contributing

Contributions are what make the open source community such an amazing place to
learn, inspire, and create. Any contributions you make are **greatly
appreciated**. For detailed contributing guidelines, please see
[CONTRIBUTING.md](CONTRIBUTING.md).

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
