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

// NodeConfig holds storage node configuration (nested structure for node.yaml).
type NodeConfig struct {
	Auth        AuthConfig        `mapstructure:"auth"`
	Coordinator CoordinatorConfig `mapstructure:"coordinator"`
	IPFS        IPFSConfig        `mapstructure:"ipfs"`
	Node        NodeIdentityConfig `mapstructure:"node"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Intervals   IntervalsConfig   `mapstructure:"intervals"`
	Log         LogConfig         `mapstructure:"log"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Token              string `mapstructure:"token"`                // JWT access token (or use refresh_token for programmatic refresh)
	RefreshToken       string `mapstructure:"refresh_token"`         // Keycloak refresh token; if set with keycloak_token_url, node will refresh access token automatically
	KeycloakTokenURL   string `mapstructure:"keycloak_token_url"`    // Keycloak token endpoint, e.g. http://localhost:8180/realms/wabisaby/protocol/openid-connect/token
	KeycloakClientID   string `mapstructure:"keycloak_client_id"`    // OIDC client id for token refresh (default: wabisaby-api)
}

// CoordinatorConfig holds coordinator connection settings.
type CoordinatorConfig struct {
	Address string `mapstructure:"address"`
}

// IPFSConfig holds IPFS daemon settings.
type IPFSConfig struct {
	APIURL  string `mapstructure:"api_url"`
	DataDir string `mapstructure:"data_dir"`
}

// NodeIdentityConfig holds node identity (name, region, wallet).
type NodeIdentityConfig struct {
	Name          string `mapstructure:"name"`
	Region        string `mapstructure:"region"`
	WalletAddress string `mapstructure:"wallet_address"`
}

// StorageConfig holds storage capacity settings.
type StorageConfig struct {
	CapacityGB int64 `mapstructure:"capacity_gb"`
}

// IntervalsConfig holds heartbeat and poll intervals.
type IntervalsConfig struct {
	Heartbeat time.Duration `mapstructure:"heartbeat"`
	Poll      time.Duration `mapstructure:"poll"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `mapstructure:"level"`
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

	// Nested defaults (viper uses dot for nesting)
	viper.SetDefault("coordinator.address", "localhost:50052")
	viper.SetDefault("ipfs.api_url", "http://localhost:5001")
	viper.SetDefault("node.name", "wabisaby-community-node")
	viper.SetDefault("storage.capacity_gb", 100)
	viper.SetDefault("intervals.heartbeat", 1*time.Minute)
	viper.SetDefault("intervals.poll", 30*time.Second)
	viper.SetDefault("log.level", "info")

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

	// Auth: fallback to legacy env
	if config.Auth.Token == "" {
		config.Auth.Token = os.Getenv("WABISABY_AUTH_TOKEN")
	}
	if config.Auth.RefreshToken == "" {
		config.Auth.RefreshToken = os.Getenv("WABISABY_NODE_AUTH_REFRESH_TOKEN")
	}
	if config.Auth.KeycloakTokenURL == "" {
		config.Auth.KeycloakTokenURL = os.Getenv("WABISABY_NODE_KEYCLOAK_TOKEN_URL")
	}
	if config.Auth.KeycloakClientID == "" {
		config.Auth.KeycloakClientID = "wabisaby-api"
	}
	if config.Coordinator.Address == "" {
		config.Coordinator.Address = os.Getenv("WABISABY_COORDINATOR_ADDR")
	}

	// Auto-detect storage capacity if not provided
	if config.Storage.CapacityGB == 0 {
		capacityGB := detectStorageCapacity()
		if capacityGB > 0 {
			config.Storage.CapacityGB = capacityGB
		} else {
			config.Storage.CapacityGB = 100
		}
	}

	if config.Node.Region == "" {
		config.Node.Region = detectRegion()
	}
	if config.Node.Name == "" {
		config.Node.Name = generateNodeName()
	}

	if config.IPFS.APIURL == "" {
		config.IPFS.APIURL = "http://localhost:5001"
	}
	if config.IPFS.DataDir == "" {
		homeDir, _ := os.UserHomeDir()
		config.IPFS.DataDir = filepath.Join(homeDir, ".wabisaby", "ipfs")
	}

	return &config
}

// detectStorageCapacity detects available disk space and returns capacity in GB.
func detectStorageCapacity() int64 {
	var stat syscall.Statfs_t
	wd, err := os.Getwd()
	if err != nil {
		return 0
	}
	if err := syscall.Statfs(wd, &stat); err != nil {
		return 0
	}
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	usableBytes := availableBytes * 80 / 100
	return int64(usableBytes / (1024 * 1024 * 1024))
}

func detectRegion() string {
	tz := os.Getenv("TZ")
	if tz != "" {
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
	return "unknown"
}

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
	return strings.ToLower(strings.ReplaceAll(
		strings.Join([]string{"wabisaby-node", hostname, username}, "-"),
		" ", "-",
	))
}
