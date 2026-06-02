package cache

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

type pkceRedisStore struct {
	client     *redis.RedisClient
	pkcePrefix string
}

func NewPKCEStore(cfg *config.Config, client *redis.RedisClient) domain.PKCERepository {
	return &pkceRedisStore{
		client:     client,
		pkcePrefix: cfg.PKCE.CachePrefix,
	}
}

func (s *pkceRedisStore) Save(ctx context.Context, e *domain.PKCE) error {
	key := fmt.Sprintf("%s:%s", s.pkcePrefix, e.State.String())
	ttl := e.ExpiryTTL

	err := s.client.SetStruct(ctx, key, e, ttl)
	if err != nil {
		return err
	}

	return nil
}

func (s *pkceRedisStore) FindByState(ctx context.Context, state uuid.UUID) (*domain.PKCE, error) {
	key := fmt.Sprintf("%s:%s", s.pkcePrefix, state.String())

	e := &domain.PKCE{}
	err := s.client.GetStruct(ctx, key, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func (s *pkceRedisStore) Delete(ctx context.Context, state uuid.UUID) error {
	key := fmt.Sprintf("%s:%s", s.pkcePrefix, state.String())
	return s.client.Delete(ctx, key)
}
