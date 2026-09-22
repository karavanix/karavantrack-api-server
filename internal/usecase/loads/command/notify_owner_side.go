package command

import (
	"context"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/karavanix/karavantrack-api-server/internal/domain"
	"github.com/karavanix/karavantrack-api-server/internal/tasks"
	"github.com/karavanix/karavantrack-api-server/pkg/logger"
)

// ownerSideRecipients returns the set of users who should be notified about a
// status change on the shipper side of a load: the load's own creator plus
// every owner/admin of the load's company. Company membership can change
// (loads.member_id is ON DELETE SET NULL when a dispatcher leaves), so this
// is computed fresh on every call rather than cached on the load.
func ownerSideRecipients(ctx context.Context, companyMembersRepo domain.CompanyMemberRepository, load *domain.Load) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, 4)
	recipients := make([]uuid.UUID, 0, 4)

	add := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		recipients = append(recipients, id)
	}

	add(load.MemberID)

	members, err := companyMembersRepo.FindByCompanyID(ctx, load.CompanyID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to list company members for notification fan-out", err,
			"company_id", load.CompanyID.String(),
		)
		return recipients
	}

	for _, member := range members {
		if member.Role == domain.MemberRoleOwner || member.Role == domain.MemberRoleAdmin {
			add(member.MemberID)
		}
	}

	return recipients
}

// enqueueOwnerSidePush sends a push notification to everyone on the shipper
// side of a load (see ownerSideRecipients). It is best-effort: a failure to
// enqueue for one recipient is logged and does not stop the others or fail
// the calling usecase, matching how other side-effect tasks (e.g. Assign) are
// treated elsewhere in this package.
func enqueueOwnerSidePush(ctx context.Context, taskQueue *asynq.Client, companyMembersRepo domain.CompanyMemberRepository, load *domain.Load, notification tasks.PushNotification) {
	for _, userID := range ownerSideRecipients(ctx, companyMembersRepo, load) {
		task, err := tasks.NewSendPushNotificationTask(userID.String(), notification)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create push notification task", err, "user_id", userID.String())
			continue
		}
		if _, err := taskQueue.EnqueueContext(ctx, task); err != nil {
			logger.ErrorContext(ctx, "failed to enqueue push notification", err, "user_id", userID.String())
		}
	}
}
