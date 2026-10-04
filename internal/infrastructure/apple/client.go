package apple

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/service/ports"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"resty.dev/v3"
)

const (
	appleBaseURL = "https://appleid.apple.com"
	appleJWKSURL = "https://appleid.apple.com/auth/keys"
	appleIssuer  = "https://appleid.apple.com"

	// Apple allows up to 6 months; one is made per request.
	clientSecretTTL = 5 * time.Minute
)

// ErrNotConfigured: APPLE_TEAM_ID, APPLE_KEY_ID or APPLE_PRIVATE_KEY is
// missing, so there's no client secret for Apple's token endpoints.
var ErrNotConfigured = errors.New("apple: team id, key id or private key not configured")

type appleClient struct {
	bundleID   string
	teamID     string
	keyID      string
	privateKey *ecdsa.PrivateKey
	jwksCache  *jwk.Cache
	httpClient *resty.Client
}

func New(ctx context.Context, cfg *config.Config) (ports.AppleProvider, error) {
	cache := jwk.NewCache(ctx)

	if err := cache.Register(appleJWKSURL, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("apple: failed to register JWKS cache: %w", err)
	}

	if _, err := cache.Refresh(ctx, appleJWKSURL); err != nil {
		return nil, fmt.Errorf("apple: initial JWKS fetch failed: %w", err)
	}

	c, err := newClient(cfg, appleBaseURL)
	if err != nil {
		return nil, err
	}
	c.jwksCache = cache
	return c, nil
}

func newClient(cfg *config.Config, baseURL string) (*appleClient, error) {
	c := &appleClient{
		bundleID:   cfg.Apple.BundleID,
		teamID:     cfg.Apple.TeamID,
		keyID:      cfg.Apple.KeyID,
		httpClient: resty.New().SetBaseURL(baseURL),
	}
	if cfg.Apple.PrivateKey != "" {
		key, err := parsePrivateKey(cfg.Apple.PrivateKey)
		if err != nil {
			return nil, err
		}
		c.privateKey = key
	}
	return c, nil
}

// parsePrivateKey reads the .p8 key from Apple Developer, base64-encoded
// like the other keys in the environment.
func parsePrivateKey(encoded string) (*ecdsa.PrivateKey, error) {
	decoded, err := security.DecodeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("apple: APPLE_PRIVATE_KEY: %w", err)
	}
	block, _ := pem.Decode([]byte(decoded))
	if block == nil {
		return nil, errors.New("apple: APPLE_PRIVATE_KEY is not a PEM key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("apple: APPLE_PRIVATE_KEY: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("apple: APPLE_PRIVATE_KEY is not an EC key")
	}
	return ecKey, nil
}

func (c *appleClient) Verify(ctx context.Context, idToken string) (*ports.AppleUserInfo, error) {
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

	return &ports.AppleUserInfo{Sub: sub, Email: emailStr}, nil
}

type tokenResponse struct {
	RefreshToken string `json:"refresh_token"`
}

type errorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (c *appleClient) ExchangeCode(ctx context.Context, code string) (string, error) {
	secret, err := c.clientSecret()
	if err != nil {
		return "", err
	}

	var result tokenResponse
	var errResult errorResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"client_id":     c.bundleID,
			"client_secret": secret,
			"code":          code,
			"grant_type":    "authorization_code",
		}).
		SetResult(&result).
		SetError(&errResult).
		Post("/auth/token")
	if err != nil {
		return "", fmt.Errorf("apple: token request failed: %w", err)
	}
	if resp.IsError() {
		return "", fmt.Errorf("apple: code exchange failed (status %d): %s", resp.StatusCode(), errResult.Error)
	}
	if result.RefreshToken == "" {
		return "", errors.New("apple: no refresh_token in token response")
	}
	return result.RefreshToken, nil
}

func (c *appleClient) Revoke(ctx context.Context, refreshToken string) error {
	secret, err := c.clientSecret()
	if err != nil {
		return err
	}

	var errResult errorResponse
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"client_id":       c.bundleID,
			"client_secret":   secret,
			"token":           refreshToken,
			"token_type_hint": "refresh_token",
		}).
		SetError(&errResult).
		Post("/auth/revoke")
	if err != nil {
		return fmt.Errorf("apple: revoke request failed: %w", err)
	}
	if resp.IsError() {
		return fmt.Errorf("apple: revoke failed (status %d): %s", resp.StatusCode(), errResult.Error)
	}
	return nil
}

// clientSecret is the JWT Apple's token endpoints take as client_secret,
// signed with the Sign in with Apple key.
func (c *appleClient) clientSecret() (string, error) {
	if c.teamID == "" || c.keyID == "" || c.privateKey == nil {
		return "", ErrNotConfigured
	}
	now := time.Now()
	token, err := jwt.NewBuilder().
		Issuer(c.teamID).
		IssuedAt(now).
		Expiration(now.Add(clientSecretTTL)).
		Audience([]string{appleIssuer}).
		Subject(c.bundleID).
		Build()
	if err != nil {
		return "", fmt.Errorf("apple: client secret: %w", err)
	}
	headers := jws.NewHeaders()
	if err := headers.Set(jws.KeyIDKey, c.keyID); err != nil {
		return "", fmt.Errorf("apple: client secret: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, c.privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		return "", fmt.Errorf("apple: client secret: %w", err)
	}
	return string(signed), nil
}
