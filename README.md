# Directory Runtime

![GitHub Release (latest by date)](https://img.shields.io/github/v/release/agntcy/dir-runtime)
[![CI](https://github.com/agntcy/dir-runtime/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/agntcy/dir-runtime/actions/workflows/ci.yml)
[![Coverage](https://codecov.io/gh/agntcy/dir-runtime/branch/main/graph/badge.svg)](https://codecov.io/gh/agntcy/dir-runtime)
[![License](https://img.shields.io/github/license/agntcy/dir-runtime)](./LICENSE.md)
[![Contributor-Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-fbab2c.svg)](CODE_OF_CONDUCT.md)

The Directory Runtime is a discovery service that watches workloads and resolves capabilities using the [Directory](https://github.com/agntcy/dir). It enables runtime-level integration with the Directory's capability-based discovery system, allowing workloads to be automatically discovered and matched based on their functional characteristics.

## About The Project

Directory Runtime provides runtime discovery and capability resolution for workloads registered in the [Directory](https://github.com/agntcy/dir). It bridges the gap between static agent records and live, running services by continuously monitoring workloads and resolving their capabilities against Directory records.

## Getting Started

To get a local copy up and running follow these simple steps.

### Prerequisites

- [Go](https://go.dev/doc/devel/release)
- [Docker](https://www.docker.com/)
- [Taskfile](https://taskfile.dev/) (optional)

### Installation

1. Clone the repository

   ```sh
   git clone https://github.com/agntcy/dir-runtime.git
   ```

2. Build the project

   ```sh
   cd dir-runtime
   go build ./...
   ```

## Usage

_For more examples, please refer to the [Documentation](https://docs.agntcy.org/) or
the [Wiki](https://github.com/agntcy/dir-runtime/wiki)._

## Roadmap

See the [open issues](https://github.com/agntcy/dir-runtime/issues) for a list
of proposed features (and known issues).

## Contributing

Contributions are what make the open source community such an amazing place to
learn, inspire, and create. Any contributions you make are **greatly
appreciated**. For detailed contributing guidelines, please see
[CONTRIBUTING.md](CONTRIBUTING.md).

## License

Distributed under the Apache-2.0 License. See [LICENSE](LICENSE.md) for more
information.

## Copyright Notice

[Copyright Notice and License](./LICENSE.md)

Distributed under Apache 2.0 License. See LICENSE for more information.
Copyright AGNTCY Contributors (https://github.com/agntcy)
