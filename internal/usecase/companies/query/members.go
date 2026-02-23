package query

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ListMembersUsecase struct {
	contextDuration time.Duration
	membersRepo     domain.CompanyMemberRepository
}

func NewListMembersUsecase(contextDuration time.Duration, membersRepo domain.CompanyMemberRepository) *ListMembersUsecase {
	return &ListMembersUsecase{
		contextDuration: contextDuration,
		membersRepo:     membersRepo,
	}
}

type MemberResponse struct {
	CompanyID string    `json:"company_id"`
	MemberID  string    `json:"member_id"`
	Alias     string    `json:"alias"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *ListMembersUsecase) ListMembers(ctx context.Context, companyID string) (_ []*MemberResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "ListMembers",
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	var input struct {
		companyID uuid.UUID
	}
	{
		input.companyID, err = uuid.Parse(companyID)
		if err != nil {
			return nil, inerr.NewErrValidation("company_id", "invalid company ID")
		}
	}

	members, err := u.membersRepo.FindByCompanyID(ctx, input.companyID)
	if err != nil {
		return nil, err
	}

	result := make([]*MemberResponse, len(members))
	for i, m := range members {
		result[i] = &MemberResponse{
			CompanyID: m.CompanyID.String(),
			MemberID:  m.MemberID.String(),
			Alias:     m.Alias,
			Role:      m.Role.String(),
			CreatedAt: m.CreatedAt,
		}
	}

	return result, nil
}
