package messaging

import "errors"

var (
	ErrMessageNotAcknowledged       = errors.New("message was not acknowledged")
	ErrMessageAcknowledgedWithError = errors.New("message was processed and acknowledged but with error")
)
