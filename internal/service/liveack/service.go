package liveack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

const (
	keyPrefix = "live_ack:load"
	keyTTL    = 10 * time.Minute

	StatusStarted = "started"
	StatusFailed  = "failed"
)

// Ack records whether the driver's phone actually managed to start streaming
// live GPS after the server asked it to (start_live_location), or why it
// couldn't — as opposed to the server's own one-way NATS publish, which has
// no notion of whether the phone ever received or acted on it.
type Ack struct {
	Status    string    `json:"status"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Service interface {
	SetStarted(ctx context.Context, loadID string) error
	SetFailed(ctx context.Context, loadID string, reason string) error
	Clear(ctx context.Context, loadID string) error
	// Get returns nil, nil when the phone hasn't acked this load yet.
	Get(ctx context.Context, loadID string) (*Ack, error)
}

type service struct {
	redis *redis.RedisClient
}

func NewService(redisClient *redis.RedisClient) Service {
	return &service{redis: redisClient}
}

func (s *service) SetStarted(ctx context.Context, loadID string) error {
	return s.set(ctx, loadID, &Ack{Status: StatusStarted, UpdatedAt: time.Now()})
}

func (s *service) SetFailed(ctx context.Context, loadID string, reason string) error {
	return s.set(ctx, loadID, &Ack{Status: StatusFailed, Reason: reason, UpdatedAt: time.Now()})
}

func (s *service) set(ctx context.Context, loadID string, ack *Ack) error {
	return s.redis.SetStruct(ctx, s.key(loadID), ack, keyTTL)
}

func (s *service) Clear(ctx context.Context, loadID string) error {
	return s.redis.Delete(ctx, s.key(loadID))
}

func (s *service) Get(ctx context.Context, loadID string) (*Ack, error) {
	var ack Ack
	err := s.redis.GetStruct(ctx, s.key(loadID), &ack)
	if err != nil {
		if errors.Is(err, inerr.ErrNotFound{}) {
			return nil, nil
		}
		return nil, err
	}
	return &ack, nil
}

func (s *service) key(loadID string) string {
	return fmt.Sprintf("%s:%s", keyPrefix, loadID)
}
