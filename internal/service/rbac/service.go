package rbac

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type service struct {
	contextTimeout    time.Duration
	companyMemberRepo domain.CompanyMemberRepository
}

func NewService(contextTimeout time.Duration, companyMemberRepo domain.CompanyMemberRepository) Service {
	return &service{
		contextTimeout:    contextTimeout,
		companyMemberRepo: companyMemberRepo,
	}
}

func (s *service) HasPermission(ctx context.Context, companyID, userID string, permission ...domain.CompanyPermission) (_ bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, s.contextTimeout)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("service.rbac"), "HasPermission")
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		userID    uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return false, inerr.NewErrValidation("company_id", err.Error())
		}
		input.userID, err = uuid.Parse(userID)
		if err != nil {
			return false, inerr.NewErrValidation("user_id", err.Error())
		}
	}

	member, err := s.companyMemberRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find company member", err)
		return false, err
	}

	for _, p := range permission {
		if !member.HasPermission(p) {
			return false, nil
		}
	}

	return true, nil
}
