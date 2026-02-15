package outbox

import "time"

type Backoff interface {
	NextDelay(attemptCount int) time.Duration
}

type ExponentialBackoff struct {
	Base time.Duration
	Max  time.Duration
}

func (b ExponentialBackoff) NextDelay(attemptCount int) time.Duration {
	if b.Base <= 0 {
		return 0
	}
	if attemptCount < 0 {
		attemptCount = 0
	}

	d := b.Base
	for i := 0; i < attemptCount; i++ {
		if b.Max > 0 && d >= b.Max/2 {
			d = b.Max
			break
		}
		d *= 2
	}

	if b.Max > 0 && d > b.Max {
		return b.Max
	}
	return d
}
