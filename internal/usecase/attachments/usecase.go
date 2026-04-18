package attachments

import (
	"time"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/attachments/command"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/attachments/query"
	"github.com/karavanix/karavantrack-api-server/pkg/config"
	"github.com/karavanix/karavantrack-api-server/pkg/database/postgres"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
)

// Command groups all write-side usecases (CUD operations).
type Command struct {
	*command.UploadImageUsecase
	*command.UploadFileUsecase
}

// Query groups all read-side usecases.
type Query struct {
	*query.GetUsecase
	*query.DownloadFileUsecase
}

// Usecase is the top-level CQRS entry-point for the attachments domain.
type Usecase struct {
	Command Command
	Query   Query
}

func NewUsecase(
	contextDuration time.Duration,
	cfg *config.Config,
	txManager postgres.TxManager,
	attachmentsRepo domain.AttachmentRepository,
	s3Client *s3.S3Client,
) *Usecase {
	return &Usecase{
		Command: Command{
			UploadImageUsecase: command.NewUploadImageUsecase(contextDuration, cfg, txManager, attachmentsRepo, s3Client),
			UploadFileUsecase:  command.NewUploadFileUsecase(contextDuration, cfg, txManager, attachmentsRepo, s3Client),
		},
		Query: Query{
			GetUsecase:          query.NewGetUsecase(contextDuration, attachmentsRepo, s3Client),
			DownloadFileUsecase: query.NewDownloadFileUsecase(attachmentsRepo, s3Client),
		},
	}
}
