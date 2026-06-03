package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"resty.dev/v3"
)

const (
	telegramBaseURL = "https://oauth.telegram.org"
	telegramJWKSURL = "https://oauth.telegram.org/.well-known/jwks.json"
	telegramIssuer  = "https://oauth.telegram.org"
)

type telegramClient struct {
	clientID         string
	allowedAudiences map[string]bool
	jwksCache        *jwk.Cache
	httpClient       *resty.Client
}

func New(ctx context.Context, cfg *config.Config) (domain.TelegramProvider, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(telegramJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("telegram: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, telegramJWKSURL); err != nil {
		return nil, fmt.Errorf("telegram: initial JWKS fetch failed: %w", err)
	}

	ids := []string{cfg.Telegram.ClientID, cfg.Telegram.IOSClientID, cfg.Telegram.AndroidClientID}
	allowed := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id != "" {
			allowed[id] = true
		}
	}

	client := resty.New().
		SetBaseURL(telegramBaseURL).
		SetBasicAuth(cfg.Telegram.ClientID, cfg.Telegram.ClientSecret).
		SetResponseBodyUnlimitedReads(true)

	return &telegramClient{
		clientID:         cfg.Telegram.ClientID,
		allowedAudiences: allowed,
		jwksCache:        cache,
		httpClient:       client,
	}, nil
}

func (c *telegramClient) ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (string, error) {
	var result tokenResponse
	var errResult errorResponse

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"redirect_uri":  redirectURI,
			"code_verifier": codeVerifier,
		}).
		SetResult(&result).
		SetError(&errResult).
		Post("/token")
	if err != nil {
		return "", fmt.Errorf("telegram: token request failed: %w", err)
	}

	if resp.IsError() {
		return "", fmt.Errorf("telegram: token exchange failed (status %d): %s", resp.StatusCode(), errResult.ErrorDescription)
	}

	if result.IDToken == "" {
		return "", fmt.Errorf("telegram: no id_token in token response")
	}

	return result.IDToken, nil
}

func (c *telegramClient) Verify(ctx context.Context, idToken string) (*domain.TelegramUserInfo, error) {
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
	// Telegram may put the bot-level client ID or the platform-specific native
	// app ID in aud depending on which OAuth client initiated the flow.
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

	id, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: non-numeric sub claim: %s", sub)
	}

	firstName, lastName := splitName(claimStr(token, "name"))

	return &domain.TelegramUserInfo{
		ID:          id,
		FirstName:   firstName,
		LastName:    lastName,
		Username:    claimStr(token, "preferred_username"),
		PhotoURL:    claimStr(token, "picture"),
		PhoneNumber: claimStr(token, "phone_number"),
	}, nil
}

func splitName(name string) (string, string) {
	if name == "" {
		return "", ""
	}
	parts := strings.SplitN(strings.TrimSpace(name), " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func claimStr(token jwt.Token, key string) string {
	v, ok := token.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
