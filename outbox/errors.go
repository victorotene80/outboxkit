package outbox

import "errors"

var (
	ErrInvalidEvent   = errors.New("invalid event")
	ErrNotOwner       = errors.New("not owner / claim token mismatch")
	ErrAlreadySent    = errors.New("already sent")
	ErrStoreFailure   = errors.New("store failure")
	ErrPublishFailure = errors.New("publish failure")
)
