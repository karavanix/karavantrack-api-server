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
	contextDuration    time.Duration
	companyMembersRepo domain.CompanyMemberRepository
	usersRepo          domain.UserRepository
}

func NewAddMemberUsecase(
	contextDuration time.Duration,
	companyMembersRepo domain.CompanyMemberRepository,
	usersRepo domain.UserRepository,
) *AddMemberUsecase {
	return &AddMemberUsecase{
		contextDuration:    contextDuration,
		companyMembersRepo: companyMembersRepo,
		usersRepo:          usersRepo,
	}
}

type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Alias  string `json:"alias" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=admin member"`
}

func (u *AddMemberUsecase) AddMember(ctx context.Context, requesterID, companyID string, req *AddMemberRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "AddMember",
		attribute.String("company_id", companyID),
		attribute.String("requester_id", requesterID),
		attribute.String("user_id", req.UserID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
		actorID   uuid.UUID
		userID    uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return inerr.NewErrValidation("company_id", "invalid company ID")
		}
		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return inerr.NewErrValidation("requester_id", "invalid requester ID")
		}
		input.userID, err = uuid.Parse(req.UserID)
		if err != nil {
			return inerr.NewErrValidation("user_id", "invalid user ID")
		}
	}

	// Ownership check: only owner or admin can add members
	actorMember, err := u.companyMembersRepo.FindByCompanyIDAndMemberID(ctx, input.companyID, input.actorID)
	if err != nil {
		return inerr.ErrorPermissionDenied
	}

	if !actorMember.IsOwner() && !actorMember.IsAdmin() {
		return inerr.ErrorPermissionDenied
	}

	// Verify target user exists
	_, err = u.usersRepo.FindByID(ctx, input.userID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find user", err)
		return err
	}

	newMember, err := domain.NewCompanyMember(input.companyID, input.userID, req.Alias, domain.MemberRole(req.Role))
	if err != nil {
		logger.ErrorContext(ctx, "failed to create company member", err)
		return inerr.NewErrValidation("member", err.Error())
	}

	if err := u.companyMembersRepo.Save(ctx, newMember); err != nil {
		logger.ErrorContext(ctx, "failed to save company member", err)
		return err
	}

	return nil
}
