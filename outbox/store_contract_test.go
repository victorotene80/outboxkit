package outbox

import (
	"context"
	"testing"
	"time"
)

func TestStoreInterfaceCompiles(t *testing.T) {
	var _ Store = (*dummyStore)(nil)
	_ = context.Background()
	_ = time.Now()
}

type dummyStore struct{}

func (d *dummyStore) Insert(ctx context.Context, e Event, now time.Time) error { return nil }
func (d *dummyStore) ClaimReady(ctx context.Context, now time.Time, limit int, workerID string, lease time.Duration) ([]ClaimedEvent, error) {
	return nil, nil
}
func (d *dummyStore) MarkSent(ctx context.Context, id EventID, token ClaimToken, now time.Time) error {
	return nil
}
func (d *dummyStore) MarkFailed(ctx context.Context, id EventID, token ClaimToken, errMsg string, nextRetryAt time.Time, now time.Time) error {
	return nil
}
