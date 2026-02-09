// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package config

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/viper"
)

// NodeConfig holds storage node configuration.
type NodeConfig struct {
	// Required fields
	AuthToken       string `mapstructure:"auth_token"`
	CoordinatorAddr string `mapstructure:"coordinator_addr"`

	// Optional fields (auto-detected if not provided)
	IPFSAPIURL        string        `mapstructure:"ipfs_api_url"`
	IPFSDataDir       string        `mapstructure:"ipfs_data_dir"`
	NodeName          string        `mapstructure:"node_name"`
	Region            string        `mapstructure:"region"`
	WalletAddress     string        `mapstructure:"wallet_address"`
	StorageCapacityGB int64         `mapstructure:"storage_capacity_gb"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	PollInterval      time.Duration `mapstructure:"poll_interval"`
	LogLevel          string        `mapstructure:"log_level"`
}

// LoadNodeConfig loads storage node configuration from config file and environment variables.
func LoadNodeConfig() *NodeConfig {
	viper.SetConfigName("node")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetEnvPrefix("WABISABY_NODE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Storage node defaults
	viper.SetDefault("coordinator_addr", "localhost:50051")
	viper.SetDefault("ipfs_api_url", "http://localhost:5001")
	viper.SetDefault("node_name", "wabisaby-community-node")
	viper.SetDefault("storage_capacity_gb", 100)
	viper.SetDefault("heartbeat_interval", 1*time.Minute)
	viper.SetDefault("poll_interval", 30*time.Second)
	viper.SetDefault("log_level", "info")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Node config file not found, using defaults and environment variables")
		} else {
			log.Printf("Error reading config file: %s", err)
		}
	}

	var config NodeConfig
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	// Auto-detect missing values
	if config.AuthToken == "" {
		config.AuthToken = os.Getenv("WABISABY_AUTH_TOKEN")
	}
	if config.CoordinatorAddr == "" {
		config.CoordinatorAddr = os.Getenv("WABISABY_COORDINATOR_ADDR")
	}

	// Auto-detect storage capacity if not provided
	if config.StorageCapacityGB == 0 {
		capacityGB := detectStorageCapacity()
		if capacityGB > 0 {
			config.StorageCapacityGB = capacityGB
		} else {
			config.StorageCapacityGB = 100 // Default fallback
		}
	}

	// Auto-detect region if not provided
	if config.Region == "" {
		config.Region = detectRegion()
	}

	// Auto-generate node name if not provided
	if config.NodeName == "" {
		config.NodeName = generateNodeName()
	}

	// Set defaults for optional fields
	if config.IPFSAPIURL == "" {
		config.IPFSAPIURL = "http://localhost:5001"
	}
	if config.IPFSDataDir == "" {
		homeDir, _ := os.UserHomeDir()
		config.IPFSDataDir = filepath.Join(homeDir, ".wabisaby", "ipfs")
	}

	return &config
}

// detectStorageCapacity detects available disk space and returns capacity in GB.
// Uses 80% of available space to leave room for OS.
func detectStorageCapacity() int64 {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0
	}

	if err := syscall.Statfs(wd, &stat); err != nil {
		return 0
	}

	// Calculate available bytes (blocks * block size)
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	// Use 80% of available space
	usableBytes := availableBytes * 80 / 100
	// Convert to GB
	capacityGB := int64(usableBytes / (1024 * 1024 * 1024))

	return capacityGB
}

// detectRegion detects the region based on timezone or returns a default.
func detectRegion() string {
	// Try to get timezone
	tz := os.Getenv("TZ")
	if tz != "" {
		// Extract region from timezone (e.g., "America/New_York" -> "us-east")
		if strings.Contains(tz, "America") {
			return "us"
		}
		if strings.Contains(tz, "Europe") {
			return "eu"
		}
		if strings.Contains(tz, "Asia") {
			return "asia"
		}
	}

	// Default to unknown if can't detect
	return "unknown"
}

// generateNodeName generates a unique node name based on hostname.
func generateNodeName() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	currentUser, err := user.Current()
	username := "user"
	if err == nil {
		username = currentUser.Username
	}

	// Format: wabisaby-node-{hostname}-{username}
	return strings.ToLower(strings.ReplaceAll(
		strings.Join([]string{"wabisaby-node", hostname, username}, "-"),
		" ", "-",
	))
}
