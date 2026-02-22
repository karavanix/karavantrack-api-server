package security

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
)

// AccessClaims defines JWT claims for access tokens.
type AccessClaims struct {
	jwt.RegisteredClaims
}

// RefreshClaims defines JWT claims for refresh tokens, including JTI.
type RefreshClaims struct {
	jwt.RegisteredClaims
}

type TokenDetails struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RefreshJTI       string
}

type JWTProvider struct {
	AccessSecret  []byte
	RefreshSecret []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	SigningMethod jwt.SigningMethod
}

func NewJWTProvider(accessSecret, refreshSecret []byte, accessTTL, refreshTTL time.Duration, signingMethod jwt.SigningMethod) *JWTProvider {
	return &JWTProvider{
		AccessSecret:  accessSecret,
		RefreshSecret: refreshSecret,
		AccessTTL:     accessTTL,
		RefreshTTL:    refreshTTL,
		SigningMethod: signingMethod,
	}
}

func (m *JWTProvider) GenerateTokens(userID string) (*TokenDetails, error) {
	now := time.Now()
	jti := uuid.New().String()

	// Access token
	accessExp := now.Add(m.AccessTTL)
	accessClaims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	at := jwt.NewWithClaims(m.SigningMethod, accessClaims)
	accessToken, err := at.SignedString(m.AccessSecret)
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshExp := now.Add(m.RefreshTTL)
	refreshClaims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}
	rt := jwt.NewWithClaims(m.SigningMethod, refreshClaims)
	refreshToken, err := rt.SignedString(m.RefreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenDetails{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		RefreshJTI:       jti,
	}, nil
}

func (m *JWTProvider) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != m.SigningMethod.Alg() {
			return nil, inerr.ErrorWrongAlgo
		}
		return m.AccessSecret, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, inerr.ErrorExpiredToken
		}
		return nil, inerr.NewErrInvalidToken(err)
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, inerr.NewErrInvalidToken(err)
	}
	return claims, nil
}

func (m *JWTProvider) ValidateRefreshToken(tokenStr string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != m.SigningMethod.Alg() {
			return nil, inerr.ErrorWrongAlgo
		}
		return m.RefreshSecret, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, inerr.NewErrInvalidToken(err)
		}
		return nil, inerr.NewErrInvalidToken(err)
	}
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, inerr.NewErrInvalidToken(err)
	}
	return claims, nil
}
