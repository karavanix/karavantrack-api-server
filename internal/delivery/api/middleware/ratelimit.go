package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/delivery/outerr"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

// RateLimit throttles requests per client IP using a fixed window counter in
// Redis, keyed by name (route) + IP. On a Redis error it fails open, since a
// broken limiter should not take the whole endpoint down.
func RateLimit(redisClient *redis.RedisClient, name string, limit int, window time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			key := "ratelimit:" + name + ":" + ip

			count, err := redisClient.GetRedisClient().Incr(r.Context(), key).Result()
			if err == nil {
				if count == 1 {
					redisClient.GetRedisClient().Expire(r.Context(), key, window)
				}
				if count > int64(limit) {
					outerr.TooManyRequests(w, r, "too many requests, try again later")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	// chimiddleware.RealIP (mounted before this middleware) already rewrites
	// r.RemoteAddr to the real client address when trusted proxy headers are set.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
