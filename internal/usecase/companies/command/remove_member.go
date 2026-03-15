package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RemoveMemberUsecase struct {
	contextDuration    time.Duration
	companyMembersRepo domain.CompanyMemberRepository
	rbacService        rbac.Service
}

func NewRemoveMemberUsecase(contextDuration time.Duration, companyMembersRepo domain.CompanyMemberRepository, rbacService rbac.Service) *RemoveMemberUsecase {
	return &RemoveMemberUsecase{
		contextDuration:    contextDuration,
		companyMembersRepo: companyMembersRepo,
		rbacService:        rbacService,
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

	targetMember, err := u.companyMembersRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.memberID)
	if err != nil {
		return err
	}

	allow, err := u.rbacService.HasPermission(ctx,
		input.companyID.String(),
		input.actorID.String(),
		utils.If(
			targetMember.IsAdmin(),
			domain.CompanyPermissionMemberDeleteAdmin,
			domain.CompanyPermissionMemberDeleteMember,
		),
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check permission", err)
		return err
	}

	if !allow {
		return inerr.ErrorPermissionDenied
	}

	if err := u.companyMembersRepo.DeleteByCompanyIDAndMemberID(ctx, input.companyID, input.memberID); err != nil {
		logger.ErrorContext(ctx, "failed to remove company member", err)
		return err
	}

	return nil
}
