package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

type presenceRedisStore struct {
	client         *redis.RedisClient
	ttl            time.Duration
	presnecePrefix string
}

func NewPresenceRedisStore(cfg *config.Config, client *redis.RedisClient) domain.PresenceRepository {
	return &presenceRedisStore{
		client:         client,
		presnecePrefix: "presence",
		ttl:            1 * time.Minute,
	}
}

func (s *presenceRedisStore) Save(ctx context.Context, e *domain.Presence) error {
	key := fmt.Sprintf("%s:%s", s.presnecePrefix, e.UserID.String())
	value := fmt.Sprint(e.LastSeenAt.Unix())

	err := s.client.Set(ctx, key, value, s.ttl)
	if err != nil {
		return err
	}

	return nil
}

func (s *presenceRedisStore) Delete(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("%s:%s", s.presnecePrefix, userID.String())

	return s.client.Delete(ctx, key)
}

func (s *presenceRedisStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("%s:%s", s.presnecePrefix, userID.String())

	return s.client.Exists(ctx, key)
}
