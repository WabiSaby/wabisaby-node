// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package container

import (
	"context"
	"log/slog"
	"os"

	"github.com/wabisaby/wabisaby-node/internal/agent"
	"github.com/wabisaby/wabisaby-node/internal/config"
	"github.com/wabisaby/wabisaby-node/internal/ipfs"
	"go.uber.org/fx"
)

// ProvideNodeLogger provides a structured logger for the node based on config.
func ProvideNodeLogger(cfg *config.NodeConfig) *slog.Logger {
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

// ProvideIPFSManager provides the IPFS lifecycle manager.
func ProvideIPFSManager(cfg *config.NodeConfig, logger *slog.Logger) *ipfs.IPFSManager {
	managerCfg := ipfs.ManagerConfig{
		BinaryPath: "", // Auto-detect
		DataDir:    cfg.IPFSDataDir,
		APIURL:     cfg.IPFSAPIURL,
		Logger:     logger,
	}
	return ipfs.NewIPFSManager(managerCfg)
}

// ProvideNodeAgent provides the storage node agent.
func ProvideNodeAgent(
	cfg *config.NodeConfig,
	ipfsManager *ipfs.IPFSManager,
	logger *slog.Logger,
) *agent.Agent {
	agentCfg := agent.AgentConfig{
		CoordinatorAddr:   cfg.CoordinatorAddr,
		AuthToken:         cfg.AuthToken,
		IPFSAPIURL:        cfg.IPFSAPIURL,
		IPFSDataDir:       cfg.IPFSDataDir,
		NodeName:          cfg.NodeName,
		Region:            cfg.Region,
		WalletAddress:     cfg.WalletAddress,
		CapacityBytes:     cfg.StorageCapacityGB * 1024 * 1024 * 1024,
		HeartbeatInterval: cfg.HeartbeatInterval,
		PollInterval:      cfg.PollInterval,
	}
	return agent.NewAgent(agentCfg, ipfsManager, logger)
}

// StartNodeAgent starts the node agent and handles graceful shutdown.
func StartNodeAgent(
	lc fx.Lifecycle,
	cfg *config.NodeConfig,
	nodeAgent *agent.Agent,
	logger *slog.Logger,
) {
	logger.Info("starting WabiSaby storage node", "name", cfg.NodeName, "version", "1.0.0")

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start agent in a goroutine - it will block until ctx.Done()
			go func() {
				if err := nodeAgent.Start(ctx); err != nil {
					logger.Error("agent stopped with error", "error", err)
					os.Exit(1)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("storage node shutdown successful")
			return nil
		},
	})
}

// NodeModule provides all node-specific dependencies.
// This module is standalone and does not require CommonModule since
// the node is community-deployable and doesn't need core app dependencies.
var NodeModule = fx.Module("node",
	fx.Provide(
		config.LoadNodeConfig,
		ProvideNodeLogger,
		ProvideIPFSManager,
		ProvideNodeAgent,
	),
	fx.Invoke(
		StartNodeAgent,
	),
)
