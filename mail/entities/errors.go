package entities

import "errors"

type MailError error

var (
	ErrInvalidConfig  MailError = errors.New("invalid config")
	ErrSendMailFailed MailError = errors.New("send mail failed")
)
