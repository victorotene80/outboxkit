package outbox

import (
	"context"
	"sync"
	"testing"
	"time"
)

type memStore struct {
	mu sync.Mutex

	events map[EventID]ClaimedEvent
}

func newMemStore() *memStore {
	return &memStore{events: make(map[EventID]ClaimedEvent)}
}

func (s *memStore) Insert(ctx context.Context, e Event, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[e.ID] = ClaimedEvent{
		Event:  e,
		Status: StatusPending,
		Attempt: Attempt{
			Count:       0,
			NextRetryAt: time.Time{},
			UpdatedAt:   now,
		},
	}
	return nil
}

func (s *memStore) ClaimReady(ctx context.Context, now time.Time, limit int, workerID string, lease time.Duration) ([]ClaimedEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]ClaimedEvent, 0, limit)
	for id, ce := range s.events {
		if len(out) >= limit {
			break
		}
		if ce.Status == StatusSent {
			continue
		}
		if !ce.Attempt.NextRetryAt.IsZero() && ce.Attempt.NextRetryAt.After(now) {
			continue
		}
		ce.Status = StatusInProgress
		ce.ClaimedBy = workerID
		ce.ClaimedAt = now
		ce.LeaseUntil = now.Add(lease)
		ce.ClaimToken = ClaimToken("tok-" + id.String())
		s.events[id] = ce
		out = append(out, ce)
	}
	return out, nil
}

func (s *memStore) MarkSent(ctx context.Context, id EventID, token ClaimToken, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ce := s.events[id]
	if ce.ClaimToken != token {
		return ErrNotOwner
	}
	ce.Status = StatusSent
	ce.Attempt.UpdatedAt = now
	s.events[id] = ce
	return nil
}

func (s *memStore) MarkFailed(ctx context.Context, id EventID, token ClaimToken, errMsg string, nextRetryAt time.Time, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ce := s.events[id]
	if ce.ClaimToken != token {
		return ErrNotOwner
	}
	ce.Status = StatusFailed
	ce.Attempt.Count++
	ce.Attempt.LastError = errMsg
	ce.Attempt.NextRetryAt = nextRetryAt
	ce.Attempt.UpdatedAt = now
	s.events[id] = ce
	return nil
}

type memPublisher struct {
	mu sync.Mutex

	failFirst bool
	calls     int
}

func (p *memPublisher) Publish(ctx context.Context, e Event) (PublishResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls++
	if p.failFirst && p.calls == 1 {
		return PublishResult{Success: false, Error: "temp"}, nil
	}
	return PublishResult{Success: true}, nil
}

func TestWorker_PublishesAndMarksSent(t *testing.T) {
	store := newMemStore()
	pub := &memPublisher{}

	now := time.Unix(100, 0).UTC()
	e := Event{
		ID:        EventID("e1"),
		Stream:    "payments",
		Type:      "payment.created",
		DedupKey:  "payment:1:created",
		Payload:   []byte(`{"x":1}`),
		CreatedAt: now,
	}
	if !e.Valid() {
		t.Fatalf("event invalid")
	}

	_ = store.Insert(context.Background(), e, now)

	w, err := NewWorker(store, pub, WorkerConfig{
		WorkerID:     "w1",
		BatchSize:    10,
		Concurrency:  1,
		PollInterval: 10 * time.Millisecond,
		Lease:        1 * time.Second,
		Backoff:      ExponentialBackoff{Base: time.Second, Max: time.Second},
		Now:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("worker init: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = w.Run(ctx)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.events["e1"].Status != StatusSent {
		t.Fatalf("expected SENT, got %s", store.events["e1"].Status.String())
	}
}

func TestWorker_RetriesAfterFailure(t *testing.T) {
	store := newMemStore()
	pub := &memPublisher{failFirst: true}

	base := time.Unix(100, 0).UTC()
	e := Event{
		ID:        EventID("e2"),
		Stream:    "payments",
		Type:      "payment.created",
		DedupKey:  "payment:2:created",
		Payload:   []byte(`{"x":2}`),
		CreatedAt: base,
	}
	_ = store.Insert(context.Background(), e, base)

	now := base
	w, _ := NewWorker(store, pub, WorkerConfig{
		WorkerID:     "w1",
		BatchSize:    10,
		Concurrency:  1,
		PollInterval: 10 * time.Millisecond,
		Lease:        1 * time.Second,
		Backoff:      ExponentialBackoff{Base: time.Second, Max: time.Second},
		Now:          func() time.Time { return now },
	})

	ce, _ := store.ClaimReady(context.Background(), now, 1, "w1", time.Second)
	w.processOne(context.Background(), ce[0])

	store.mu.Lock()
	next := store.events["e2"].Attempt.NextRetryAt
	status := store.events["e2"].Status
	store.mu.Unlock()

	if status != StatusFailed {
		t.Fatalf("expected FAILED, got %s", status.String())
	}
	if !next.Equal(base.Add(time.Second)) {
		t.Fatalf("expected next retry = base+1s, got %v", next)
	}

	now = base.Add(time.Second)
	ce2, _ := store.ClaimReady(context.Background(), now, 1, "w1", time.Second)
	w.processOne(context.Background(), ce2[0])

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.events["e2"].Status != StatusSent {
		t.Fatalf("expected SENT after retry, got %s", store.events["e2"].Status.String())
	}
}
