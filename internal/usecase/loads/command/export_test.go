package command

import (
	"context"

	"github.com/google/uuid"
	"github.com/karavanix/karavantrack-api-server/internal/domain"
)

// OwnerSideRecipientsForTest exposes the unexported ownerSideRecipients to
// the command_test package (Go's standard "export_test.go" pattern), so the
// notification fan-out rule can be tested without going through asynq.
func OwnerSideRecipientsForTest(ctx context.Context, companyMembersRepo domain.CompanyMemberRepository, load *domain.Load) []uuid.UUID {
	return ownerSideRecipients(ctx, companyMembersRepo, load)
}
