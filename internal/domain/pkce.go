package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
)

type Verifier string

func GenerateVerifier() (Verifier, error) {
	verifier, err := security.GenerateCodeVerifier()
	if err != nil {
		return "", fmt.Errorf("error generating verifier: %v", err)
	}

	return Verifier(verifier), nil
}

func (v Verifier) String() string {
	return string(v)
}

type Challange string

func GenerateChallange(verifier Verifier) (Challange, error) {
	challange, err := security.GenerateCodeChallenge(string(verifier))
	if err != nil {
		return "", fmt.Errorf("error generating challange: %v", err)
	}

	return Challange(challange), nil
}

func (c Challange) String() string {
	return string(c)
}

type PKCE struct {
	State     uuid.UUID
	Verifier  Verifier
	Challange Challange
	ExpiryTTL time.Duration
	CreatedAt time.Time
}

func NewPKCE(verifier Verifier, challange Challange, expiryTTL time.Duration) (*PKCE, error) {
	if verifier == "" {
		return nil, fmt.Errorf("verifier is required")
	}

	if challange == "" {
		return nil, fmt.Errorf("challange is required")
	}

	if expiryTTL <= 0 {
		return nil, fmt.Errorf("expire ttl is required")
	}

	return &PKCE{
		State:     uuid.New(),
		Verifier:  verifier,
		Challange: challange,
		ExpiryTTL: expiryTTL,
		CreatedAt: time.Now(),
	}, nil
}

func (p *PKCE) IsExpired(now time.Time) bool {
	return now.After(p.CreatedAt.Add(p.ExpiryTTL))
}

type PKCERepository interface {
	Save(ctx context.Context, e *PKCE) error
	FindByState(ctx context.Context, state uuid.UUID) (*PKCE, error)
	Delete(ctx context.Context, state uuid.UUID) error
}
