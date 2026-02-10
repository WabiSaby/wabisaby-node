// Copyright (c) 2026 WabiSaby
// SPDX-License-Identifier: MIT

package ipfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides an interface to the IPFS HTTP API.
// It contains only the methods required by the storage node (Version, ID, RepoStat, Pin).
type Client struct {
	apiURL     string
	httpClient *http.Client
}

// NewClient creates a new IPFS HTTP API client.
func NewClient(apiURL string) *Client {
	return &Client{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// RepoStatResult holds IPFS repository statistics returned from /repo/stat.
type RepoStatResult struct {
	RepoSize   uint64 `json:"RepoSize"`
	StorageMax uint64 `json:"StorageMax"`
}

// Pin pins a CID to the local IPFS node.
func (c *Client) Pin(ctx context.Context, cid string) error {
	url := fmt.Sprintf("%s/api/v0/pin/add?arg=%s", c.apiURL, cid)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		bodyStr := ""
		if err == nil {
			bodyStr = string(bodyBytes)
		}
		return fmt.Errorf("IPFS pin failed with status %d: %s", resp.StatusCode, bodyStr)
	}

	return nil
}

// ID returns the IPFS node's peer ID and addresses.
func (c *Client) ID(ctx context.Context) (string, []string, error) {
	url := fmt.Sprintf("%s/api/v0/id", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("IPFS id failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ID        string   `json:"ID"`
		Addresses []string `json:"Addresses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, result.Addresses, nil
}

// Version returns the IPFS node version.
func (c *Client) Version(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v0/version", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("IPFS version failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Version, nil
}

// RepoStat queries the local IPFS node for its storage (repo) statistics.
func (c *Client) RepoStat(ctx context.Context) (*RepoStatResult, error) {
	url := fmt.Sprintf("%s/api/v0/repo/stat", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IPFS repo stat failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result RepoStatResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
