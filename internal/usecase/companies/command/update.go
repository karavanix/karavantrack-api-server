package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type UpdateUsecase struct {
	contextDuration time.Duration
	companiesRepo   domain.CompanyRepository
	membersRepo     domain.CompanyMemberRepository
}

func NewUpdateUsecase(contextDuration time.Duration, companiesRepo domain.CompanyRepository, membersRepo domain.CompanyMemberRepository) *UpdateUsecase {
	return &UpdateUsecase{
		contextDuration: contextDuration,
		companiesRepo:   companiesRepo,
		membersRepo:     membersRepo,
	}
}

type UpdateRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

func (u *UpdateUsecase) Update(ctx context.Context, requesterID, companyID string, req *UpdateRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "Update",
		attribute.String("company_id", companyID),
		attribute.String("requester_id", requesterID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID   uuid.UUID
		requesterID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.requesterID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("user_id", "invalid user ID")
		}
	}

	company, err := u.companiesRepo.FindByID(ctx, input.companyID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find company", err)
		return err
	}

	company.Update(req.Name)
	if err := u.companiesRepo.Save(ctx, company); err != nil {
		logger.ErrorContext(ctx, "failed to update company", err)
		return err
	}

	return nil
}
