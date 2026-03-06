package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type ListUsecase struct {
	contextDuration time.Duration
	loadsRepo       domain.LoadRepository
}

func NewListUsecase(contextDuration time.Duration, loadsRepo domain.LoadRepository) *ListUsecase {
	return &ListUsecase{contextDuration: contextDuration, loadsRepo: loadsRepo}
}

type ListRequest struct {
	CompanyID string   `form:"company_id"`
	CarrierID string   `form:"carrier_id"`
	Status    []string `form:"status"`
	Limit     int      `form:"limit"`
	Offset    int      `form:"offset"`
}

type ListResponse struct {
	Result []*LoadResponse `json:"result"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Total  int             `json:"count"`
}

type LoadResponse struct {
	ID             string     `json:"id"`
	CompanyID      string     `json:"company_id,omitempty"`
	MemberID       string     `json:"member_id,omitempty"`
	CarrierID      string     `json:"carrier_id,omitempty"`
	ReferenceID    string     `json:"reference_id,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	Status         string     `json:"status"`
	PickupAddress  string     `json:"pickup_address,omitempty"`
	PickupLat      float64    `json:"pickup_lat"`
	PickupLng      float64    `json:"pickup_lng"`
	DropoffAddress string     `json:"dropoff_address,omitempty"`
	DropoffLat     float64    `json:"dropoff_lat"`
	DropoffLng     float64    `json:"dropoff_lng"`
	PickupAt       *time.Time `json:"pickup_at,omitempty"`
	DropoffAt      *time.Time `json:"dropoff_at,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

func loadToResponse(l *domain.Load) *LoadResponse {
	r := &LoadResponse{
		ID:             l.ID.String(),
		Title:          l.Title,
		Description:    l.Description,
		ReferenceID:    l.ReferenceID,
		Status:         l.Status.String(),
		PickupAddress:  l.PickupAddress,
		PickupLat:      l.PickupLat,
		PickupLng:      l.PickupLng,
		DropoffAddress: l.DropoffAddress,
		DropoffLat:     l.DropoffLat,
		DropoffLng:     l.DropoffLng,
		PickupAt:       l.PickupAt,
		DropoffAt:      l.DropoffAt,
		CreatedAt:      l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      l.UpdatedAt.Format(time.RFC3339),
	}
	if l.CompanyID != uuid.Nil {
		r.CompanyID = l.CompanyID.String()
	}
	if l.MemberID != uuid.Nil {
		r.MemberID = l.MemberID.String()
	}
	if l.CarrierID != uuid.Nil {
		r.CarrierID = l.CarrierID.String()
	}
	return r
}

func (u *ListUsecase) List(ctx context.Context, req *ListRequest) (_ *ListResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("loads"), "List")
	defer func() { end(err) }()

	filter := domain.LoadFilter{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	if req.CompanyID != "" {
		id, err := uuid.Parse(req.CompanyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
		filter.CompanyID = &id
	}

	if req.CarrierID != "" {
		id, err := uuid.Parse(req.CarrierID)
		if err != nil {
			return nil, inerr.NewErrValidation("carrier_id", "invalid carrier ID")
		}
		filter.CarrierID = &id
	}

	if len(req.Status) > 0 {
		for _, s := range req.Status {
			filter.Status = append(filter.Status, domain.LoadStatus(s))
		}
	}

	loads, total, err := u.loadsRepo.FindAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := &ListResponse{
		Result: make([]*LoadResponse, len(loads)),
		Limit:  req.Limit,
		Offset: req.Offset,
		Total:  total,
	}

	for i, l := range loads {
		result.Result[i] = loadToResponse(l)
	}

	return result, nil
}
