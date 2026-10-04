package ports

import "context"

type AppleUserInfo struct {
	Sub   string
	Email string
}

type AppleProvider interface {
	// Verify checks a Sign in with Apple id_token issued for our app.
	Verify(ctx context.Context, idToken string) (*AppleUserInfo, error)
	// ExchangeCode trades the sign-in's authorization code (valid 5 minutes,
	// once) for Apple's refresh token, kept to revoke the app's access later.
	ExchangeCode(ctx context.Context, code string) (string, error)
	// Revoke ends the app's access to the Apple ID behind refreshToken, as
	// Apple requires when the account is deleted. Apple then treats the next
	// sign-in as a first one and sends the user's name again.
	Revoke(ctx context.Context, refreshToken string) error
}
