# Runtime Discovery

Runtime Discovery is a service that watches container runtimes (Docker, Kubernetes) for workloads and provides a gRPC API for querying them.
In addition, it also resolves workloads using various resolvers (A2A, OASF) to extract and provide details about the workload's capabilities.

## Architecture

The system is split into two independent components:

```mermaid
flowchart LR
 subgraph Discovery["Discovery (discovery/)"]
        RA["Runtime Adapters<br>• Docker<br>• Kubernetes<br>• Process"]
        RES["Resolvers<br>• A2A<br>• OASF"]
        SW["Store Writer"]
  end
 subgraph Server["Server (server/)"]
        SPACE[" "]
        API["gRPC API<br>/ListWorkloads"]
  end
    RA --> RES
    RES --> SW
    SPACE ~~~ API
    Discovery ~~~ Server
    SW -- write/manage --> SB["Storage Backend<br>• etcd (distributed)<br>• Kubernetes CRDs (native)"]
    SB -- read/watch --> API

    style SPACE fill:none,stroke:none
```

## Components

### Discovery (`discovery/`)

The discovery component is responsible for:

- Watching runtimes for workloads with the `org.agntcy/discover=true` label. Supported runtimes:

    - Docker: Watches Docker daemon for labeled containers.
    - Kubernetes: Watches Kubernetes API for labeled pods/services.
    - Process: Watches a directory of workload descriptor files written by processes running directly on the host.
    - Extensible architecture allows adding more runtimes in the future.

- Resolving workload metadata using configurable resolvers:

    - A2A resolver: Extracts A2A agent card from workloads with `org.agntcy/agent-type=a2a` label.
    - OASF resolver: Resolves OASF records from Directory for workloads with `org.agntcy/agent-record=<fqdn>` label.
    - Extensible architecture allows adding more resolvers in the future.

- Writing workloads to the storage backend (etcd or CRDs).

The storage backend can be used to expose discovered workloads to other components (e.g., clients/servers) without coupling them directly and to reduce attack surface.

Choose the storage backend based on where the components run:

- **[etcd](https://etcd.io/)** is recommended in non-Kubernetes environments where discovery and server run as separate services (e.g. Docker Compose or across hosts), since both only need network access to etcd.
- **CRDs** can be used in Kubernetes environments for a more native experience, so clients can query workloads via both gRPC and the Kubernetes API.
- **SQLite** (`sqlite`) suits a single host where discovery and server run as local binaries, e.g. a developer machine. They share a database file instead of requiring an etcd instance, so both must be configured with the same path. The file is opened in WAL mode, so several processes can use it at once: readers don't block the writer, and writes are serialized (one writer at a time, others wait up to 5s). This only works for processes on the same host using a local filesystem; don't put the file on a network filesystem (e.g. NFS) or share it between containers through Docker Desktop bind mounts, where file locking is unreliable. Use etcd in those cases.

```mermaid
flowchart LR
    n1["Runtime Discovery"] --> n3["Runtime Server (for querying workloads via gRPC)"] & n4["CRD (for querying workloads via Kubernetes API)"]

    n1@{ shape: rect}
```

### Server (`server/`)

The server component provides a gRPC API for querying discovered workloads.

## Example Setup

### Build Container Images

```bash
IMAGE_TAG=latest task build
```

### Docker Compose

```bash
# Deploy the stack
docker compose -f install/docker/docker-compose.yml up -d

# Deploy example workloads
docker compose -f install/examples/docker-compose.yml up -d

# Add discovery to all networks (required for resolvers to work)
docker network connect examples_team-a runtime-discovery
docker network connect examples_team-b runtime-discovery
docker restart runtime-discovery

# Query the API
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Cleanup
docker compose -f install/docker/docker-compose.yml down
docker compose -f install/examples/docker-compose.yml down
```

### Kubernetes

The Helm chart supports both CRD and etcd storage backends.

#### Setup KIND Cluster

```bash
# Create cluster
kind create cluster --name runtime

# Load images into KIND
kind load docker-image ghcr.io/agntcy/dir-runtime-discovery:latest --name runtime
kind load docker-image ghcr.io/agntcy/dir-runtime-server:latest --name runtime
```

#### Deploy Example Workloads

```bash
kubectl apply -f install/examples/k8s.workloads.yaml
```

#### Deploy with CRD Storage

```bash
# Install the chart with CRD storage (default)
helm install runtime install/chart/

# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=server --timeout=60s

# Query the gRPC API
kubectl port-forward svc/runtime-server 8080:8080 &
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Query the Kubernetes API to see discovered workloads
kubectl get dw
```

#### Deploy with etcd Storage

```bash
# Install the chart with etcd storage
helm install runtime install/chart/ \
  --set etcd.enabled=true \
  --set discovery.config.store.type=etcd \
  --set server.config.store.type=etcd

# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=etcd --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=discovery --timeout=60s
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=server --timeout=60s

# Query the gRPC API
kubectl port-forward svc/runtime-server 8080:8080 &
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads
```

#### Cleanup

```bash
kind delete cluster --name runtime
```

### Process (Host)

The process runtime discovers agents running directly on a host. Each process announces itself by writing a JSON descriptor file to a watched directory, and is removed when the file is deleted or the process exits. Discovery runs as a host binary, since it must see host PIDs (this does not work from a container on Docker Desktop).

#### Descriptor Format

One file per process at `<dir>/<id>.json`; the file name without `.json` is the workload ID. See [`install/examples/process/hello-agent.json`](../install/examples/process/hello-agent.json).

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Workload name |
| `pid` | yes | Process ID; the workload is listed only while this process is alive |
| `ports` | yes | Ports the process listens on (strings or numbers) |
| `labels` | no | Must include `org.agntcy/discover=true`; resolver labels work as for containers |
| `annotations` | no | E.g. `org.agntcy/agent-record` |
| `addresses` | no | Defaults to `["127.0.0.1"]` |

Discovery sets `runtime` and `type` to `process`, `hostname` to the host name, and `isolationGroups` to `["host"]`.

Write descriptors atomically (write a dotfile or `<id>.json.tmp`, then rename it to `<id>.json`) and delete them on clean shutdown. Discovery never modifies or removes descriptor files. Anyone who can write to the directory can register a workload, so keep it private to the user (discovery creates it with mode `0700`).

#### Run Locally

```bash
# Start a toy A2A agent serving an agent card on port 9999
mkdir -p /tmp/hello-agent/.well-known
echo '{"name": "Hello Agent", "description": "Example agent"}' > /tmp/hello-agent/.well-known/agent-card.json
python3 -m http.server 9999 --directory /tmp/hello-agent &
AGENT_PID=$!

# Create the descriptor directory (discovery also creates it on start)
mkdir -p ~/.agntcy/dir-runtime/workloads.d

# Start discovery and server, sharing a SQLite store
DISCOVERY_RUNTIME_TYPE=process DISCOVERY_STORE_TYPE=sqlite DISCOVERY_RESOLVER_OASF_ENABLED=false \
  go -C discovery run ./cmd &
SERVER_STORE_TYPE=sqlite go -C server run ./cmd &

# Announce the agent
cat > ~/.agntcy/dir-runtime/workloads.d/hello-agent.json <<EOF
{
  "name": "hello-agent",
  "pid": $AGENT_PID,
  "labels": {"org.agntcy/discover": "true", "org.agntcy/agent-type": "a2a"},
  "ports": ["9999"]
}
EOF

# Query the API (services.a2a holds the agent card)
grpcurl -plaintext localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Stop the agent; the workload disappears within the poll interval
kill $AGENT_PID
```

To resolve OASF records from a local Directory (`dirctl daemon start`), drop `DISCOVERY_RESOLVER_OASF_ENABLED=false`, set `DIRECTORY_CLIENT_SERVER_ADDRESS=localhost:8888` and `DIRECTORY_CLIENT_AUTH_MODE=insecure`, and add `"org.agntcy/agent-record": "<cid or name:version>"` to the descriptor's labels or annotations.

The walkthrough uses SQLite so no extra service is needed; etcd works the same way by running etcd and setting `DISCOVERY_STORE_TYPE=etcd` and `SERVER_STORE_TYPE=etcd`.

## Workload Labels

Workloads are discovered based on labels. The discovery component watches for workloads with specific labels and processes their metadata.

### Discovery Labels

| Label | Runtime | Description |
|-------|---------|-------------|
| `org.agntcy/discover=true` | Kubernetes | Marks a pod/service for discovery |
| `org.agntcy/discover=true` | Docker | Marks a container for discovery |
| `org.agntcy/discover=true` | Process | Marks a process descriptor for discovery |

### Resolver Labels

Resolvers extract metadata from discovered workloads based on their labels.
They provide additional information about the workload's capabilities.

| Label/Annotation | Description |
|------------------|-------------|
| `org.agntcy/agent-type=a2a` | Enables A2A resolver - fetches A2A agent card from workload |
| `org.agntcy/agent-record=<fqdn>` | Enables OASF resolver - resolves record from Directory (e.g., `my-agent:v1.0.0`) |

To configure OASF resolver, the Directory client must be set up using environment variables (e.g., `DIRECTORY_CLIENT_SERVER_ADDRESS`, `DIRECTORY_CLIENT_AUTH_MODE`).

### Workload Services

Discovered workloads have a `services` field that holds metadata extracted by resolvers:

```json
{
  "id": "4467371c-84fd-4683-ab30-93895d78bab7",
  "name": "service-a2a",
  "hostname": "service-a2a",
  "runtime": "kubernetes",
  "type": "pod",
  "labels": {
    "app": "service-a2a",
    "org.agntcy/discover": "true",
    "org.agntcy/agent-type": "a2a",
    "org.agntcy/agent-record": "my-agent:1.0.0"
  },
  "addresses": [
    "10-244-0-9.team-a.pod"
  ],
  "ports": [
    "8080",
    "9999"
  ],
  "isolationGroups": [
    "team-a"
  ],
  "services": {
    "a2a": {
      "name": "My Agent",
      "description": "...",
      "capabilities": [...]
    },
    "oasf": {
      "cid": "baf123",
      "name": "my-agent:1.0.0",
      "record": {
        "name": "my-agent",
        "version": "1.0.0",
        "skills": [...]
      }
    }
  }
}
```

## Configuration

### Discovery Component

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `DISCOVERY_WORKERS` | Number of resolver workers | `16` |
| `DISCOVERY_STORE_TYPE` | Storage type (`etcd`, `crd`, `sqlite`) | `etcd` |
| `DISCOVERY_STORE_ETCD_HOST` | etcd server hostname | `localhost` |
| `DISCOVERY_STORE_ETCD_PORT` | etcd server port | `2379` |
| `DISCOVERY_STORE_ETCD_USERNAME` | etcd username for authentication | `` |
| `DISCOVERY_STORE_ETCD_PASSWORD` | etcd password for authentication | `` |
| `DISCOVERY_STORE_ETCD_DIAL_TIMEOUT` | Timeout for connecting to etcd | `5s` |
| `DISCOVERY_STORE_ETCD_WORKLOADS_PREFIX` | etcd key prefix for workloads | `/discovery/workloads/` |
| `DISCOVERY_STORE_CRD_NAMESPACE` | Namespace to store workloads in | `default` |
| `DISCOVERY_STORE_CRD_KUBECONFIG` | Path to kubeconfig file (empty for in-cluster) | `` |
| `DISCOVERY_STORE_CRD_RESYNC_PERIOD` | How often to resync the cache from the API server | `30s` |
| `DISCOVERY_STORE_SQLITE_PATH` | SQLite database file shared with the server (`~` expands to home) | `~/.agntcy/dir-runtime/workloads.db` |
| `DISCOVERY_RUNTIME_TYPE` | Runtime type (`docker`, `kubernetes`, `process`) | `docker` |
| `DISCOVERY_RUNTIME_DOCKER_HOST` | Docker daemon socket path | `unix:///var/run/docker.sock` |
| `DISCOVERY_RUNTIME_DOCKER_LABEL_KEY` | Label key to filter containers | `org.agntcy/discover` |
| `DISCOVERY_RUNTIME_DOCKER_LABEL_VALUE` | Label value to filter containers | `true` |
| `DISCOVERY_RUNTIME_KUBERNETES_KUBECONFIG` | Path to kubeconfig file (empty for in-cluster) | `` |
| `DISCOVERY_RUNTIME_KUBERNETES_NAMESPACE` | Namespace to watch (empty for all namespaces) | `` |
| `DISCOVERY_RUNTIME_KUBERNETES_LABEL_KEY` | Label key to filter pods | `org.agntcy/discover` |
| `DISCOVERY_RUNTIME_KUBERNETES_LABEL_VALUE` | Label value to filter pods | `true` |
| `DISCOVERY_RUNTIME_PROCESS_DIR` | Directory containing workload descriptor files (`~` expands to home) | `~/.agntcy/dir-runtime/workloads.d` |
| `DISCOVERY_RUNTIME_PROCESS_POLL_INTERVAL` | How often to rescan the descriptor directory | `2s` |
| `DISCOVERY_RUNTIME_PROCESS_LABEL_KEY` | Label key to filter process descriptors | `org.agntcy/discover` |
| `DISCOVERY_RUNTIME_PROCESS_LABEL_VALUE` | Label value to filter process descriptors | `true` |
| `DISCOVERY_RESOLVER_A2A_ENABLED` | Enable A2A resolver | `true` |
| `DISCOVERY_RESOLVER_A2A_TIMEOUT` | Timeout for A2A discovery | `5s` |
| `DISCOVERY_RESOLVER_A2A_PATHS` | Comma-separated list of paths to probe for A2A discovery | `/.well-known/agent-card.json,/.well-known/card.json` |
| `DISCOVERY_RESOLVER_A2A_LABEL_KEY` | Label key to identify A2A workloads | `org.agntcy/agent-type` |
| `DISCOVERY_RESOLVER_A2A_LABEL_VALUE` | Label value to identify A2A workloads | `a2a` |
| `DISCOVERY_RESOLVER_OASF_ENABLED` | Enable OASF resolver | `true` |
| `DISCOVERY_RESOLVER_OASF_TIMEOUT` | Timeout for OASF resolution | `5s` |
| `DISCOVERY_RESOLVER_OASF_LABEL_KEY` | Label key to identify OASF workloads | `org.agntcy/agent-record` |

#### Discovery OASF Resolver

The OASF resolver requires Directory client configuration via environment variables.
These are the same as those used by the Directory client library, e.g. `DIRECTORY_CLIENT_SERVER_ADDRESS`.
Refer to the [Directory Client documentation](https://github.com/agntcy/dir/tree/main/client) for all available options.

When a workload has the configured OASF resolver label, the resolver attempts to fetch the corresponding record from Directory and validate its signature before attaching it to the workload.

### Server Component

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SERVER_HOST` | Server bind address | `0.0.0.0` |
| `SERVER_PORT` | Server listen port | `8080` |
| `SERVER_STORE_TYPE` | Storage type (`etcd`, `crd`, `sqlite`) | `etcd` |
| `SERVER_STORE_ETCD_HOST` | etcd server hostname | `localhost` |
| `SERVER_STORE_ETCD_PORT` | etcd server port | `2379` |
| `SERVER_STORE_ETCD_USERNAME` | etcd username for authentication | `` |
| `SERVER_STORE_ETCD_PASSWORD` | etcd password for authentication | `` |
| `SERVER_STORE_ETCD_DIAL_TIMEOUT` | Timeout for connecting to etcd | `5s` |
| `SERVER_STORE_ETCD_WORKLOADS_PREFIX` | etcd key prefix for workloads | `/discovery/workloads/` |
| `SERVER_STORE_CRD_NAMESPACE` | Namespace to read workloads from | `default` |
| `SERVER_STORE_CRD_KUBECONFIG` | Path to kubeconfig file (empty for in-cluster) | `` |
| `SERVER_STORE_CRD_RESYNC_PERIOD` | How often to resync the cache from the API server | `30s` |
| `SERVER_STORE_SQLITE_PATH` | SQLite database file shared with discovery (`~` expands to home) | `~/.agntcy/dir-runtime/workloads.db` |


## gRPC API

The server exposes a gRPC API defined in `proto/agntcy/dir/runtime/v1/discovery_service.proto`.

### GetWorkload

Get a specific workload by ID, name, or hostname.

```bash
grpcurl -plaintext -d '{"id": "my-service"}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/GetWorkload
```

### ListWorkloads

Stream all workloads with optional label filters. Labels support regex patterns.

```bash
# List all workloads
grpcurl -plaintext -d '{}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads

# Filter by labels (supports regex)
grpcurl -plaintext -d '{"labels": {"org.agntcy/agent-type": "a2a"}}' \
  localhost:8080 agntcy.dir.runtime.v1.DiscoveryService/ListWorkloads
```
