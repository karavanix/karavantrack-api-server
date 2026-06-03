package domain

import "context"

type TelegramUserInfo struct {
	ID          uint64
	FirstName   string
	LastName    string
	Username    string
	PhotoURL    string
	PhoneNumber string
}

type TelegramProvider interface {
	ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (string, error)
	Verify(ctx context.Context, idToken string) (*TelegramUserInfo, error)
}
