package cache

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type BloomFilter interface {
	Add(ctx context.Context, key string, item string) error
	Exists(ctx context.Context, key string, item string) (bool, error)
	Reserve(ctx context.Context, key string, errorRate float64, capacity int64) error
}

type RedisBloomFilter struct {
	client *redis.Client
}

func NewRedisBloomFilter(client *redis.Client) *RedisBloomFilter {
	return &RedisBloomFilter{client: client}
}

func (r *RedisBloomFilter) Add(ctx context.Context, key string, item string) error {
	// Use command "BF.ADD" with the key and item
	return r.client.Do(ctx, "BF.ADD", key, item).Err()
}

func (r *RedisBloomFilter) Exists(ctx context.Context, key string, item string) (bool, error) {
	// Use command "BF.EXISTS" with the key and item
	// Call Do to get *redis.Cmd
	cmd := r.client.Do(ctx, "BF.EXISTS", key, item)

	// Call .Bool() on the cmd object
	return cmd.Bool()
}

func (r *RedisBloomFilter) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) error {
	// The BF.RESERVE command creates a new Bloom Filter with the specified error rate and capacity.
	return r.client.Do(ctx, "BF.RESERVE", key, errorRate, capacity).Err()
}
