package query

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// DownloadFileUsecase streams an attachment's file bytes from S3.
//
// IMPORTANT: intentionally does NOT set a context timeout. The S3 GetObject call
// returns a streaming HTTP reader — cancelling the context on return would close
// the connection before the caller can copy the stream to the HTTP response.
type DownloadFileUsecase struct {
	attachmentsRepo domain.AttachmentRepository
	s3              *s3.S3Client
}

func NewDownloadFileUsecase(attachmentsRepo domain.AttachmentRepository, s3Client *s3.S3Client) *DownloadFileUsecase {
	return &DownloadFileUsecase{
		attachmentsRepo: attachmentsRepo,
		s3:              s3Client,
	}
}

type DownloadFileResponse struct {
	ID         string
	Filename   string
	MimeType   string
	Visibility string
	Status     string
	CreatedAt  time.Time
}

func (u *DownloadFileUsecase) DownloadFile(ctx context.Context, requesterID string, attachmentID string) (_ *DownloadFileResponse, _ io.Reader, err error) {
	ctx, end := otlp.Start(ctx, otel.Tracer("internal.usecases.attachments"), "domain.DownloadFile",
		attribute.String("user_id", requesterID),
		attribute.String("attachment_id", attachmentID),
	)
	defer func() { end(err) }()

	var input struct {
		userID       uuid.UUID
		attachmentID uuid.UUID
	}
	{
		input.userID, err = uuid.Parse(requesterID)
		if err != nil {
			return nil, nil, inerr.NewErrValidation("user_id", "user id must be uuid")
		}
		input.attachmentID, err = uuid.Parse(attachmentID)
		if err != nil {
			return nil, nil, inerr.NewErrValidation("attachment_id", "attachment id must be uuid")
		}
	}

	attachment, err := u.attachmentsRepo.FindByID(ctx, input.attachmentID)
	if err != nil {
		return nil, nil, err
	}

	if !attachment.IsPublic() && !attachment.IsOwner(input.userID) {
		return nil, nil, inerr.ErrorPermissionDenied
	}

	reader, _, err := u.s3.GetObject(ctx, attachment.ObjectBucket, attachment.ObjectKey)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get object from s3", err)
		return nil, nil, err
	}

	return &DownloadFileResponse{
		ID:         attachment.ID.String(),
		Filename:   attachment.FileNameWithExtension(),
		MimeType:   attachment.FileMime,
		Visibility: attachment.Visibility.String(),
		Status:     attachment.Status.String(),
		CreatedAt:  attachment.CreatedAt,
	}, reader, nil
}
