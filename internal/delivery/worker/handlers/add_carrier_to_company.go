package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/companies/command"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

func (h *Handler) AddCarrierToCompany(ctx context.Context, t *asynq.Task) error {
	var payload tasks.AddCarrierToCompanyPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal add carrier to company payload: %w", err)
	}

	logger.InfoContext(ctx, "processing add carrier to company task",
		"actor_id", payload.ActorID,
		"company_id", payload.CompanyID,
		"carrier_id", payload.CarrierID,
		"alias", payload.Alias,
	)

	err := h.companyUsecase.Command.AddCarrier(ctx, payload.ActorID, payload.CompanyID, &command.AddCarrierRequest{
		CarrierID: payload.CarrierID,
		Alias:     payload.Alias,
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to add carrier to company", err,
			"actor_id", payload.ActorID,
			"company_id", payload.CompanyID,
			"carrier_id", payload.CarrierID,
		)
		return err
	}

	return nil
}
