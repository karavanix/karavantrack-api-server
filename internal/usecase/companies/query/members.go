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
	usersRepo       domain.UserRepository
}

func NewListMembersUsecase(contextDuration time.Duration, membersRepo domain.CompanyMemberRepository, usersRepo domain.UserRepository) *ListMembersUsecase {
	return &ListMembersUsecase{
		contextDuration: contextDuration,
		membersRepo:     membersRepo,
		usersRepo:       usersRepo,
	}
}

type ListMembersRequest struct {
	Query  string `form:"q"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

type MemberResponse struct {
	CompanyID string    `json:"company_id"`
	MemberID  string    `json:"member_id"`
	Alias     string    `json:"alias"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

func (u *ListMembersUsecase) ListMembers(ctx context.Context, companyID string, req *ListMembersRequest) (_ []*MemberResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "ListMembers",
		attribute.String("company_id", companyID),
	)
	defer func() { end(err) }()

	compID, err := uuid.Parse(companyID)
	if err != nil {
		return nil, inerr.NewErrValidation("company_id", "invalid company ID")
	}

	members, err := u.membersRepo.FindByCompanyIDWithFilter(ctx, compID, &domain.CompanyMemberFilter{
		Query:  req.Query,
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return nil, err
	}

	memberUserIDs := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		memberUserIDs = append(memberUserIDs, m.MemberID)
	}

	users, err := u.usersRepo.FindByIDs(ctx, memberUserIDs)
	if err != nil {
		return nil, err
	}

	var filtered []*MemberResponse
	for _, m := range members {
		user, ok := users[m.MemberID]
		if !ok {
			continue
		}

		filtered = append(filtered, &MemberResponse{
			CompanyID: m.CompanyID.String(),
			MemberID:  m.MemberID.String(),
			Alias:     m.Alias,
			Role:      m.Role.String(),
			CreatedAt: m.CreatedAt,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})
	}
	return filtered, nil
}
