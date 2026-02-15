package outbox

import (
	"context"
	"errors"
	"sync"
	"time"
)

type WorkerConfig struct {
	WorkerID     string
	PollInterval time.Duration
	BatchSize    int
	Concurrency  int
	Lease        time.Duration
	Backoff      Backoff
	Now          func() time.Time
}

type Worker struct {
	store     Store
	publisher Publisher
	cfg       WorkerConfig
}

func NewWorker(store Store, publisher Publisher, cfg WorkerConfig) (*Worker, error) {
	if store == nil || publisher == nil {
		return nil, errors.New("nil dependency")
	}
	if cfg.WorkerID == "" {
		return nil, errors.New("missing WorkerID")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 8
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 250 * time.Millisecond
	}
	if cfg.Lease <= 0 {
		cfg.Lease = 30 * time.Second
	}
	if cfg.Backoff == nil {
		cfg.Backoff = ExponentialBackoff{Base: 500 * time.Millisecond, Max: 30 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}

	return &Worker{store: store, publisher: publisher, cfg: cfg}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	sem := make(chan struct{}, w.cfg.Concurrency)
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
		}

		now := w.cfg.Now().UTC()

		claimed, err := w.store.ClaimReady(ctx, now, w.cfg.BatchSize, w.cfg.WorkerID, w.cfg.Lease)
		if err != nil {
			if sleepCtx(ctx, w.cfg.PollInterval) != nil {
				wg.Wait()
				return ctx.Err()
			}
			continue
		}

		if len(claimed) == 0 {
			if sleepCtx(ctx, w.cfg.PollInterval) != nil {
				wg.Wait()
				return ctx.Err()
			}
			continue
		}

		for _, ce := range claimed {
			ce := ce

			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				w.processOne(ctx, ce)
			}()
		}
	}
}

func (w *Worker) processOne(ctx context.Context, ce ClaimedEvent) {
	res, err := w.publisher.Publish(ctx, ce.Event)

	now := w.cfg.Now().UTC()

	if err == nil && res.Success {
		_ = w.store.MarkSent(ctx, ce.Event.ID, ce.ClaimToken, now)
		return
	}

	msg := ""
	if err != nil {
		msg = err.Error()
	} else {
		msg = res.Error
	}
	delay := w.cfg.Backoff.NextDelay(ce.Attempt.Count)
	next := now.Add(delay)

	_ = w.store.MarkFailed(ctx, ce.Event.ID, ce.ClaimToken, msg, next, now)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
