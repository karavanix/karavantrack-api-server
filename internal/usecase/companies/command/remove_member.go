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
	contextDuration    time.Duration
	companyMembersRepo domain.CompanyMemberRepository
}

func NewRemoveMemberUsecase(contextDuration time.Duration, companyMembersRepo domain.CompanyMemberRepository) *RemoveMemberUsecase {
	return &RemoveMemberUsecase{
		contextDuration: contextDuration,
		companyMembersRepo:     companyMembersRepo,
	}
}

func (u *RemoveMemberUsecase) RemoveMember(ctx context.Context, requesterID, companyID, memberID string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "RemoveMember",
		attribute.String("company_id", companyID),
		attribute.String("requester_id", requesterID),
		attribute.String("member_id", memberID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		actorID   uuid.UUID
		memberID  uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("actor_id", "invalid actor ID")
		}
		input.memberID, err = uuid.Parse(memberID)
		if err != nil {
			return inerr.NewErrValidation("member_id", "invalid member ID")
		}
	}

	// Ownership check: only owner can remove members
	actorMember, err := u.companyMembersRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.actorID)
	if err != nil {
		return inerr.ErrorPermissionDenied
	}

	if !actorMember.IsOwner() {
		return inerr.ErrorPermissionDenied
	}

	// Check that we're not removing the owner
	targetMember, err := u.companyMembersRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.memberID)
	if err != nil {
		return err
	}

	if targetMember.IsOwner() {
		return inerr.NewErrValidation("member", "cannot remove company owner")
	}

	if err := u.companyMembersRepo.DeleteByCompanyIDAndMemberID(ctx, input.companyID, input.memberID); err != nil {
		logger.ErrorContext(ctx, "failed to remove company member", err)
		return err
	}

	return nil
}
