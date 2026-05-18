package telegram

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	telegramJWKSURL = "https://oauth.telegram.org/.well-known/jwks.json"
	telegramIssuer  = "https://oauth.telegram.org"
)

type UserInfo struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	PhotoURL  string
}

type Client struct {
	clientID  string
	jwksCache *jwk.Cache
}

func NewClient(ctx context.Context, clientID string) (*Client, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(telegramJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("telegram: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, telegramJWKSURL); err != nil {
		return nil, fmt.Errorf("telegram: initial JWKS fetch failed: %w", err)
	}

	return &Client{clientID: clientID, jwksCache: cache}, nil
}

// Verify validates a Telegram OIDC id_token and returns the authenticated user's info.
func (c *Client) Verify(ctx context.Context, idToken string) (*UserInfo, error) {
	keySet, err := c.jwksCache.Get(ctx, telegramJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("telegram: failed to get JWKS: %w", err)
	}

	token, err := jwt.Parse(
		[]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(telegramIssuer),
		jwt.WithAudience(c.clientID),
	)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid id_token: %w", err)
	}

	sub := token.Subject()
	if sub == "" {
		return nil, fmt.Errorf("telegram: missing sub claim")
	}

	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: non-numeric sub claim: %s", sub)
	}

	return &UserInfo{
		ID:        id,
		FirstName: claimStr(token, "given_name"),
		LastName:  claimStr(token, "family_name"),
		Username:  claimStr(token, "preferred_username"),
		PhotoURL:  claimStr(token, "picture"),
	}, nil
}

func claimStr(token jwt.Token, key string) string {
	v, ok := token.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
