// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	nodepb "github.com/wabisaby/wabisaby-protos-go/go/node"
	"github.com/wabisaby/wabisaby-node/internal/ipfs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Agent manages the communication and coordination between a storage node and the network coordinator.
// It handles node registration, periodic heartbeats, polling and execution of pinning tasks,
// and maintains status reporting logic.
type Agent struct {
	nodeID      string                       // Unique ID assigned by coordinator after registration
	peerID      string                       // IPFS peer ID of this node
	config      AgentConfig                  // Configuration for the Agent
	client      nodepb.NodeCoordinatorClient // gRPC client for NodeCoordinator API
	conn        *grpc.ClientConn             // Underlying gRPC connection
	logger      *slog.Logger                 // Logger for agent events
	ipfs        *ipfs.Client                  // Client for local IPFS API
	ipfsManager *ipfs.IPFSManager            // IPFS lifecycle manager
	startTime   time.Time                    // Time when the agent started (for uptime tracking)
}

// AgentConfig encapsulates the configuration settings used to initialize an Agent.
type AgentConfig struct {
	CoordinatorAddr   string        // Network address of the coordinator gRPC endpoint
	AuthToken         string        // JWT token for authentication
	IPFSAPIURL        string        // HTTP API base URL for local IPFS node
	IPFSDataDir       string        // IPFS data directory
	NodeName          string        // Human-readable name for this node
	Region            string        // Region identifier for this node
	WalletAddress     string        // Associated wallet address
	CapacityBytes     int64         // Storage capacity of the node (in bytes)
	HeartbeatInterval time.Duration // How often heartbeats are sent to coordinator
	PollInterval      time.Duration // How often to poll for new tasks
}

// NewAgent creates a new storage node agent with the provided configuration and logger.
// It does not perform any network operations or side effects.
func NewAgent(cfg AgentConfig, ipfsManager *ipfs.IPFSManager, logger *slog.Logger) *Agent {
	return &Agent{
		config:      cfg,
		ipfsManager: ipfsManager,
		logger:      logger,
	}
}

// Start begins the main lifecycle of the agent. It connects to the coordinator, registers the node,
// and launches background goroutines for periodic heartbeats and pinning task polling.
// This call is blocking until the context is canceled, at which time it closes the gRPC connection.
func (a *Agent) Start(ctx context.Context) error {
	// 1. Initialize and start IPFS
	if err := a.setupIPFS(ctx); err != nil {
		return fmt.Errorf("failed to setup IPFS: %w", err)
	}

	// 2. Get peer ID and multiaddresses
	peerID, multiaddrs, err := a.ipfsManager.GetPeerInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get peer info: %w", err)
	}
	a.peerID = peerID

	// 3. Connect to coordinator via gRPC with auth
	conn, err := grpc.NewClient(
		a.config.CoordinatorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to coordinator: %w", err)
	}
	a.conn = conn
	a.client = nodepb.NewNodeCoordinatorClient(conn)
	a.ipfs = ipfs.NewClient(a.config.IPFSAPIURL)

	// 4. Register with coordinator
	if err := a.register(ctx, multiaddrs); err != nil {
		return fmt.Errorf("initial registration failed: %w", err)
	}

	a.startTime = time.Now()
	a.logger.Info("node agent started and registered", "node_id", a.nodeID, "peer_id", a.peerID)

	// 5. Connect to peers
	if err := a.connectToPeers(ctx); err != nil {
		a.logger.Warn("failed to connect to some peers", "error", err)
	}

	// 6. Start background workers
	go a.heartbeatLoop(ctx)
	go a.taskLoop(ctx)

	<-ctx.Done()

	// Stop IPFS daemon on shutdown
	if err := a.ipfsManager.StopDaemon(ctx); err != nil {
		a.logger.Warn("failed to stop IPFS daemon", "error", err)
	}

	return a.conn.Close()
}

// setupIPFS initializes IPFS: installs, initializes repo, configures private network, and starts daemon.
func (a *Agent) setupIPFS(ctx context.Context) error {
	a.logger.Info("setting up IPFS")

	// Ensure IPFS is installed
	if err := a.ipfsManager.EnsureInstalled(ctx); err != nil {
		return fmt.Errorf("failed to ensure IPFS is installed: %w", err)
	}

	// Initialize repository
	if err := a.ipfsManager.InitializeRepo(ctx); err != nil {
		return fmt.Errorf("failed to initialize IPFS repository: %w", err)
	}

	// Configure private network (swarm key and bootstrap peers)
	// For now, use empty swarm key and bootstrap peers - can be enhanced later
	if err := a.ipfsManager.ConfigurePrivateNetwork(ctx, "", []string{}); err != nil {
		return fmt.Errorf("failed to configure private network: %w", err)
	}

	// Start daemon
	if err := a.ipfsManager.StartDaemon(ctx); err != nil {
		return fmt.Errorf("failed to start IPFS daemon: %w", err)
	}

	a.logger.Info("IPFS setup completed")
	return nil
}

// register performs a registration with the network coordinator, exchanging node information and
// receiving a node ID which is persisted in the Agent instance.
// Returns an error if registration is unsuccessful or coordinator rejects the operation.
func (a *Agent) register(ctx context.Context, multiaddrs []string) error {
	// Add auth token to metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + a.config.AuthToken,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	resp, err := a.client.Register(ctx, &nodepb.RegisterRequest{
		PeerId:               a.peerID,
		Name:                 a.config.NodeName,
		Region:               a.config.Region,
		IpfsMultiaddrs:       multiaddrs,
		StorageCapacityBytes: a.config.CapacityBytes,
		WalletAddress:        a.config.WalletAddress,
		AuthToken:            a.config.AuthToken,
	})
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("coordinator rejected registration: %s", resp.Error)
	}

	a.nodeID = resp.NodeId
	return nil
}

// connectToPeers connects to peers returned by the coordinator.
func (a *Agent) connectToPeers(ctx context.Context) error {
	// Add auth token to metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + a.config.AuthToken,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Get peers from coordinator
	resp, err := a.client.GetPeers(ctx, &nodepb.GetPeersRequest{
		NodeId: a.nodeID,
	})
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	if resp.Error != "" {
		return fmt.Errorf("coordinator error: %s", resp.Error)
	}

	// Connect to each peer
	connected := 0
	for _, peer := range resp.Peers {
		for _, multiaddr := range peer.Multiaddrs {
			if err := a.ipfsManager.ConnectToPeer(ctx, multiaddr); err != nil {
				a.logger.Warn("failed to connect to peer", "peer", multiaddr, "error", err)
				continue
			}
			connected++
		}
	}

	a.logger.Info("connected to peers", "connected", connected, "total", len(resp.Peers))
	return nil
}

// heartbeatLoop periodically sends heartbeat messages to the coordinator, reporting current storage usage and other statistics.
// Runs as a background goroutine until context cancellation.
func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Gather storage usage stats from local IPFS
			stat, err := a.ipfs.RepoStat(ctx)
			storageUsed := int64(0)
			if err == nil && stat != nil {
				if stat.RepoSize > uint64(math.MaxInt64) {
					storageUsed = math.MaxInt64
				} else {
					storageUsed = int64(stat.RepoSize)
				}
			}
			uptimeSeconds := int64(time.Since(a.startTime).Seconds())
			// Add auth token to metadata
			md := metadata.New(map[string]string{
				"authorization": "Bearer " + a.config.AuthToken,
			})
			heartbeatCtx := metadata.NewOutgoingContext(ctx, md)

			_, err = a.client.Heartbeat(heartbeatCtx, &nodepb.HeartbeatRequest{
				NodeId:           a.nodeID,
				StorageUsedBytes: storageUsed,
				UptimeSeconds:    uptimeSeconds,
			})
			if err != nil {
				a.logger.Warn("heartbeat failed", "error", err)
			}
		}
	}
}

// taskLoop periodically polls the coordinator for new pinning tasks and spins up goroutines to process each task as they are received.
// Runs as a background goroutine until context cancellation.
func (a *Agent) taskLoop(ctx context.Context) {
	ticker := time.NewTicker(a.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Add auth token to metadata
			md := metadata.New(map[string]string{
				"authorization": "Bearer " + a.config.AuthToken,
			})
			taskCtx := metadata.NewOutgoingContext(ctx, md)

			// Poll coordinator for outstanding pinning tasks
			resp, err := a.client.GetPinTasks(taskCtx, &nodepb.GetPinTasksRequest{
				NodeId: a.nodeID,
			})
			if err != nil {
				a.logger.Warn("failed to poll for tasks", "error", err)
				continue
			}

			for _, task := range resp.Tasks {
				a.logger.Info("received pin task", "task_id", task.TaskId, "cid", task.Cid)
				go a.processTask(ctx, task)
			}
		}
	}
}

// processTask handles the full lifecycle for a single pinning task: Execute the pin via IPFS, then report status back to coordinator.
func (a *Agent) processTask(ctx context.Context, task *nodepb.PinTask) {
	// Attempt pin operation on local IPFS node
	a.logger.Info("pinning content", "cid", task.Cid)

	err := a.ipfs.Pin(ctx, task.Cid)

	status := nodepb.ReportPinStatusRequest_PIN_STATUS_PINNED
	if err != nil {
		a.logger.Error("failed to pin content", "cid", task.Cid, "error", err)
		status = nodepb.ReportPinStatusRequest_PIN_STATUS_FAILED
	}

	// Add auth token to metadata
	md := metadata.New(map[string]string{
		"authorization": "Bearer " + a.config.AuthToken,
	})
	reportCtx := metadata.NewOutgoingContext(ctx, md)

	// Notify coordinator of task outcome
	_, err = a.client.ReportPinStatus(reportCtx, &nodepb.ReportPinStatusRequest{
		NodeId: a.nodeID,
		TaskId: task.TaskId,
		Status: status,
	})
	if err != nil {
		a.logger.Error("failed to report pin status", "task_id", task.TaskId, "error", err)
	} else if status == nodepb.ReportPinStatusRequest_PIN_STATUS_PINNED {
		a.logger.Info("pin task completed", "task_id", task.TaskId)
	}
}
