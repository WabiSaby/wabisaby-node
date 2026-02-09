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

See [`config/node.yaml`](config/node.yaml) for all available options.

**Required:**
- `auth_token` - JWT authentication token (or `WABISABY_AUTH_TOKEN` env var)
- `coordinator_addr` - Network coordinator gRPC address (or `WABISABY_NODE_COORDINATOR_ADDR` env var)

**Auto-detected (optional):**
- `storage_capacity_gb` - Defaults to 80% of available disk space
- `region` - Detected from timezone
- `node_name` - Generated from hostname + username
- `ipfs_api_url` - Defaults to `http://localhost:5001`
- `ipfs_data_dir` - Defaults to `~/.wabisaby/ipfs`

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
