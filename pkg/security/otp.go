package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"math/big"
)

// GenerateNumericOTP returns a zero-padded string of length n.
func GenerateNumericOTP(n int) (string, error) {
	max := big.NewInt(10)
	otp := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		otp[i] = byte('0' + num.Int64())
	}
	return string(otp), nil
}

// HashOTP produces an HMAC-SHA256 hex digest of the code.
func HashOTP(code string, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("otp secret is empty")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// CompareHashAndOTP does a constant‐time HMAC compare.
func CompareHashAndOTP(hashHex, code string, secret []byte) (bool, error) {
	expected, err := HashOTP(code, secret)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(hashHex)) == 1, nil
}
