package query_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/usecase/loads/query"
	"github.com/karavanix/karavantrack-api-server/pkg/s3"
)

// fakeAttachmentRepo implements domain.AttachmentRepository. Only FindByIDs
// is exercised by the URL resolver; everything else panics.
type fakeAttachmentRepo struct {
	byID map[uuid.UUID]*domain.Attachment
	err  error
}

func (r *fakeAttachmentRepo) Save(ctx context.Context, attachment *domain.Attachment) error {
	panic("not implemented")
}
func (r *fakeAttachmentRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Attachment, error) {
	panic("not implemented")
}
func (r *fakeAttachmentRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Attachment, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make(map[uuid.UUID]*domain.Attachment, len(ids))
	for _, id := range ids {
		if att, ok := r.byID[id]; ok {
			out[id] = att
		}
	}
	return out, nil
}
func (r *fakeAttachmentRepo) FindAllByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Attachment, error) {
	panic("not implemented")
}
func (r *fakeAttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	panic("not implemented")
}

func newTestS3Client(t *testing.T) *s3.S3Client {
	t.Helper()
	client, err := s3.New()
	if err != nil {
		t.Fatalf("s3.New() error = %v", err)
	}
	return client
}

// TestAttachmentURLResolver_PublicGetsStableURL covers the load-detail POD
// gallery case: a public attachment resolves to a direct, stable URL (no
// network call — PresignGet/EndpointURL are pure local computations).
func TestAttachmentURLResolver_PublicGetsStableURL(t *testing.T) {
	id := uuid.New()
	repo := &fakeAttachmentRepo{byID: map[uuid.UUID]*domain.Attachment{
		id: mustAttachment(t, id, domain.VisibilityPublic, "public-bucket", "pod/photo.jpg"),
	}}

	urls := query.ResolveAttachmentURLsForTest(context.Background(), repo, newTestS3Client(t), []uuid.UUID{id})

	got, ok := urls[id]
	if !ok {
		t.Fatalf("resolve() did not return a URL for public attachment %v", id)
	}
	if !strings.Contains(got, "public-bucket") || !strings.Contains(got, "pod/photo.jpg") {
		t.Fatalf("resolve() = %q, want it to reference bucket+key directly for a public attachment", got)
	}
}

// TestAttachmentURLResolver_PrivateGetsPresignedURL covers the case a POD
// photo was uploaded private (the sensible default): the shipper side still
// gets a URL, because access here rides on the caller already having passed
// CanAccessLoad — not on domain.Attachment.IsOwner.
func TestAttachmentURLResolver_PrivateGetsPresignedURL(t *testing.T) {
	id := uuid.New()
	repo := &fakeAttachmentRepo{byID: map[uuid.UUID]*domain.Attachment{
		id: mustAttachment(t, id, domain.VisibilityPrivate, "private-bucket", "pod/photo.jpg"),
	}}

	urls := query.ResolveAttachmentURLsForTest(context.Background(), repo, newTestS3Client(t), []uuid.UUID{id})

	got, ok := urls[id]
	if !ok {
		t.Fatalf("resolve() did not return a URL for private attachment %v", id)
	}
	if !strings.Contains(got, "X-Amz-Signature") {
		t.Fatalf("resolve() = %q, want a presigned URL for a private attachment", got)
	}
}

// TestAttachmentURLResolver_RepoErrorOmitsAllURLs ensures a failure to batch
// -load attachments degrades to "no photos in this response" rather than
// failing the whole load-detail call.
func TestAttachmentURLResolver_RepoErrorOmitsAllURLs(t *testing.T) {
	id := uuid.New()
	repo := &fakeAttachmentRepo{err: context.DeadlineExceeded}

	urls := query.ResolveAttachmentURLsForTest(context.Background(), repo, newTestS3Client(t), []uuid.UUID{id})

	if len(urls) != 0 {
		t.Fatalf("resolve() = %v, want empty map on repo error", urls)
	}
}

func mustAttachment(t *testing.T, id uuid.UUID, visibility domain.Visibility, bucket, key string) *domain.Attachment {
	t.Helper()
	att, err := domain.NewAttachment(id, uuid.New(), visibility, "photo", "jpg", 1024, "image/jpeg", bucket, key)
	if err != nil {
		t.Fatalf("domain.NewAttachment() error = %v", err)
	}
	return att
}
