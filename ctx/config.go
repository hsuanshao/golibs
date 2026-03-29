package ctx

import (
	"errors"
)

/**
Here define a configuration for different environment
usage
*/

var (
	// ErrInvalidConfig define error for invalid config
	ErrInvalidConfig = errors.New("invalid config")
)

type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	FatalLevel Level = "fatal"
	PanicLevel Level = "panic"
)

type Formatter string

const (
	JSONFormatter Formatter = "json"
	TextFormatter Formatter = "text"
)

// Configuration struct define configuration to ctx.CTX
type Configuration struct {
	LogFormat Formatter `json:"log_format"`
	LogLevel  Level     `json:"log_level"`
}

// Config define
func Config(conf *Configuration) error {
	if conf == nil {
		return ErrInvalidConfig
	}
	return nil
}
