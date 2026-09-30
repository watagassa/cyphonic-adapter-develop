package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/Pluslab/cyphonic-adapter/adapterd/config"
	"github.com/Pluslab/cyphonic-adapter/adapterd/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// VirtualIPInfo represents the virtual IP information for a child device
type VirtualIPInfo struct {
	DeviceName  string `json:"device_name"`
	DeviceID    string `json:"device_id"`
	MacAddress  string `json:"mac_address"`
	VirtualIPv4 string `json:"virtual_ipv4,omitempty"`
	VirtualIPv6 string `json:"virtual_ipv6,omitempty"`
	UpdatedAt   string `json:"updated_at"`
	LastSeen    string `json:"last_seen"`
}

// Client manages Redis operations for virtual IP information
type Client struct {
	rdb *redis.Client
	ctx context.Context
}

const (
	keyPrefix  = "cyphonic:adapter:vip:" // Redis key prefix for virtual IP information
	defaultTTL = 24 * time.Hour          // Default TTL for virtual IP entries (24 hours)
)

// NewClient creates a new Redis client
func NewClient(cfg *config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info(fmt.Sprintf("Connected to Redis at %s:%d", cfg.Redis.Host, cfg.Redis.Port))

	return &Client{
		rdb: rdb,
		ctx: ctx,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// SaveVirtualIP saves virtual IP information to Redis
func (c *Client) SaveVirtualIP(deviceName, deviceID, macAddress string, ipv4, ipv6 netip.Addr) error {
	now := getCurrentTimestamp()
	info := &VirtualIPInfo{
		DeviceName: deviceName,
		DeviceID:   deviceID,
		MacAddress: macAddress,
		UpdatedAt:  now,
		LastSeen:   now,
	}

	if ipv4.IsValid() {
		info.VirtualIPv4 = ipv4.String()
	}

	if ipv6.IsValid() {
		info.VirtualIPv6 = ipv6.String()
	}

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual IP info: %w", err)
	}

	key := keyPrefix + macAddress
	if err := c.rdb.Set(c.ctx, key, data, defaultTTL).Err(); err != nil {
		return fmt.Errorf("failed to save to Redis: %w", err)
	}

	return nil
}

// GetAllVirtualIPs retrieves all virtual IP information
func (c *Client) GetAllVirtualIPs() ([]*VirtualIPInfo, error) {
	keys, err := c.rdb.Keys(c.ctx, keyPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys from Redis: %w", err)
	}

	if len(keys) == 0 {
		return []*VirtualIPInfo{}, nil
	}

	infos := make([]*VirtualIPInfo, 0, len(keys))

	for _, key := range keys {
		data, err := c.rdb.Get(c.ctx, key).Result()
		if err != nil {
			logger.Warn(fmt.Sprintf("failed to get data for key %s: %v", key, err))
			continue
		}

		var info VirtualIPInfo
		if err := json.Unmarshal([]byte(data), &info); err != nil {
			logger.Warn(fmt.Sprintf("failed to unmarshal data for key %s: %v", key, err))
			continue
		}

		infos = append(infos, &info)
	}

	return infos, nil
}

// UpdateLastSeen updates the last seen timestamp for a device
func (c *Client) UpdateLastSeen(macAddress string) error {
	key := keyPrefix + macAddress

	data, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get from Redis: %w", err)
	}

	var info VirtualIPInfo
	if err = json.Unmarshal([]byte(data), &info); err != nil {
		return fmt.Errorf("failed to unmarshal virtual IP info: %w", err)
	}

	info.LastSeen = getCurrentTimestamp()

	newData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal virtual IP info: %w", err)
	}

	if err = c.rdb.Set(c.ctx, key, newData, defaultTTL).Err(); err != nil {
		return fmt.Errorf("failed to update Redis: %w", err)
	}

	return nil
}

// DeleteVirtualIP removes virtual IP information for a specific MAC address
func (c *Client) DeleteVirtualIP(macAddress string) error {
	key := keyPrefix + macAddress

	if err := c.rdb.Del(c.ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from Redis: %w", err)
	}

	return nil
}

// getCurrentTimestamp returns the current timestamp in RFC3339 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
