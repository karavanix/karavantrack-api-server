package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type AcceptInviteUsecase struct {
	contextDuration time.Duration
	invitesRepo     domain.LoadInviteRepository
	loadsRepo       domain.LoadRepository
	usersRepo       domain.UserRepository
	taskQueue       *asynq.Client
}

func NewAcceptInviteUsecase(
	contextDuration time.Duration,
	invitesRepo domain.LoadInviteRepository,
	loadsRepo domain.LoadRepository,
	usersRepo domain.UserRepository,
	taskQueue *asynq.Client,
) *AcceptInviteUsecase {
	return &AcceptInviteUsecase{
		contextDuration: contextDuration,
		invitesRepo:     invitesRepo,
		loadsRepo:       loadsRepo,
		usersRepo:       usersRepo,
		taskQueue:       taskQueue,
	}
}

type AcceptInviteResponse struct {
	LoadID string `json:"load_id"`
}

// AcceptInvite lets the authenticated carrier who opened an invite link
// accept the load it points to: the carrier is assigned to the load (same
// domain transition as the shipper-initiated /assign flow) and the invite is
// marked accepted.
func (u *AcceptInviteUsecase) AcceptInvite(ctx context.Context, token string, carrierID string) (_ *AcceptInviteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("invites"), "AcceptInvite",
		attribute.String("carrier_id", carrierID),
	)
	defer func() { end(err) }()

	if token == "" {
		return nil, inerr.NewErrValidation("token", "token is required")
	}

	var input struct {
		carrierID uuid.UUID
	}
	input.carrierID, err = uuid.Parse(carrierID)
	if err != nil {
		return nil, inerr.NewErrValidation("carrier_id", "invalid carrier ID")
	}

	invite, err := u.invitesRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	switch invite.EffectiveStatus() {
	case domain.LoadInviteStatusAccepted:
		return nil, inerr.ErrInviteAlreadyAccepted
	case domain.LoadInviteStatusRevoked:
		return nil, inerr.ErrInviteRevoked
	case domain.LoadInviteStatusExpired:
		return nil, inerr.ErrInviteExpired
	}

	load, err := u.loadsRepo.FindByID(ctx, invite.LoadID)
	if err != nil {
		return nil, err
	}

	// Race: someone else got assigned between invite creation and acceptance.
	if load.CarrierID != uuid.Nil {
		return nil, inerr.ErrLoadAlreadyAssigned
	}

	carrier, err := u.usersRepo.FindByID(ctx, input.carrierID)
	if err != nil {
		return nil, err
	}
	if !carrier.IsCarrier() {
		return nil, inerr.NewErrValidation("carrier_id", "user is not a carrier")
	}

	if err := load.Assign("Accepted via invite link", carrier.ID); err != nil {
		return nil, inerr.NewErrValidation("status", err.Error())
	}

	if err := u.loadsRepo.Save(ctx, load); err != nil {
		logger.ErrorContext(ctx, "failed to update load", err)
		return nil, err
	}

	if err := invite.Accept(carrier.ID); err != nil {
		return nil, inerr.NewErrValidation("status", err.Error())
	}

	if err := u.invitesRepo.Save(ctx, invite); err != nil {
		logger.ErrorContext(ctx, "failed to update invite", err)
		return nil, err
	}

	// Mirror the normal /assign flow's side effect of onboarding the
	// carrier into the company, but skip the "you were assigned a new
	// load" push notification — the driver is the one who just acted.
	addCarrierTask, err := tasks.NewSendAddCarrierToCompanyTask(
		&tasks.AddCarrierToCompanyPayload{
			ActorID:   invite.CreatedBy.String(),
			CompanyID: load.CompanyID.String(),
			CarrierID: carrier.ID.String(),
			Alias:     carrier.FullName(),
		},
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create add carrier task", err)
	} else if _, err := u.taskQueue.EnqueueContext(ctx, addCarrierTask); err != nil {
		logger.ErrorContext(ctx, "failed to enqueue add carrier task", err)
	}

	return &AcceptInviteResponse{LoadID: load.ID.String()}, nil
}
