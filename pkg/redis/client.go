package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration
}

func New(opt ...Options) (*RedisClient, error) {
	o := options{
		Address:      "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DefaultTTL:   10 * time.Minute,
		DialTimeout:  5 * time.Second,
		KeyPrefix:    "",
	}

	for _, opt := range opt {
		opt(&o)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         o.Address,
		Password:     o.Password,
		DB:           o.DB,
		PoolSize:     o.PoolSize,
		MinIdleConns: o.MinIdleConns,
		DialTimeout:  o.DialTimeout,
	})

	return &RedisClient{
		client:     rdb,
		keyPrefix:  o.KeyPrefix,
		defaultTTL: o.DefaultTTL,
	}, nil
}

func (c *RedisClient) Close() error {
	return c.client.Close()
}

func (c *RedisClient) prefixer(key string) string {
	return c.keyPrefix + key
}

func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.prefixer(key)).Result()
}

func (c *RedisClient) Set(ctx context.Context, key, value string, ttl ...time.Duration) error {
	if len(ttl) == 0 {
		ttl = []time.Duration{c.defaultTTL}
	}

	return c.client.Set(ctx, c.prefixer(key), value, ttl[0]).Err()
}

func (c *RedisClient) Delete(ctx context.Context, key string) error {
	fmt.Println("Delete: ", c.prefixer(key))
	return c.client.Del(ctx, c.prefixer(key)).Err()
}

func (c *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, c.prefixer(key)).Bytes()
}

func (c *RedisClient) SetBytes(ctx context.Context, key string, value []byte, ttl ...time.Duration) error {
	if len(ttl) == 0 {
		ttl = []time.Duration{c.defaultTTL}
	}

	return c.client.Set(ctx, c.prefixer(key), value, ttl[0]).Err()
}

func (c *RedisClient) GetStruct(ctx context.Context, key string, dest any) error {
	fmt.Println("Get: ", c.prefixer(key))
	data, err := c.Get(ctx, c.prefixer(key))
	if err == redis.Nil {
		return inerr.NewErrNotFound(key)
	} else if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (c *RedisClient) SetStruct(ctx context.Context, key string, src any, ttl ...time.Duration) error {
	if len(ttl) == 0 {
		ttl = []time.Duration{c.defaultTTL}
	}

	data, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("redis: failed to marshal struct: %w", err)
	}
	fmt.Println("Set: ", c.prefixer(key), string(data))
	return c.Set(ctx, c.prefixer(key), string(data), ttl[0])
}

func (c *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, c.prefixer(key)).Result()
	return count > 0, err
}

func (c *RedisClient) SAdd(ctx context.Context, key string, members ...string) error {
	return c.client.SAdd(ctx, c.prefixer(key), members).Err()
}

func (c *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, c.prefixer(key)).Result()
}

func (c *RedisClient) SRem(ctx context.Context, key string, members ...string) error {
	return c.client.SRem(ctx, c.prefixer(key), members).Err()
}

func (c *RedisClient) SIsMember(ctx context.Context, key string, member string) (bool, error) {
	return c.client.SIsMember(ctx, c.prefixer(key), member).Result()
}
