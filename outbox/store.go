package outbox

import (
	"context"
	"time"
)

type Store interface {
	Insert(ctx context.Context, e Event, now time.Time) error

	ClaimReady(
		ctx context.Context,
		now time.Time,
		limit int,
		workerID string,
		lease time.Duration,
	) ([]ClaimedEvent, error)

	MarkSent(
		ctx context.Context,
		id EventID,
		token ClaimToken,
		now time.Time,
	) error

	MarkFailed(
		ctx context.Context,
		id EventID,
		token ClaimToken,
		errMsg string,
		nextRetryAt time.Time,
		now time.Time,
	) error
}
