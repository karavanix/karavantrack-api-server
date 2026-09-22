package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
)

// attachmentURLResolver turns POD attachment IDs referenced in a load's
// status history into URLs the client can load directly — a public
// attachment gets its stable public-bucket URL, a private one a short-lived
// presigned URL. Without this, a client would have to call
// GET /attachments/{id} once per photo (N+1) to render a history timeline.
//
// Access control for the URLs themselves rides on the caller having already
// passed CanAccessLoad for the load this history belongs to (see
// GetUsecase/GetActiveUsecase) — the attachment's own owner-only visibility
// rule (domain.Attachment.IsOwner) does not apply here, on purpose: a POD
// photo is shared by nature once it's attached to a load's history, and the
// shipper side viewing it was never the uploader.
type attachmentURLResolver struct {
	attachmentsRepo domain.AttachmentRepository
	s3              *s3.S3Client
}

func newAttachmentURLResolver(attachmentsRepo domain.AttachmentRepository, s3Client *s3.S3Client) *attachmentURLResolver {
	return &attachmentURLResolver{attachmentsRepo: attachmentsRepo, s3: s3Client}
}

// resolve returns a best-effort id->URL map. An attachment that fails to
// resolve (deleted, presign error, etc.) is simply omitted rather than
// failing the whole load-detail response.
func (r *attachmentURLResolver) resolve(ctx context.Context, ids []uuid.UUID) map[uuid.UUID]string {
	urls := make(map[uuid.UUID]string, len(ids))
	if r == nil || len(ids) == 0 {
		return urls
	}

	attachmentsByID, err := r.attachmentsRepo.FindByIDs(ctx, ids)
	if err != nil {
		logger.ErrorContext(ctx, "failed to resolve attachment URLs for load history", err)
		return urls
	}

	for id, att := range attachmentsByID {
		if att.IsPublic() {
			urls[id] = fmt.Sprintf("%s/%s/%s", r.s3.EndpointURL(), att.ObjectBucket, att.ObjectKey)
			continue
		}

		presigned, err := r.s3.PresignGet(ctx, att.ObjectBucket, att.ObjectKey, 0)
		if err != nil {
			logger.ErrorContext(ctx, "failed to presign attachment URL", err, "attachment_id", id.String())
			continue
		}
		urls[id] = presigned.String()
	}

	return urls
}
