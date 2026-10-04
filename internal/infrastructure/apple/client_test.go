package apple

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// newTestClient answers every request with status and body, and records
// the request's path and form.
func newTestClient(t *testing.T, status int, body string) (*appleClient, *ecdsa.PrivateKey, *string, *url.Values) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	p8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	var gotPath string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{}
	cfg.Apple.BundleID = "live.yool.app"
	cfg.Apple.TeamID = "TEAM123456"
	cfg.Apple.KeyID = "KEY1234567"
	cfg.Apple.PrivateKey = base64.StdEncoding.EncodeToString(p8)
	c, err := newClient(cfg, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c, key, &gotPath, &gotForm
}

// checkClientSecret: signed by our key, with the claims Apple checks.
func checkClientSecret(t *testing.T, secret string, key *ecdsa.PrivateKey) {
	t.Helper()
	token, err := jwt.Parse([]byte(secret), jwt.WithKey(jwa.ES256, &key.PublicKey), jwt.WithValidate(true))
	if err != nil {
		t.Fatalf("client secret: %v", err)
	}
	if token.Issuer() != "TEAM123456" || token.Subject() != "live.yool.app" {
		t.Errorf("client secret iss %q sub %q", token.Issuer(), token.Subject())
	}
	if aud := token.Audience(); len(aud) != 1 || aud[0] != "https://appleid.apple.com" {
		t.Errorf("client secret aud %v", aud)
	}
	msg, err := jws.Parse([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if kid := msg.Signatures()[0].ProtectedHeaders().KeyID(); kid != "KEY1234567" {
		t.Errorf("client secret kid %q", kid)
	}
}

func TestExchangeCode(t *testing.T) {
	c, key, path, form := newTestClient(t, http.StatusOK,
		`{"access_token":"a","token_type":"Bearer","expires_in":3600,"refresh_token":"r.apple","id_token":"i"}`)

	got, err := c.ExchangeCode(context.Background(), "code-1")
	if err != nil {
		t.Fatal(err)
	}
	if got != "r.apple" {
		t.Errorf("refresh token %q", got)
	}
	if *path != "/auth/token" {
		t.Errorf("path %q", *path)
	}
	f := *form
	if f.Get("grant_type") != "authorization_code" || f.Get("code") != "code-1" || f.Get("client_id") != "live.yool.app" {
		t.Errorf("form %v", f)
	}
	checkClientSecret(t, f.Get("client_secret"), key)
}

func TestExchangeCodeRejected(t *testing.T) {
	c, _, _, _ := newTestClient(t, http.StatusBadRequest, `{"error":"invalid_grant"}`)

	if _, err := c.ExchangeCode(context.Background(), "used"); err == nil {
		t.Fatal("want an error")
	}
}

func TestRevoke(t *testing.T) {
	c, key, path, form := newTestClient(t, http.StatusOK, ``)

	if err := c.Revoke(context.Background(), "r.apple"); err != nil {
		t.Fatal(err)
	}
	if *path != "/auth/revoke" {
		t.Errorf("path %q", *path)
	}
	f := *form
	if f.Get("token") != "r.apple" || f.Get("token_type_hint") != "refresh_token" || f.Get("client_id") != "live.yool.app" {
		t.Errorf("form %v", f)
	}
	checkClientSecret(t, f.Get("client_secret"), key)
}

func TestRevokeRejected(t *testing.T) {
	c, _, _, _ := newTestClient(t, http.StatusBadRequest, `{"error":"invalid_client"}`)

	if err := c.Revoke(context.Background(), "r.apple"); err == nil {
		t.Fatal("want an error")
	}
}

func TestNotConfigured(t *testing.T) {
	cfg := &config.Config{}
	cfg.Apple.BundleID = "live.yool.app"
	c, err := newClient(cfg, "http://unused")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ExchangeCode(context.Background(), "c"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("exchange: %v", err)
	}
	if err := c.Revoke(context.Background(), "r"); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("revoke: %v", err)
	}
}
