package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

// Key prefixes keep each flow's codes apart, so a password reset code never
// replaces a pending email verification code, or the other way round.
const (
	OTPPrefixVerification  = "otp"
	OTPPrefixPasswordReset = "otp_reset"
)

type otpStore struct {
	client *redis.RedisClient
	prefix string
}

func NewOTPStore(client *redis.RedisClient, prefix string) otp.Store {
	return &otpStore{client: client, prefix: prefix}
}

func (s *otpStore) Save(ctx context.Context, email string, record *otp.OTPRecord, ttl time.Duration) error {
	key := fmt.Sprintf("%s:%s", s.prefix, email)
	return s.client.SetStruct(ctx, key, record, ttl)
}

func (s *otpStore) Get(ctx context.Context, email string) (*otp.OTPRecord, error) {
	key := fmt.Sprintf("%s:%s", s.prefix, email)
	var record otp.OTPRecord
	if err := s.client.GetStruct(ctx, key, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *otpStore) Delete(ctx context.Context, email string) error {
	key := fmt.Sprintf("%s:%s", s.prefix, email)
	return s.client.Delete(ctx, key)
}
