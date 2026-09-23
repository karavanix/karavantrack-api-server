package leads

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/leads/command"
)

type Command struct {
	*command.SubmitLeadUsecase
}

type Usecase struct {
	Command Command
}

func NewUsecase(contextDuration time.Duration, leadsRepo domain.LeadRepository) *Usecase {
	return &Usecase{
		Command: Command{
			SubmitLeadUsecase: command.NewSubmitLeadUsecase(contextDuration, leadsRepo),
		},
	}
}
