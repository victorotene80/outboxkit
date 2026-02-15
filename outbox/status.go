package outbox

import "time"

type Status uint8

const (
	StatusPending Status = iota + 1
	StatusInProgress
	StatusSent
	StatusFailed
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "PENDING"
	case StatusInProgress:
		return "IN_PROGRESS"
	case StatusSent:
		return "SENT"
	case StatusFailed:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

type Attempt struct {
	Count       int
	LastError   string
	NextRetryAt time.Time
	UpdatedAt   time.Time
}

type ClaimedEvent struct {
	Event      Event
	Status     Status
	Attempt    Attempt
	ClaimedBy  string
	ClaimedAt  time.Time
	LeaseUntil time.Time

	ClaimToken ClaimToken
}

type ClaimToken string

func (t ClaimToken) String() string { return string(t) }
