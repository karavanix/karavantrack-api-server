package command

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type SubmitLeadUsecase struct {
	contextDuration time.Duration
	leadsRepo       domain.LeadRepository
}

func NewSubmitLeadUsecase(contextDuration time.Duration, leadsRepo domain.LeadRepository) *SubmitLeadUsecase {
	return &SubmitLeadUsecase{contextDuration: contextDuration, leadsRepo: leadsRepo}
}

type SubmitLeadRequest struct {
	Name      string
	Company   string
	Phone     string
	Fleet     string
	UserAgent string
}

// SubmitLead records a sales contact request from the public marketing
// landing page. It is reachable without authentication, so the only
// validation is the same "all fields present" check the landing form does.
func (u *SubmitLeadUsecase) SubmitLead(ctx context.Context, req SubmitLeadRequest) (err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("leads"), "SubmitLead")
	defer func() { end(err) }()

	lead, err := domain.NewLead(req.Name, req.Company, req.Phone, req.Fleet, req.UserAgent)
	if err != nil {
		return inerr.NewErrValidation("lead", err.Error())
	}

	return u.leadsRepo.Save(ctx, lead)
}
