package tasks

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const TaskSendAddCarrierToCompany = "task:send:add_carrier_to_company"

type AddCarrierToCompanyPayload struct {
	ActorID   string `json:"actor_id"`
	CompanyID string `json:"company_id"`
	CarrierID string `json:"carrier_id"`
	Alias     string `json:"alias"`
}

func NewSendAddCarrierToCompanyTask(payload *AddCarrierToCompanyPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal add carrier to company payload: %w", err)
	}

	return asynq.NewTask(TaskSendAddCarrierToCompany, data, asynq.MaxRetry(3), asynq.Queue("default")), nil
}
