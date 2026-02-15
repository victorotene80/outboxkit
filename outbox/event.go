package outbox

import (
	"strings"
	"time"
)

type EventID string

func (id EventID) String() string { return string(id) }

func (id EventID) Valid() bool {
	v := strings.TrimSpace(string(id))
	return v != "" && len(v) <= 128
}

type Event struct {
	ID          EventID
	Stream      string
	Type        string
	AggregateID string
	DedupKey    string
	Payload     []byte
	Meta        map[string]string
	CreatedAt   time.Time
}

func (e Event) Valid() bool {
	if !e.ID.Valid() {
		return false
	}
	if strings.TrimSpace(e.Stream) == "" || len(e.Stream) > 64 {
		return false
	}
	if strings.TrimSpace(e.Type) == "" || len(e.Type) > 128 {
		return false
	}
	if strings.TrimSpace(e.DedupKey) == "" || len(e.DedupKey) > 256 {
		return false
	}
	if len(e.Payload) == 0 {
		return false
	}
	return true
}
