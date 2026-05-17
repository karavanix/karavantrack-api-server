package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
)

// Generate returns a zero-padded random numeric string of length n.
func Generate(n int) (string, error) {
	max := big.NewInt(10)
	code := make([]byte, n)
	for i := range code {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + num.Int64())
	}
	return string(code), nil
}

// Hash produces an HMAC-SHA256 hex digest of the code.
func Hash(code string, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("otp: secret is empty")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// Verify does a constant-time HMAC comparison.
func Verify(hashHex, code string, secret []byte) (bool, error) {
	expected, err := Hash(code, secret)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(hashHex)) == 1, nil
}
