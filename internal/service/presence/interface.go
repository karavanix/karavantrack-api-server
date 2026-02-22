package presence

import "context"

type Service interface {
	Online(ctx context.Context, userID string) error
	Offline(ctx context.Context, userID string) error
	IsOnline(ctx context.Context, userID string) (bool, error)
}
