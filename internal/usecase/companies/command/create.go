package command

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type CreateUsecase struct {
	contextDuration time.Duration
	txManager       postgres.TxManager
	usersRepo       domain.UserRepository
	companiesRepo   domain.CompanyRepository
	membersRepo     domain.CompanyMemberRepository
}

func NewCreateUsecase(
	contextDuration time.Duration,
	txManager postgres.TxManager,
	usersRepo domain.UserRepository,
	companiesRepo domain.CompanyRepository,
	membersRepo domain.CompanyMemberRepository,
) *CreateUsecase {
	return &CreateUsecase{
		contextDuration: contextDuration,
		txManager:       txManager,
		usersRepo:       usersRepo,
		companiesRepo:   companiesRepo,
		membersRepo:     membersRepo,
	}
}

type CreateRequest struct {
	Name string `json:"name" validate:"required,min=2,max=255"`
}

type CreateResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (u *CreateUsecase) Create(ctx context.Context, userID string, req *CreateRequest) (_ *CreateResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("companies"), "Create",
		attribute.String("user_id", userID),
	)
	defer func() { end(err) }()

	var input struct {
		ownerID uuid.UUID
	}
	{
		input.ownerID, err = uuid.Parse(userID)
		if err != nil {
			return nil, inerr.NewErrValidation("user_id", "invalid user ID")
		}
	}

	user, err := u.usersRepo.FindByID(ctx, input.ownerID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find user", err)
		return nil, err
	}

	company, err := domain.NewCompany(input.ownerID, req.Name)
	if err != nil {
		return nil, inerr.NewErrValidation("company", err.Error())
	}

	err = u.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := u.companiesRepo.Save(ctx, company); err != nil {
			logger.ErrorContext(ctx, "failed to save company", err)
			return err
		}

		member, err := domain.NewCompanyMember(company.ID, input.ownerID, user.FullName(), domain.MemberRoleOwner)
		if err != nil {
			return err
		}
		if err := u.membersRepo.Save(ctx, member); err != nil {
			logger.ErrorContext(ctx, "failed to save company member", err)
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CreateResponse{
		ID:   company.ID.String(),
		Name: company.Name,
	}, nil
}
