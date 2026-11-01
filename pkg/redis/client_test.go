package redis_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/redis"
)

// Needs a real Redis: REDIS_TEST_ADDR=localhost:6390 go test ./pkg/redis/
func newTestClient(t *testing.T) *redis.RedisClient {
	t.Helper()
	addr := os.Getenv("REDIS_TEST_ADDR")
	if addr == "" {
		t.Skip("REDIS_TEST_ADDR not set")
	}
	client, err := redis.New(redis.WithAddress(addr), redis.WithKeyPrefix("test"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

type record struct {
	Code string `json:"code"`
}

// A struct saved with SetStruct must be the key Delete removes: one-time
// codes (OTP, PKCE) are spent through Delete.
func TestStructRoundTripAndDelete(t *testing.T) {
	ctx := context.Background()
	client := newTestClient(t)
	key := "otp:someone@example.com"

	if err := client.SetStruct(ctx, key, &record{Code: "123"}, time.Minute); err != nil {
		t.Fatalf("SetStruct: %v", err)
	}

	var got record
	if err := client.GetStruct(ctx, key, &got); err != nil || got.Code != "123" {
		t.Fatalf("GetStruct = %+v, %v; want code 123", got, err)
	}

	// Stored under the prefix once, like every other method uses it.
	if raw, err := client.GetRedisClient().Get(ctx, "test:"+key).Result(); err != nil || raw == "" {
		t.Fatalf("raw key test:%s = %q, %v; want the struct", key, raw, err)
	}

	if err := client.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := client.GetStruct(ctx, key, &got); !errors.Is(err, inerr.ErrNotFound{}) {
		t.Fatalf("GetStruct after Delete = %v, want ErrNotFound", err)
	}
}
