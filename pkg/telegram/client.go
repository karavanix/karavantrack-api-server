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
	allowedAudiences map[string]bool
	jwksCache        *jwk.Cache
}

// NewClient creates a Telegram OIDC client that accepts tokens issued for any
// of the provided clientIDs (one per platform: iOS, Android, web, etc.).
func NewClient(ctx context.Context, clientIDs ...string) (*Client, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(telegramJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("telegram: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, telegramJWKSURL); err != nil {
		return nil, fmt.Errorf("telegram: initial JWKS fetch failed: %w", err)
	}

	allowed := make(map[string]bool, len(clientIDs))
	for _, id := range clientIDs {
		if id != "" {
			allowed[id] = true
		}
	}

	return &Client{allowedAudiences: allowed, jwksCache: cache}, nil
}

// Verify validates a Telegram OIDC id_token and returns the authenticated user's info.
func (c *Client) Verify(ctx context.Context, idToken string) (*UserInfo, error) {
	keySet, err := c.jwksCache.Get(ctx, telegramJWKSURL)
	if err != nil {
		return nil, fmt.Errorf("telegram: failed to get JWKS: %w", err)
	}

	// Parse and validate signature + standard claims, but NOT audience —
	// we check audience manually below to support multiple client IDs.
	token, err := jwt.Parse(
		[]byte(idToken),
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
		jwt.WithIssuer(telegramIssuer),
	)
	if err != nil {
		return nil, fmt.Errorf("telegram: invalid id_token: %w", err)
	}

	// Verify the token was issued for one of our registered client IDs.
	audienceOK := false
	for _, aud := range token.Audience() {
		if c.allowedAudiences[aud] {
			audienceOK = true
			break
		}
	}
	if !audienceOK {
		return nil, fmt.Errorf("telegram: token audience %v is not in the allowed client ID list", token.Audience())
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
