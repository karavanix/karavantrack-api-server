package security

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateToken returns a cryptographically random, URL-safe opaque token
// built from nBytes random bytes (crypto/rand) and base64.RawURLEncoding —
// the same style as GenerateCodeVerifier. Used for public, unguessable
// secrets such as invite and tracking-link tokens. Never derive a public
// token from a Load/User UUID — those are not secret.
func GenerateToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
