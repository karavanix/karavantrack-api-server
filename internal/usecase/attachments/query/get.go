package query

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/inerr"
	"github.com/karavanix/karavantrack-api-server/pkg/otlp"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
	"github.com/karavanix/karavantrack-api-server/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// GetUsecase fetches attachment metadata and resolves its public URL when applicable.
type GetUsecase struct {
	contextDuration time.Duration
	attachmentsRepo domain.AttachmentRepository
	s3              *s3.S3Client
}

func NewGetUsecase(contextDuration time.Duration, attachmentsRepo domain.AttachmentRepository, s3Client *s3.S3Client) *GetUsecase {
	return &GetUsecase{
		contextDuration: contextDuration,
		attachmentsRepo: attachmentsRepo,
		s3:              s3Client,
	}
}

type AttachmentResponse struct {
	ID         string
	Filename   string
	MimeType   string
	Visibility string
	Status     string
	PublicURL  string
	CreatedAt  time.Time
}

func (u *GetUsecase) Get(ctx context.Context, requesterID string, attachmentID string) (_ *AttachmentResponse, err error) {
	ctx, cancel := context.WithTimeout(ctx, u.contextDuration)
	defer cancel()

	ctx, end := otlp.Start(ctx, otel.Tracer("internal.usecases.attachments"), "attachments.Get",
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
			return nil, inerr.NewErrValidation("user_id", "user id must be uuid")
		}
		input.attachmentID, err = uuid.Parse(attachmentID)
		if err != nil {
			return nil, inerr.NewErrValidation("attachment_id", "attachment id must be uuid")
		}
	}

	attachment, err := u.attachmentsRepo.FindByID(ctx, input.attachmentID)
	if err != nil {
		return nil, err
	}

	if !attachment.IsPublic() && !attachment.IsOwner(input.userID) {
		return nil, inerr.ErrorPermissionDenied
	}

	return &AttachmentResponse{
		ID:         attachment.ID.String(),
		Filename:   attachment.FileNameWithExtension(),
		MimeType:   attachment.FileMime,
		Visibility: attachment.Visibility.String(),
		Status:     attachment.Status.String(),
		PublicURL:  utils.If(attachment.IsPublic(), fmt.Sprintf("%s/%s/%s", u.s3.EndpointURL(), attachment.ObjectBucket, attachment.ObjectKey), ""),
		CreatedAt:  attachment.CreatedAt,
	}, nil
}
