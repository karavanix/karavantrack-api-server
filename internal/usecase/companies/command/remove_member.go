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

type RemoveMemberUsecase struct {
	contextDuration time.Duration
	membersRepo     domain.CompanyMemberRepository
}

func NewRemoveMemberUsecase(contextDuration time.Duration, membersRepo domain.CompanyMemberRepository) *RemoveMemberUsecase {
	return &RemoveMemberUsecase{
		contextDuration: contextDuration,
		membersRepo:     membersRepo,
	}
}

func (u *RemoveMemberUsecase) RemoveMember(ctx context.Context, companyIDStr, userIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "RemoveMember",
		attribute.String("company_id", companyIDStr),
		attribute.String("user_id", userIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Check that we're not removing the owner
	member, err := u.membersRepo.FindByCompanyAndUser(ctx, companyID, userID)
	if err != nil {
		return err
	}
	if member.Role == domain.MemberRoleOwner {
		return inerr.NewErrValidation("member", "cannot remove company owner")
	}

	if err := u.membersRepo.Delete(ctx, companyID, userID); err != nil {
		logger.ErrorContext(ctx, "failed to remove company member", err)
		return err
	}

	return nil
}
