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

const trackingTokenBytes = 32

type CreateTrackingLinkUsecase struct {
	contextDuration   time.Duration
	loadsRepo         domain.LoadRepository
	trackingLinksRepo domain.LoadTrackingLinkRepository
	rbacService       rbac.Service
	publicAppBaseURL  string
}

func NewCreateTrackingLinkUsecase(
	contextDuration time.Duration,
	loadsRepo domain.LoadRepository,
	trackingLinksRepo domain.LoadTrackingLinkRepository,
	rbacService rbac.Service,
	publicAppBaseURL string,
) *CreateTrackingLinkUsecase {
	return &CreateTrackingLinkUsecase{
		contextDuration:   contextDuration,
		loadsRepo:         loadsRepo,
		trackingLinksRepo: trackingLinksRepo,
		rbacService:       rbacService,
		publicAppBaseURL:  publicAppBaseURL,
	}
}

type CreateTrackingLinkResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
}

// CreateTrackingLink generates (or, if one is already active, returns) a
// public, no-login, read-only tracking link for a load that a broker can
// hand to their client. Unlike invite links this has no expiry — it stays
// valid for the life of the load unless explicitly revoked (revoke is not
// implemented yet, see the "revoke" endpoint follow-up).
func (u *CreateTrackingLinkUsecase) CreateTrackingLink(ctx context.Context, requesterID string, loadID string) (_ *CreateTrackingLinkResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("tracking"), "CreateTrackingLink",
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

	existing, err := u.trackingLinksRepo.FindActiveByLoadID(ctx, input.loadID)
	if err != nil && !errors.Is(err, inerr.ErrNotFound{}) {
		return nil, err
	}
	if existing != nil {
		return u.toResponse(existing), nil
	}

	token, err := security.GenerateToken(trackingTokenBytes)
	if err != nil {
		return nil, err
	}

	link, err := domain.NewLoadTrackingLink(load.ID, input.actorID, token)
	if err != nil {
		return nil, inerr.NewErrValidation("load_id", err.Error())
	}

	if err := u.trackingLinksRepo.Save(ctx, link); err != nil {
		return nil, err
	}

	return u.toResponse(link), nil
}

func (u *CreateTrackingLinkUsecase) toResponse(link *domain.LoadTrackingLink) *CreateTrackingLinkResponse {
	return &CreateTrackingLinkResponse{
		Token: link.Token,
		URL:   strings.TrimRight(u.publicAppBaseURL, "/") + "/track/" + link.Token,
	}
}
