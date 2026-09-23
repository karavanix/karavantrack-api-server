package query

import (
	"context"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
)

// ResolveAttachmentURLsForTest exposes attachmentURLResolver.resolve to the
// query_test package (Go's standard "export_test.go" pattern).
func ResolveAttachmentURLsForTest(ctx context.Context, attachmentsRepo domain.AttachmentRepository, s3Client *s3.S3Client, ids []uuid.UUID) map[uuid.UUID]string {
	return newAttachmentURLResolver(attachmentsRepo, s3Client).resolve(ctx, ids)
}
