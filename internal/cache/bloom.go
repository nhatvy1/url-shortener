package cache

import "context"

type BloomFilter interface {
	Add(ctx context.Context, key string, item string) error
	Exists(ctx context.Context, key string, item string) (bool, error)
	Reserve(ctx context.Context, key string, errorRate float64, capacity int64) error
}
