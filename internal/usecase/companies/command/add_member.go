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

type AddMemberUsecase struct {
	contextDuration time.Duration
	membersRepo     domain.CompanyMemberRepository
	usersRepo       domain.UserRepository
}

func NewAddMemberUsecase(
	contextDuration time.Duration,
	membersRepo domain.CompanyMemberRepository,
	usersRepo domain.UserRepository,
) *AddMemberUsecase {
	return &AddMemberUsecase{
		contextDuration: contextDuration,
		membersRepo:     membersRepo,
		usersRepo:       usersRepo,
	}
}

type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Alias  string `json:"alias" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=admin member"`
}

func (u *AddMemberUsecase) AddMember(ctx context.Context, companyIDStr string, req *AddMemberRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "AddMember",
		attribute.String("company_id", companyIDStr),
		attribute.String("user_id", req.UserID),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Verify user exists
	_, err = u.usersRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	member, err := domain.NewCompanyMember(companyID, userID, req.Alias, domain.MemberRole(req.Role))
	if err != nil {
		return inerr.NewErrValidation("member", err.Error())
	}

	if err := u.membersRepo.Save(ctx, member); err != nil {
		logger.ErrorContext(ctx, "failed to save company member", err)
		return err
	}

	return nil
}
