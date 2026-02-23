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

func (u *RemoveMemberUsecase) RemoveMember(ctx context.Context, callerUserID, companyIDStr, targetUserIDStr string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "RemoveMember",
		attribute.String("company_id", companyIDStr),
		attribute.String("user_id", targetUserIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return inerr.NewErrValidation("company_id", "invalid company ID")
	}

	callerID, err := uuid.Parse(callerUserID)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid caller user ID")
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return inerr.NewErrValidation("user_id", "invalid user ID")
	}

	// Ownership check: only owner can remove members
	callerMember, err := u.membersRepo.FindByCompanyAndUser(ctx, companyID, callerID)
	if err != nil {
		return inerr.ErrorPermissionDenied
	}
	if callerMember.Role != domain.MemberRoleOwner {
		return inerr.ErrorPermissionDenied
	}

	// Check that we're not removing the owner
	target, err := u.membersRepo.FindByCompanyAndUser(ctx, companyID, targetUserID)
	if err != nil {
		return err
	}
	if target.Role == domain.MemberRoleOwner {
		return inerr.NewErrValidation("member", "cannot remove company owner")
	}

	if err := u.membersRepo.Delete(ctx, companyID, targetUserID); err != nil {
		logger.ErrorContext(ctx, "failed to remove company member", err)
		return err
	}

	return nil
}
