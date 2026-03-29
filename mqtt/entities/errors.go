package entities

import "errors"

// MQTTError is the sentinel error type for this package.
type MQTTError error

var (
	ErrInvalidConfig    MQTTError = errors.New("invalid mqtt config")
	ErrConnectFailed    MQTTError = errors.New("failed to connect to mqtt broker")
	ErrDisconnectFailed MQTTError = errors.New("failed to disconnect from mqtt broker")
	ErrPublishFailed    MQTTError = errors.New("failed to publish mqtt message")
	ErrSubscribeFailed  MQTTError = errors.New("failed to subscribe to mqtt topic")
	ErrNotConnected     MQTTError = errors.New("mqtt client is not connected")
)
