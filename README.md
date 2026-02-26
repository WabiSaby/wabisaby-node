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

The coordinator expects a **valid JWT** (signed by your auth provider). How you get it depends on the environment.

#### Local development (Keycloak)

When running against a local network-coordinator that uses Keycloak (e.g. WabiSaby devkit):

1. **Start Keycloak** (if not already running), e.g. from the repo root:
   ```bash
   docker compose -f docker/docker-compose.yml up -d keycloak
   ```
2. **Create a user** in the `wabisaby` realm (Keycloak Admin UI: http://localhost:8180 → realm **wabisaby** → Users → Create user, set username/password and enable the user). Set a permanent password in the Credentials tab (turn **Temporary** off). If you get "Account is not fully set up" when requesting a token, run from the repo root: `./docker/keycloak/fix-node-user.sh YOUR_USERNAME` to clear required actions via the Admin API.
3. **Get tokens** (programmatic refresh recommended so the node refreshes before expiry):
   ```bash
   # From devkit repo root: print .env lines (refresh_token + keycloak URL) so the node refreshes automatically
   ./docker/keycloak/get-node-token.sh --env node node
   ```
   Paste the output into your **.env** (devkit app root). The node will use the refresh token to obtain and refresh the access token in the background.

   Alternatively, for a one-off access token only (expires; you must re-run and update .env when it expires):
   ```bash
   ./docker/keycloak/get-node-token.sh node node
   ```
   Then set in .env: `WABISABY_NODE_AUTH_TOKEN=<paste-the-token-here>`

#### Production

In production, tokens are typically obtained via your app’s login flow (e.g. OAuth/OIDC in the frontend). Use the same **access token** (Bearer token) that the app uses for authenticated API calls. Copy it from the app (e.g. from the frontend after login) and set `WABISABY_NODE_AUTH_TOKEN` or `auth.token` for the node. For long‑running nodes, consider a dedicated service account or token refresh flow if your provider supports it.

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
