// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package ipfs

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wabisaby/wabisaby-node/internal/ipfsclient"
)

// IPFSManager manages the IPFS lifecycle: installation, initialization, configuration, and daemon management.
type IPFSManager struct {
	ipfsClient  *ipfsclient.IPFSClient
	binaryPath  string
	dataDir     string
	apiURL      string
	logger      *slog.Logger
	daemonCmd   *exec.Cmd
	daemonReady bool
}

// ManagerConfig holds configuration for the IPFS manager.
type ManagerConfig struct {
	BinaryPath string // Path to IPFS binary (if empty, will be auto-detected/installed)
	DataDir    string // IPFS data directory (default: ~/.wabisaby/ipfs)
	APIURL     string // IPFS API URL (default: http://localhost:5001)
	Logger     *slog.Logger
}

// NewIPFSManager creates a new IPFS manager.
func NewIPFSManager(cfg ManagerConfig) *IPFSManager {
	if cfg.DataDir == "" {
		homeDir, _ := os.UserHomeDir()
		cfg.DataDir = filepath.Join(homeDir, ".wabisaby", "ipfs")
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:5001"
	}

	return &IPFSManager{
		binaryPath: cfg.BinaryPath,
		dataDir:    cfg.DataDir,
		apiURL:     cfg.APIURL,
		logger:     cfg.Logger,
	}
}

// EnsureInstalled checks if IPFS binary exists and downloads it if missing.
func (m *IPFSManager) EnsureInstalled(ctx context.Context) error {
	if m.binaryPath != "" {
		if _, err := os.Stat(m.binaryPath); err == nil {
			m.logger.Info("IPFS binary found", "path", m.binaryPath)
			return nil
		}
	}

	// Try to find IPFS in PATH
	if path, err := exec.LookPath("ipfs"); err == nil {
		m.binaryPath = path
		m.logger.Info("IPFS binary found in PATH", "path", path)
		return nil
	}

	// Download IPFS binary
	m.logger.Info("IPFS binary not found, downloading...")
	return m.downloadIPFS(ctx)
}

// InitializeRepo initializes the IPFS repository if it doesn't exist.
func (m *IPFSManager) InitializeRepo(ctx context.Context) error {
	repoPath := filepath.Join(m.dataDir, ".ipfs")
	configPath := filepath.Join(repoPath, "config")

	// Check if repo already exists
	if _, err := os.Stat(configPath); err == nil {
		m.logger.Info("IPFS repository already initialized", "path", repoPath)
		return nil
	}

	// Create data directory
	if err := os.MkdirAll(m.dataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Set IPFS_PATH environment variable
	env := os.Environ()
	env = append(env, fmt.Sprintf("IPFS_PATH=%s", repoPath))

	// Run ipfs init
	cmd := exec.CommandContext(ctx, m.binaryPath, "init")
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	m.logger.Info("Initializing IPFS repository", "path", repoPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize IPFS repository: %w", err)
	}

	m.logger.Info("IPFS repository initialized successfully")
	return nil
}

// ConfigurePrivateNetwork configures IPFS for private network with swarm key and bootstrap peers.
func (m *IPFSManager) ConfigurePrivateNetwork(ctx context.Context, swarmKey string, bootstrapPeers []string) error {
	repoPath := filepath.Join(m.dataDir, ".ipfs")
	swarmKeyPath := filepath.Join(repoPath, "swarm.key")

	// Write swarm key
	if swarmKey != "" {
		if err := os.WriteFile(swarmKeyPath, []byte(swarmKey), 0o600); err != nil {
			return fmt.Errorf("failed to write swarm key: %w", err)
		}
		m.logger.Info("Swarm key configured", "path", swarmKeyPath)
	}

	// Configure bootstrap peers
	if len(bootstrapPeers) > 0 {
		env := os.Environ()
		env = append(env, fmt.Sprintf("IPFS_PATH=%s", repoPath))

		// Remove default bootstrap peers
		cmd := exec.CommandContext(ctx, m.binaryPath, "bootstrap", "rm", "--all")
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			m.logger.Warn("Failed to remove default bootstrap peers", "error", err)
		}

		// Add custom bootstrap peers
		for _, peer := range bootstrapPeers {
			cmd := exec.CommandContext(ctx, m.binaryPath, "bootstrap", "add", peer)
			cmd.Env = env
			if err := cmd.Run(); err != nil {
				m.logger.Warn("Failed to add bootstrap peer", "peer", peer, "error", err)
			} else {
				m.logger.Info("Added bootstrap peer", "peer", peer)
			}
		}
	}

	return nil
}

// StartDaemon starts the IPFS daemon in the background.
func (m *IPFSManager) StartDaemon(ctx context.Context) error {
	if m.daemonCmd != nil {
		// Check if daemon is still running
		if m.daemonCmd.Process != nil {
			if err := m.daemonCmd.Process.Signal(os.Signal(nil)); err == nil {
				m.logger.Info("IPFS daemon already running")
				return nil
			}
		}
	}

	repoPath := filepath.Join(m.dataDir, ".ipfs")
	env := os.Environ()
	env = append(env, fmt.Sprintf("IPFS_PATH=%s", repoPath))

	cmd := exec.CommandContext(ctx, m.binaryPath, "daemon", "--enable-pubsub-experiment")
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	m.logger.Info("Starting IPFS daemon", "api_url", m.apiURL)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start IPFS daemon: %w", err)
	}

	m.daemonCmd = cmd
	m.daemonReady = false

	// Wait for daemon to be ready
	go m.waitForDaemon(ctx)

	return nil
}

// waitForDaemon polls the IPFS API until the daemon is ready.
func (m *IPFSManager) waitForDaemon(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)
	client := ipfsclient.NewIPFSClient(m.apiURL)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			m.logger.Error("IPFS daemon failed to start within timeout")
			return
		case <-ticker.C:
			if _, err := client.Version(ctx); err == nil {
				m.daemonReady = true
				m.ipfsClient = client
				m.logger.Info("IPFS daemon is ready")
				return
			}
		}
	}
}

// StopDaemon gracefully stops the IPFS daemon.
func (m *IPFSManager) StopDaemon(ctx context.Context) error {
	if m.daemonCmd == nil || m.daemonCmd.Process == nil {
		return nil
	}

	m.logger.Info("Stopping IPFS daemon")
	if err := m.daemonCmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to signal IPFS daemon: %w", err)
	}

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- m.daemonCmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("IPFS daemon exited with error: %w", err)
		}
		m.logger.Info("IPFS daemon stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown fails
		m.logger.Warn("IPFS daemon did not stop gracefully, forcing kill")
		return m.daemonCmd.Process.Kill()
	}
}

// GetPeerInfo returns the peer ID and multiaddresses of the local IPFS node.
func (m *IPFSManager) GetPeerInfo(ctx context.Context) (peerID string, multiaddrs []string, err error) {
	if m.ipfsClient == nil {
		m.ipfsClient = ipfsclient.NewIPFSClient(m.apiURL)
	}

	// Wait for daemon to be ready
	if !m.daemonReady {
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

	waitLoop:
		for {
			select {
			case <-ctx.Done():
				return "", nil, ctx.Err()
			case <-timeout:
				return "", nil, fmt.Errorf("IPFS daemon not ready")
			case <-ticker.C:
				if _, err := m.ipfsClient.Version(ctx); err == nil {
					m.daemonReady = true
					break waitLoop
				}
			}
		}
	}

	return m.ipfsClient.ID(ctx)
}

// ConnectToPeer connects to a peer using IPFS swarm connect.
func (m *IPFSManager) ConnectToPeer(ctx context.Context, peerAddr string) error {
	repoPath := filepath.Join(m.dataDir, ".ipfs")
	env := os.Environ()
	env = append(env, fmt.Sprintf("IPFS_PATH=%s", repoPath))

	cmd := exec.CommandContext(ctx, m.binaryPath, "swarm", "connect", peerAddr)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
	}

	m.logger.Info("Connected to peer", "peer", peerAddr)
	return nil
}

// downloadIPFS downloads the IPFS binary for the current platform.
func (m *IPFSManager) downloadIPFS(ctx context.Context) error {
	// Determine platform
	var platform, arch string
	switch runtime.GOOS {
	case "linux":
		platform = "linux"
	case "darwin":
		platform = "darwin"
	case "windows":
		platform = "windows"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		return fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	// For now, return an error asking user to install IPFS manually
	// Full implementation would download from GitHub releases
	return fmt.Errorf("IPFS binary not found. Please install IPFS (kubo) manually or ensure it's in your PATH. Platform: %s/%s", platform, arch)
}
