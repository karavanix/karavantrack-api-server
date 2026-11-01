package otp_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/otp"
)

// memStore implements otp.Store in memory.
type memStore struct {
	records map[string]otp.OTPRecord
}

func newMemStore() *memStore {
	return &memStore{records: map[string]otp.OTPRecord{}}
}

func (s *memStore) Save(ctx context.Context, email string, record *otp.OTPRecord, ttl time.Duration) error {
	s.records[email] = *record
	return nil
}

func (s *memStore) Get(ctx context.Context, email string) (*otp.OTPRecord, error) {
	record, ok := s.records[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return &record, nil
}

func (s *memStore) Delete(ctx context.Context, email string) error {
	delete(s.records, email)
	return nil
}

const email = "driver@example.com"

func newService(store otp.Store) *otp.Service {
	return otp.NewService(store, otp.Config{
		Secret:      []byte("test-secret"),
		TTL:         15 * time.Minute,
		MaxAttempts: 3,
		Length:      6,
	})
}

// wrongCode returns a 6-digit code that differs from code.
func wrongCode(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestVerifyAcceptsGeneratedCode(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore())

	code, err := svc.Generate(ctx, email)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if err := svc.Verify(ctx, email, code); err != nil {
		t.Fatalf("Verify(generated code) = %v, want nil", err)
	}
}

func TestVerifyMismatchCountsAttempts(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	svc := newService(store)

	code, err := svc.Generate(ctx, email)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	for want := 1; want <= 3; want++ {
		if err := svc.Verify(ctx, email, wrongCode(code)); !errors.Is(err, inerr.ErrOTPMismatch) {
			t.Fatalf("attempt %d: Verify = %v, want ErrOTPMismatch", want, err)
		}
		if got := store.records[email].Attempts; got != want {
			t.Fatalf("attempt %d: Attempts = %d, want %d", want, got, want)
		}
	}

	// Out of attempts: even the right code no longer passes.
	if err := svc.Verify(ctx, email, code); !errors.Is(err, inerr.ErrOTPTooManyAttempts) {
		t.Fatalf("Verify after 3 misses = %v, want ErrOTPTooManyAttempts", err)
	}
}

// 567657 used to be accepted for any email that had an OTP record.
func TestVerifyRejectsFormerMasterCode(t *testing.T) {
	ctx := context.Background()
	svc := newService(newMemStore())

	code, err := svc.Generate(ctx, email)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if code == "567657" {
		t.Skip("generated code happens to be 567657")
	}

	if err := svc.Verify(ctx, email, "567657"); !errors.Is(err, inerr.ErrOTPMismatch) {
		t.Fatalf("Verify(567657) = %v, want ErrOTPMismatch", err)
	}
}
