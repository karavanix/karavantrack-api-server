package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
)

func GenerateSignature(keyID, privateKeyBase64, timestamp string) (string, error) {
	msg := keyID + timestamp

	hash := sha512.Sum512([]byte(msg))

	keyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", err
	}

	priv, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return "", err
	}

	rsaKey := priv.(*rsa.PrivateKey)

	sig, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA512, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sig), nil
}
