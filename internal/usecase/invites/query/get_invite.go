package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type GetInviteUsecase struct {
	contextDuration time.Duration
	invitesRepo     domain.LoadInviteRepository
	loadsRepo       domain.LoadRepository
	companiesRepo   domain.CompanyRepository
}

func NewGetInviteUsecase(
	contextDuration time.Duration,
	invitesRepo domain.LoadInviteRepository,
	loadsRepo domain.LoadRepository,
	companiesRepo domain.CompanyRepository,
) *GetInviteUsecase {
	return &GetInviteUsecase{
		contextDuration: contextDuration,
		invitesRepo:     invitesRepo,
		loadsRepo:       loadsRepo,
		companiesRepo:   companiesRepo,
	}
}

// InviteLoadPreview is the intentionally-limited, public view of a load
// shown on the invite landing page: no internal IDs, no price, no company ID.
type InviteLoadPreview struct {
	ReferenceID    string     `json:"reference_id"`
	Title          string     `json:"title"`
	PickupAddress  string     `json:"pickup_address"`
	PickupAt       *time.Time `json:"pickup_at"`
	DropoffAddress string     `json:"dropoff_address"`
	DropoffAt      *time.Time `json:"dropoff_at"`
	CompanyName    string     `json:"company_name"`
}

type GetInviteResponse struct {
	Status string             `json:"status"`
	Load   *InviteLoadPreview `json:"load"`
}

// GetInvite is a PUBLIC, unauthenticated lookup used to render the invite
// landing page. It returns 200 even for expired/accepted/revoked invites so
// the frontend can show an explanatory state instead of a bare 404 — "expired"
// is computed dynamically rather than relying on a cron sweep.
func (u *GetInviteUsecase) GetInvite(ctx context.Context, token string) (_ *GetInviteResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("invites"), "GetInvite")
	defer func() { end(err) }()

	if token == "" {
		return nil, inerr.NewErrNotFound("invite")
	}

	invite, err := u.invitesRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	load, err := u.loadsRepo.FindByID(ctx, invite.LoadID)
	if err != nil {
		return nil, err
	}

	companyName := ""
	if load.CompanyID != uuid.Nil {
		company, cErr := u.companiesRepo.FindByID(ctx, load.CompanyID)
		if cErr != nil {
			logger.ErrorContext(ctx, "failed to load company for invite preview", cErr)
		} else {
			companyName = company.Name
		}
	}

	return &GetInviteResponse{
		Status: invite.EffectiveStatus().String(),
		Load: &InviteLoadPreview{
			ReferenceID:    load.ReferenceID,
			Title:          load.Title,
			PickupAddress:  load.PickupAddress,
			PickupAt:       load.PickupAt,
			DropoffAddress: load.DropoffAddress,
			DropoffAt:      load.DropoffAt,
			CompanyName:    companyName,
		},
	}, nil
}
