package apple

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	appleJWKSURL = "https://appleid.apple.com/auth/keys"
	appleIssuer  = "https://appleid.apple.com"
)

type UserInfo struct {
	Sub   string
	Email string
}

type Client struct {
	bundleID  string
	jwksCache *jwk.Cache
}

func NewClient(ctx context.Context, bundleID string) (*Client, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(appleJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("apple: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, appleJWKSURL); err != nil {
		return nil, fmt.Errorf("apple: initial JWKS fetch failed: %w", err)
	}

	return &Client{bundleID: bundleID, jwksCache: cache}, nil
}

func (c *Client) GetUserInfo(ctx context.Context, idToken string) (*UserInfo, error) {
	keySet, err := c.jwksCache.Get(ctx, appleJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("apple: failed to get JWKS: %w", err)
	}

	token, err := jwt.Parse(
		[]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(c.bundleID),
	)
	if err != nil {
		return nil, fmt.Errorf("apple: invalid id_token: %w", err)
	}

	sub := token.Subject()
	if sub == "" {
		return nil, fmt.Errorf("apple: missing sub claim")
	}

	emailStr := ""
	if emailVal, ok := token.Get("email"); ok {
		emailStr, _ = emailVal.(string)
	}

	return &UserInfo{Sub: sub, Email: emailStr}, nil
}
