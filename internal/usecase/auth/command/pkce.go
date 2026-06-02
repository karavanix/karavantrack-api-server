package command

import (
	"context"
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"go.opentelemetry.io/otel"
)

type GeneratePKCEUsecase struct {
	contextDuration time.Duration
	pkceRepo        domain.PKCERepository
}

func NewGeneratePKCEUsecase(
	contextDuration time.Duration,
	pkceRepo domain.PKCERepository,
) *GeneratePKCEUsecase {
	return &GeneratePKCEUsecase{
		contextDuration: contextDuration,
		pkceRepo:        pkceRepo,
	}
}

type GeneratePKCEResponse struct {
	State         string `json:"state"`
	CodeChallenge string `json:"code_challenge"`
}

func (u *GeneratePKCEUsecase) GeneratePKCE(ctx context.Context) (_ *GeneratePKCEResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("auth"), "GeneratePKCE")
	defer func() { end(err) }()

	verifier, err := domain.GenerateVerifier()
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate verifier", err)
		return nil, err
	}

	challange, err := domain.GenerateChallange(verifier)
	if err != nil {
		logger.ErrorContext(ctx, "failed to generate challange", err)
		return nil, err
	}

	pkce, err := domain.NewPKCE(verifier, challange, 5*time.Minute)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create pkce", err)
		return nil, err
	}

	err = u.pkceRepo.Save(ctx, pkce)
	if err != nil {
		logger.ErrorContext(ctx, "failed to save pkce", err)
		return nil, err
	}

	return &GeneratePKCEResponse{
		State:         pkce.State.String(),
		CodeChallenge: challange.String(),
	}, nil
}
