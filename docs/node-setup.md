# Automated Node Setup Architecture

## Overview

The WabiSaby node setup has been designed to provide a zero-configuration installation experience. Users only need to provide an authentication token; everything else (IPFS installation, configuration, network setup, peer discovery) happens automatically.

## User Experience

### Installation Flow

1. User downloads/installs WabiSaby node binary
2. User runs: `wabisaby-node --auth-token <token>` (or sets `WABISABY_AUTH_TOKEN` env var)
3. Node automatically:
   - Installs/configures IPFS if needed
   - Configures IPFS for WabiSaby private network
   - Authenticates with coordinator using token
   - Registers node linked to user's account
   - Gets peer list from coordinator
   - Connects to peers automatically
   - Starts serving storage

### Configuration Requirements

**Required:**
- `WABISABY_AUTH_TOKEN` - JWT token for authentication
- `WABISABY_COORDINATOR_ADDR` - Coordinator gRPC address

**Auto-detected:**
- Storage capacity (80% of available disk space)
- Region (based on timezone)
- Node name (generated from hostname + username)
- IPFS data directory (`~/.wabisaby/ipfs`)
- IPFS API URL (`http://localhost:5001`)

## Architecture

### Component Separation

```
internal/
├── ipfsclient/              # Minimal IPFS HTTP API client
│   └── client.go            # Version, ID, RepoStat, Pin
│
├── ipfs/                    # Node IPFS lifecycle management
│   ├── manager.go           # Install, init, configure, start
│   └── config.go            # Private network config
│
├── agent/                   # Node agent
│   └── agent.go             # Registration, heartbeat, task polling
│
├── config/                  # Node configuration
│   └── config.go            # NodeConfig, LoadNodeConfig, auto-detection
│
└── container/               # Dependency injection (fx)
    └── node.go              # NodeModule wiring
```

### Key Components

#### 1. IPFS Manager (`internal/ipfs/manager.go`)

Manages the complete IPFS lifecycle:

- **EnsureInstalled()** - Checks if IPFS binary exists, downloads if missing
- **InitializeRepo()** - Runs `ipfs init` if repository doesn't exist
- **ConfigurePrivateNetwork()** - Sets up swarm key and bootstrap peers
- **StartDaemon()** - Starts IPFS daemon process in background
- **StopDaemon()** - Gracefully stops IPFS daemon
- **GetPeerInfo()** - Gets peer ID and multiaddresses via IPFSClient
- **ConnectToPeer()** - Connects to a peer using IPFS swarm connect

#### 2. IPFS Client (`internal/ipfsclient/client.go`)

Minimal HTTP client for the IPFS API, containing only methods the node needs:

- **Version()** - Check IPFS daemon readiness
- **ID()** - Get peer ID and multiaddresses
- **RepoStat()** - Get repository storage statistics
- **Pin()** - Pin a CID to local storage

#### 3. Agent (`internal/agent/agent.go`)

Manages communication with the network coordinator:

- **Start()** - Full lifecycle: setup IPFS, register, heartbeat, poll tasks
- **register()** - Register with coordinator via gRPC
- **heartbeatLoop()** - Periodic heartbeats with storage stats
- **taskLoop()** - Poll for and execute pinning tasks
- **processTask()** - Pin content and report status

## Data Flow

### Node Registration Flow

```
1. User starts node with auth token
   |
2. IPFS Manager:
   - Ensures IPFS is installed
   - Initializes repository
   - Configures private network
   - Starts daemon
   |
3. Agent gets peer ID and multiaddresses
   |
4. Agent connects to coordinator (with auth token in metadata)
   |
5. Coordinator validates token, extracts user ID
   |
6. Coordinator registers node (linked to user account)
   |
7. Agent gets peer list from coordinator
   |
8. Agent connects to peers via IPFS swarm connect
   |
9. Node is ready to serve storage
```

### Peer Discovery Flow

```
1. Agent calls GetPeers RPC (with auth token)
   |
2. Coordinator queries active nodes
   |
3. Coordinator filters out requesting node
   |
4. Coordinator returns peer list with multiaddresses
   |
5. Agent connects to each peer via IPFS swarm connect
```

## Configuration Auto-Detection

### Storage Capacity

- Uses `syscall.Statfs` to get available disk space
- Calculates 80% of available space (leaves room for OS)
- Converts to GB
- Defaults to 100GB if detection fails

### Region Detection

- Reads `TZ` environment variable
- Maps timezone to region:
  - `America/*` -> `us`
  - `Europe/*` -> `eu`
  - `Asia/*` -> `asia`
- Defaults to `unknown` if can't detect

### Node Name Generation

- Format: `wabisaby-node-{hostname}-{username}`
- Example: `wabisaby-node-myserver-john`
- Lowercase, spaces replaced with hyphens

## Error Handling

### IPFS Installation Failure

- Clear error message asking user to install IPFS manually
- Provides platform/architecture information
- Suggests checking PATH

### Daemon Startup Failure

- Retries with exponential backoff (future enhancement)
- Logs detailed error messages
- Exits with clear error code

### Registration Failure

- Logs error with details
- Exits with error code
- User can retry with corrected configuration

### Peer Connection Failure

- Logs warning for each failed connection
- Continues with successful connections
- Does not block node operation

## Security Considerations

### Authentication

- All gRPC calls require valid JWT token
- Token validated using JWKS (public key)
- User ID extracted and linked to node
- No token stored in plain text (only in memory)

### Private Network

- Swarm key configured for private network
- Public bootstrap nodes removed
- Only connects to known peers from coordinator

### Network Isolation

- Nodes only connect to peers in WabiSaby network
- No connection to public IPFS network
- All traffic authenticated

## Future Enhancements

1. **IPFS Binary Download** - Automatically download IPFS binary from GitHub releases
2. **Swarm Key Management** - Retrieve swarm key from coordinator on first setup
3. **Bootstrap Peer Management** - Get bootstrap peers from coordinator
4. **Health Monitoring** - Enhanced health checks and automatic recovery
5. **Region Auto-Detection** - Use IP geolocation for more accurate region detection
6. **Retry Logic** - Exponential backoff for daemon startup and peer connections
