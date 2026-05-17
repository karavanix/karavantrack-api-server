package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

const otpPrefix = "otp"

type otpStore struct {
	client *redis.RedisClient
}

func NewOTPStore(client *redis.RedisClient) otp.Store {
	return &otpStore{client: client}
}

func (s *otpStore) Save(ctx context.Context, email string, record *otp.OTPRecord, ttl time.Duration) error {
	key := fmt.Sprintf("%s:%s", otpPrefix, email)
	return s.client.SetStruct(ctx, key, record, ttl)
}

func (s *otpStore) Get(ctx context.Context, email string) (*otp.OTPRecord, error) {
	key := fmt.Sprintf("%s:%s", otpPrefix, email)
	var record otp.OTPRecord
	if err := s.client.GetStruct(ctx, key, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *otpStore) Delete(ctx context.Context, email string) error {
	key := fmt.Sprintf("%s:%s", otpPrefix, email)
	return s.client.Delete(ctx, key)
}
