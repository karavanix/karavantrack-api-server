package rbac

import (
	"context"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

type Service interface {
	HasPermission(ctx context.Context, companyID string, userID string, permission ...domain.CompanyPermission) (bool, error)

	// CanAccessLoad reports whether requesterID may read the given load: either as the
	// carrier assigned to it, or as a company member holding the given permission.
	CanAccessLoad(ctx context.Context, requesterID string, load *domain.Load, permission domain.CompanyPermission) (bool, error)
}
