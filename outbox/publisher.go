package outbox

import "context"

type PublishResult struct {
	Success bool
	Error   string
}

type Publisher interface {
	Publish(ctx context.Context, e Event) (PublishResult, error)
}
