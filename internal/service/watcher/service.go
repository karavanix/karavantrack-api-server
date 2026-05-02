package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

const (
	keyPrefix = "watcher:load"
	keyTTL    = 2 * time.Hour
)

type Service interface {
	Join(ctx context.Context, loadID string) (int64, error)
	Leave(ctx context.Context, loadID string) (int64, error)
}

type service struct {
	redis *redis.RedisClient
}

func NewService(redisClient *redis.RedisClient) Service {
	return &service{redis: redisClient}
}

func (s *service) Join(ctx context.Context, loadID string) (int64, error) {
	key := fmt.Sprintf("%s:%s", keyPrefix, loadID)
	count, err := s.redis.GetRedisClient().Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	s.redis.GetRedisClient().Expire(ctx, key, keyTTL)
	return count, nil
}

func (s *service) Leave(ctx context.Context, loadID string) (int64, error) {
	key := fmt.Sprintf("%s:%s", keyPrefix, loadID)
	count, err := s.redis.GetRedisClient().Decr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count <= 0 {
		s.redis.GetRedisClient().Del(ctx, key)
		return 0, nil
	}
	return count, nil
}
