package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	telegramJWKSURL  = "https://oauth.telegram.org/.well-known/jwks.json"
	telegramIssuer   = "https://oauth.telegram.org"
	telegramTokenURL = "https://oauth.telegram.org/token"
)

type UserInfo struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	PhotoURL  string
}

type Client struct {
	clientID         string
	clientSecret     string
	allowedAudiences map[string]bool
	jwksCache        *jwk.Cache
}

// NewClient creates a Telegram OIDC client that accepts tokens whose aud claim
// matches any of the provided IDs. Pass the bot-level client ID plus any
// platform-specific native app IDs so the check covers all possible aud values.
// clientSecret is optional — only needed for server-side authorization-code exchange.
func NewClient(ctx context.Context, clientID, clientSecret string, extraClientIDs ...string) (*Client, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(telegramJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("telegram: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, telegramJWKSURL); err != nil {
		return nil, fmt.Errorf("telegram: initial JWKS fetch failed: %w", err)
	}

	allIDs := append([]string{clientID}, extraClientIDs...)
	allowed := make(map[string]bool, len(allIDs))
	for _, id := range allIDs {
		if id != "" {
			allowed[id] = true
		}
	}

	return &Client{
		clientID:         clientID,
		clientSecret:     clientSecret,
		allowedAudiences: allowed,
		jwksCache:        cache,
	}, nil
}

type tokenResponse struct {
	IDToken string `json:"id_token"`
	Error   string `json:"error"`
}

// ExchangeCode exchanges a Telegram OAuth authorization code for an OIDC id_token.
// The client_secret is used server-side only and never exposed to the browser.
func (c *Client) ExchangeCode(ctx context.Context, code, redirectURI string) (string, error) {
	body := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {redirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, telegramTokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("telegram: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("telegram: token request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("telegram: read token response: %w", err)
	}

	var tok tokenResponse
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("telegram: parse token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK || tok.Error != "" {
		return "", fmt.Errorf("telegram: token exchange failed (status %d): %s", resp.StatusCode, tok.Error)
	}

	if tok.IDToken == "" {
		return "", fmt.Errorf("telegram: no id_token in token response")
	}

	return tok.IDToken, nil
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

	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("telegram: non-numeric sub claim: %s", sub)
	}

	// Telegram issues a single "name" claim (full name) rather than separate
	// given_name / family_name. Split on the first space.
	firstName, lastName := splitName(claimStr(token, "name"))

	return &UserInfo{
		ID:        id,
		FirstName: firstName,
		LastName:  lastName,
		Username:  claimStr(token, "preferred_username"),
		PhotoURL:  claimStr(token, "picture"),
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
