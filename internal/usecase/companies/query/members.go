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
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Alias     string `json:"alias"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func (u *ListMembersUsecase) List(ctx context.Context, companyIDStr string) (_ []*MemberResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "ListMembers",
		attribute.String("company_id", companyIDStr),
	)
	defer func() { end(err) }()

	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return nil, inerr.NewErrValidation("company_id", "invalid company ID")
	}

	members, err := u.membersRepo.FindByCompanyID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	result := make([]*MemberResponse, len(members))
	for i, m := range members {
		result[i] = &MemberResponse{
			CompanyID: m.CompanyID.String(),
			UserID:    m.UserID.String(),
			Alias:     m.Alias,
			Role:      m.Role.String(),
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}
