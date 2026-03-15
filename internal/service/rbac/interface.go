package rbac

import (
	"context"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

type Service interface {
	HasPermission(ctx context.Context, companyID string, userID string, permission domain.CompanyPermission) (bool, error)
}
