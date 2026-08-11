package command

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/internal/service/rbac"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/security"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const inviteTokenBytes = 32

type CreateInviteLinkUsecase struct {
	contextDuration  time.Duration
	loadsRepo        domain.LoadRepository
	invitesRepo      domain.LoadInviteRepository
	rbacService      rbac.Service
	publicAppBaseURL string
}

func NewCreateInviteLinkUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	invitesRepo domain.LoadInviteRepository,
	rbacService rbac.Service,
	publicAppBaseURL string,
) *CreateInviteLinkUsecase {
	return &CreateInviteLinkUsecase{
		contextDuration:  contextDuration,
		loadsRepo:        loadsRepo,
		invitesRepo:      invitesRepo,
		rbacService:      rbacService,
		publicAppBaseURL: publicAppBaseURL,
	}
}

type CreateInviteLinkResponse struct {
	Token     string    `json:"token"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateInviteLink generates (or, if one is already active, returns) a
// shareable invite link for a driver to open, register/log in as a carrier,
// and accept this specific load — without the shipper needing the driver's
// phone/email upfront.
func (u *CreateInviteLinkUsecase) CreateInviteLink(ctx context.Context, requesterID string, loadID string) (_ *CreateInviteLinkResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("invites"), "CreateInviteLink",
		attribute.String("requester_id", requesterID),
		attribute.String("load_id", loadID),
	)
	defer func() { end(err) }()

	var input struct {
		actorID uuid.UUID
		loadID  uuid.UUID
	}
	{
		input.loadID, err = uuid.Parse(loadID)
		if err != nil {
			return nil, inerr.NewErrValidation("load_id", "invalid load ID")
		}

		input.actorID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, inerr.NewErrValidation("requester_id", "invalid requester ID")
		}
	}

	load, err := u.loadsRepo.FindByID(ctx, input.loadID)
	if err != nil {
		return nil, err
	}

	allow, err := u.rbacService.HasPermission(ctx,
		load.CompanyID.String(),
		input.actorID.String(),
		domain.CompanyPermissionLoadUpdate,
	)
	if err != nil {
		return nil, err
	}
	if !allow {
		return nil, inerr.ErrorPermissionDenied
	}

	if load.CarrierID != uuid.Nil {
		return nil, inerr.NewErrValidation("load_id", "load is already assigned to a carrier")
	}

	// Idempotent: return the existing active invite instead of minting a
	// second one for the same load.
	existing, err := u.invitesRepo.FindActiveByLoadID(ctx, input.loadID)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		return nil, err
	}
	if existing != nil {
		return u.toResponse(existing), nil
	}

	token, err := security.GenerateToken(inviteTokenBytes)
	if err != nil {
		return nil, err
	}

	invite, err := domain.NewLoadInvite(load.ID, input.actorID, token)
	if err != nil {
		return nil, inerr.NewErrValidation("load_id", err.Error())
	}

	if err := u.invitesRepo.Save(ctx, invite); err != nil {
		return nil, err
	}

	return u.toResponse(invite), nil
}

func (u *CreateInviteLinkUsecase) toResponse(invite *domain.LoadInvite) *CreateInviteLinkResponse {
	return &CreateInviteLinkResponse{
		Token:     invite.Token,
		URL:       strings.TrimRight(u.publicAppBaseURL, "/") + "/invite/" + invite.Token,
		ExpiresAt: invite.ExpiresAt,
	}
}
