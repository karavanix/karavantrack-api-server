package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const TaskMatchLoadTrack = "task:routing:match_load_track"

type MatchLoadTrackPayload struct {
	LoadID string `json:"load_id"`
}

// NewMatchLoadTrackTask builds a matching task for a load, run after
// `debounce`.
//
// unique adds asynq's Unique lock on type+payload+queue, so a burst of
// batches gives one match: while the lock is held, enqueueing the same load
// returns ErrDuplicateTask. For a delayed task asynq holds that lock until
// processAt+TTL (client.go, schedule), i.e. through the run itself, and a
// successful run releases it. So a batch stored while a match runs is
// rejected; the match covers it by checking for newer points when it's done
// and enqueueing a follow-up without the lock (see MatchLoadTrackUsecase).
// A task that fails for good doesn't block the load for longer than the TTL,
// unlike a fixed TaskID, which stays taken while the archived task exists.
func NewMatchLoadTrackTask(loadID string, debounce time.Duration, unique bool) (*asynq.Task, error) {
	data, err := json.Marshal(&MatchLoadTrackPayload{LoadID: loadID})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal match load track payload: %w", err)
	}

	opts := []asynq.Option{
		asynq.MaxRetry(3),
		asynq.Queue("default"),
		asynq.ProcessIn(debounce),
	}
	if unique {
		opts = append(opts, asynq.Unique(debounce))
	}
	return asynq.NewTask(TaskMatchLoadTrack, data, opts...), nil
}

// MatchLoadTrackScheduler enqueues matching of a load's track after new
// points are stored. It's a no-op when matching is disabled (and on a nil
// scheduler).
type MatchLoadTrackScheduler struct {
	queue    *asynq.Client
	enabled  bool
	debounce time.Duration
}

func NewMatchLoadTrackScheduler(queue *asynq.Client, enabled bool, debounce time.Duration) *MatchLoadTrackScheduler {
	return &MatchLoadTrackScheduler{queue: queue, enabled: enabled, debounce: debounce}
}

// Schedule must be called after the points are committed.
func (s *MatchLoadTrackScheduler) Schedule(ctx context.Context, loadID string) error {
	return s.enqueue(ctx, loadID, true)
}

// ScheduleFollowUp enqueues a match without the Unique lock. A match calls
// it when newer points were stored while it ran: their own Schedule was
// rejected by the lock this match was holding.
func (s *MatchLoadTrackScheduler) ScheduleFollowUp(ctx context.Context, loadID string) error {
	return s.enqueue(ctx, loadID, false)
}

func (s *MatchLoadTrackScheduler) enqueue(ctx context.Context, loadID string, unique bool) error {
	if s == nil || !s.enabled {
		return nil
	}
	task, err := NewMatchLoadTrackTask(loadID, s.debounce, unique)
	if err != nil {
		return err
	}
	if _, err := s.queue.EnqueueContext(ctx, task); err != nil && !errors.Is(err, asynq.ErrDuplicateTask) {
		return fmt.Errorf("failed to enqueue match load track task: %w", err)
	}
	return nil
}
