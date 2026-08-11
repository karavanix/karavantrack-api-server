package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// LoadInviteTTL is how long a generated invite link stays valid (pending)
// before it is treated as expired.
const LoadInviteTTL = 7 * 24 * time.Hour

type LoadInviteStatus string

const (
	LoadInviteStatusPending  LoadInviteStatus = "pending"
	LoadInviteStatusAccepted LoadInviteStatus = "accepted"
	LoadInviteStatusExpired  LoadInviteStatus = "expired"
	LoadInviteStatusRevoked  LoadInviteStatus = "revoked"
)

func (s LoadInviteStatus) String() string {
	return string(s)
}

// LoadInvite is a shareable, single-load invite link a shipper generates so a
// driver can open it, register/log in as a carrier, and accept the load —
// without the shipper needing the driver's phone/email upfront.
type LoadInvite struct {
	ID         uuid.UUID
	LoadID     uuid.UUID
	Token      string
	Status     LoadInviteStatus
	CreatedBy  uuid.UUID
	AcceptedBy uuid.UUID
	AcceptedAt *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewLoadInvite(loadID uuid.UUID, createdBy uuid.UUID, token string) (*LoadInvite, error) {
	if loadID == uuid.Nil {
		return nil, errors.New("load ID is required")
	}
	if token == "" {
		return nil, errors.New("token is required")
	}

	now := time.Now()
	return &LoadInvite{
		ID:        uuid.New(),
		LoadID:    loadID,
		Token:     token,
		Status:    LoadInviteStatusPending,
		CreatedBy: createdBy,
		ExpiresAt: now.Add(LoadInviteTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsExpired reports whether a still-pending invite is past its expiry.
func (i *LoadInvite) IsExpired() bool {
	return i.Status == LoadInviteStatusPending && time.Now().After(i.ExpiresAt)
}

// EffectiveStatus materializes "expired" dynamically for a pending invite
// that is past its expires_at, so no cron job is needed to sweep stale rows.
func (i *LoadInvite) EffectiveStatus() LoadInviteStatus {
	if i.IsExpired() {
		return LoadInviteStatusExpired
	}
	return i.Status
}

// IsActive reports whether the invite can still be accepted right now.
func (i *LoadInvite) IsActive() bool {
	return i.EffectiveStatus() == LoadInviteStatusPending
}

// Accept marks the invite as accepted by the given carrier user.
func (i *LoadInvite) Accept(carrierID uuid.UUID) error {
	if !i.IsActive() {
		return errors.New("invite is not pending")
	}
	now := time.Now()
	i.Status = LoadInviteStatusAccepted
	i.AcceptedBy = carrierID
	i.AcceptedAt = &now
	i.UpdatedAt = now
	return nil
}

type LoadInviteRepository interface {
	Save(ctx context.Context, invite *LoadInvite) error
	FindByToken(ctx context.Context, token string) (*LoadInvite, error)
	// FindActiveByLoadID returns the current pending, non-expired invite for
	// a load, if one exists (used to keep invite-link creation idempotent).
	FindActiveByLoadID(ctx context.Context, loadID uuid.UUID) (*LoadInvite, error)
}
