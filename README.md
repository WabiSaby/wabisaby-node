# WabiSaby Storage Node

Community-deployable storage node for the WabiSaby decentralized storage network.

## Overview

The WabiSaby Storage Node is a standalone binary that allows anyone to contribute storage to the WabiSaby network. It runs an IPFS node, registers with the network coordinator, and handles content pinning tasks automatically.

## Quick Start

### Prerequisites

- Go 1.24+ (for building from source)
- IPFS (kubo) installed and in PATH, or the node will attempt to install it

### Build

```bash
make build
```

### Run

```bash
# Minimal: provide auth token and coordinator address
export WABISABY_AUTH_TOKEN="your-jwt-token"
export WABISABY_NODE_COORDINATOR_ADDR="coordinator.wabisaby.com:50051"
./bin/wabisaby-node

# Or use a config file
cp config/node.yaml ./node.yaml
# Edit node.yaml with your settings
./bin/wabisaby-node
```

### Configuration

**Required (env fallbacks):**
- `auth.token` / `WABISABY_AUTH_TOKEN` - JWT for the coordinator
- `coordinator.address` / `WABISABY_COORDINATOR_ADDR` - Coordinator gRPC address

**Optional (with defaults or auto-detection):**
- `storage.capacity_gb` - 80% of available disk if unset
- `node.region` - From timezone
- `node.name` - From hostname + username
- `ipfs.api_url` - `http://localhost:5001`
- `ipfs.data_dir` - `~/.wabisaby/ipfs`

### Acquiring a token for the node

The coordinator expects a **valid JWT**. For **local dev** with Keycloak (e.g. WabiSaby devkit), from the devkit repo root:

1. Start Keycloak: `docker compose -f docker/docker-compose.yml up -d keycloak`
2. Create node user: `./docker/keycloak/create-node-user.sh node node` (or fix existing: `./docker/keycloak/fix-node-user.sh node`)
3. Get .env lines (node will refresh token automatically): `./docker/keycloak/get-node-token.sh --env node node` → paste into `.env`

See **[docker/keycloak/README.md](../../docker/keycloak/README.md)** for details and script options.

For **production**, use your app’s login flow or a service account; set `WABISABY_NODE_AUTH_TOKEN` or use `auth.refresh_token` + `auth.keycloak_token_url` for automatic refresh.

## Architecture

See [`docs/node-setup.md`](docs/node-setup.md) for detailed architecture documentation.

## Development

```bash
# Build
make build

# Clean
make clean

# Run tests
make test
```

## License

MIT License - see [LICENSE](LICENSE) for details.
