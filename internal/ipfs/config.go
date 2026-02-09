// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package ipfs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	BootstrapKey = "Bootstrap"
)

// PrivateNetworkConfig holds configuration for IPFS private network.
type PrivateNetworkConfig struct {
	SwarmKey       string   // Swarm key for private network
	BootstrapPeers []string // Bootstrap peer addresses
	NetworkName    string   // Network identifier
}

// ConfigureSwarmKey writes the swarm key to the IPFS config directory.
func ConfigureSwarmKey(repoPath, swarmKey string) error {
	if swarmKey == "" {
		return nil
	}

	swarmKeyPath := filepath.Join(repoPath, "swarm.key")
	if err := os.WriteFile(swarmKeyPath, []byte(swarmKey), 0o600); err != nil {
		return fmt.Errorf("failed to write swarm key: %w", err)
	}

	return nil
}

// ConfigureBootstrapPeers updates the bootstrap peers list in IPFS config.
func ConfigureBootstrapPeers(repoPath string, bootstrapPeers []string) error {
	configPath := filepath.Join(repoPath, "config")

	// Read existing config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read IPFS config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse IPFS config: %w", err)
	}

	if _, ok := config[BootstrapKey].([]interface{}); ok {
		config[BootstrapKey] = []interface{}{}
	}

	// Add new bootstrap peers
	bootstrapList := make([]interface{}, len(bootstrapPeers))
	for i, peer := range bootstrapPeers {
		bootstrapList[i] = peer
	}
	config[BootstrapKey] = bootstrapList

	// Write updated config
	updatedConfig, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal IPFS config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedConfig, 0o644); err != nil {
		return fmt.Errorf("failed to write IPFS config: %w", err)
	}

	return nil
}

// RemovePublicBootstrap removes default public IPFS bootstrap nodes from config.
func RemovePublicBootstrap(repoPath string) error {
	configPath := filepath.Join(repoPath, "config")

	// Read existing config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read IPFS config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("failed to parse IPFS config: %w", err)
	}

	// Default public bootstrap nodes to remove
	publicBootstrap := []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zp4k9rH6pYwtyWg8qK1XpoyfpMc5gB",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi8CttYg2xU2J6N3FJ3Z1JvZ8JvZ8JvZ8",
		"/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	}

	// Get current bootstrap list
	bootstrap, ok := config["Bootstrap"].([]interface{})
	if !ok {
		return nil
	}

	// Filter out public bootstrap nodes
	filtered := []interface{}{}
	for _, addr := range bootstrap {
		addrStr, ok := addr.(string)
		if !ok {
			continue
		}

		isPublic := false
		for _, public := range publicBootstrap {
			if addrStr == public {
				isPublic = true
				break
			}
		}

		if !isPublic {
			filtered = append(filtered, addr)
		}
	}

	config["Bootstrap"] = filtered

	// Write updated config
	updatedConfig, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal IPFS config: %w", err)
	}

	if err := os.WriteFile(configPath, updatedConfig, 0o644); err != nil {
		return fmt.Errorf("failed to write IPFS config: %w", err)
	}

	return nil
}
