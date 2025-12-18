package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService provides Redis-based caching for ETags
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new Redis cache service
// Returns nil if redisURL is empty (graceful degradation)
func NewCacheService(redisURL string) (*CacheService, error) {
	if redisURL == "" {
		return nil, nil // No Redis configured, cache disabled
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &CacheService{client: client}, nil
}

// GetETag retrieves cached ETag for a user+filter combination
func (c *CacheService) GetETag(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", nil
	}

	result, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Key not found
	}
	return result, err
}

// SetETag stores ETag with TTL (default 1 hour)
func (c *CacheService) SetETag(ctx context.Context, key, etag string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}

	if ttl == 0 {
		ttl = time.Hour // Default 1 hour TTL
	}

	return c.client.Set(ctx, key, etag, ttl).Err()
}

// InvalidateUserETags removes all ETags for a specific user
// Call this after email sync to invalidate cached ETags
func (c *CacheService) InvalidateUserETags(ctx context.Context, userID string) error {
	if c == nil || c.client == nil {
		return nil
	}

	pattern := fmt.Sprintf("etag:emails:%s:*", userID)
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()

	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

// Close closes the Redis connection
func (c *CacheService) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// GenerateETagKey creates a cache key for ETag storage
func GenerateETagKey(userID string, filterHash string) string {
	return fmt.Sprintf("etag:emails:%s:%s", userID, filterHash)
}
