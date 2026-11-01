// Package revocation lets the auth flow invalidate a user's outstanding refresh
// tokens without a per-session store: it records a "not valid before" cutoff per
// user and RefreshToken rejects any token issued earlier than that cutoff.
package revocation

import (
	"context"
	"strconv"
	"time"

	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

type Service interface {
	// RevokeAllBefore invalidates every refresh token issued for userID before now.
	// Used on logout and account deletion.
	RevokeAllBefore(ctx context.Context, userID string, before time.Time) error

	// IsRevoked reports whether a refresh token issued at issuedAt for userID
	// has been revoked.
	IsRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error)
}

type service struct {
	redis *redis.RedisClient
	ttl   time.Duration
}

// NewService builds a revocation service. ttl should be at least as long as the
// refresh token TTL, so a cutoff always outlives every token it could apply to.
func NewService(redisClient *redis.RedisClient, ttl time.Duration) Service {
	return &service{redis: redisClient, ttl: ttl}
}

func key(userID string) string {
	return "auth:revoked_before:" + userID
}

func (s *service) RevokeAllBefore(ctx context.Context, userID string, before time.Time) error {
	// A token's iat is whole seconds. A cutoff with a fraction would revoke a
	// token issued in the same second right after it (sign-in just after a
	// password reset), so the cutoff is cut to the second as well.
	cutoff := before.Truncate(time.Second)
	return s.redis.Set(ctx, key(userID), strconv.FormatInt(cutoff.UnixNano(), 10), s.ttl)
}

func (s *service) IsRevoked(ctx context.Context, userID string, issuedAt time.Time) (bool, error) {
	value, err := s.redis.Get(ctx, key(userID))
	if err != nil {
		// No cutoff recorded for this user means nothing has been revoked.
		return false, nil
	}

	cutoffNano, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return false, nil
	}

	return issuedAt.UnixNano() < cutoffNano, nil
}
