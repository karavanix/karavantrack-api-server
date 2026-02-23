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

func (u *UpdateUsecase) Update(ctx context.Context, callerUserID, companyIDStr string, req *UpdateRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "Update",
		attribute.String("company_id", companyIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	userID, err := uuid.Parse(callerUserID)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Ownership check: only owner or admin can update
	member, err := u.membersRepo.FindByCompanyAndUser(ctx, companyID, userID)
	if err != nil {
		return inerr.ErrorPermissionDenied
	}
	if member.Role != domain.MemberRoleOwner && member.Role != domain.MemberRoleAdmin {
		return inerr.ErrorPermissionDenied
	}

	company, err := u.companiesRepo.FindByID(ctx, companyID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find company", err)
		return err
	}

	company.Update(req.Name)

	if err := u.companiesRepo.Update(ctx, company); err != nil {
		logger.ErrorContext(ctx, "failed to update company", err)
		return err
	}

	return nil
}
