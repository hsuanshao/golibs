/*
Package ctx extends standard context to support logging.
For context detail, see https://golang.org/pkg/context/
*/
package ctx

//
import (
	"context" // <<< Add this import
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/hsuanshao/golibs/validator"
)

var (
	// logExcept set keys that should be except from log
	logExcept = []string{"accessToken", "password", "authorization", "Authorization"}

	// settleLogger as shared parameter for handle and control logger as expected setup
	settleLogger *logrus.Logger
)

// CTX extends Google's context to support logging methods.
type CTX struct {
	context.Context
	logrus.FieldLogger
}

// Background returns a non-nil, empty Context. It is never canceled, has no values, and
// has no deadline. It is typically used by the main function, initialization, and tests,
// and as the top-level Context for incoming requests
func Background() CTX {
	if settleLogger == nil {
		settleLogger = logrus.StandardLogger() // Create a new logger instance

		// Configuration for the hook
		hookConfig := &HookConfig{LogFormat: &logrus.JSONFormatter{}}
		settleLogger.SetFormatter(hookConfig.LogFormat)

		// If your hook 'h' is NOT intended to write to the console (e.g., it only sends logs to a
		// remote service or only writes to a file), and you still want console logging
		// from the logger itself, you would NOT set SetOutput(io.Discard) and instead ensure
		// the logger's output and formatter are configured as desired (they default to os.Stderr
		// and logrus.TextFormatter respectively for a logrus.New() instance).
	}
	return CTX{
		Context:     context.Background(),
		FieldLogger: settleLogger,
	}
}

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent CTX, key string, val interface{}) CTX {
	var keyi interface{} = key
	if validator.IsInStringSlice(logExcept, key) {
		return CTX{
			Context:     context.WithValue(parent, keyi, val),
			FieldLogger: parent.FieldLogger,
		}
	}

	return CTX{
		Context:     context.WithValue(parent, keyi, val),
		FieldLogger: parent.FieldLogger.WithField(key, val),
	}
}

// WithValues returns a copy of parent in which the values associated with keys are vals.
func WithValues(parent CTX, kvs map[string]interface{}) CTX {
	c := parent
	for k, v := range kvs {
		c = WithValue(c, k, v)
	}

	return c
}

// WithCancel returns a copy of parent with added cancel function
func WithCancel(parent CTX) (CTX, context.CancelFunc) {
	newCtx, cFunc := context.WithCancel(parent)
	return CTX{
		Context:     newCtx,
		FieldLogger: parent.FieldLogger,
	}, cFunc
}

// WithTimeout returns a copy of parent with timeout condition
// and cancel function
func WithTimeout(parent CTX, d time.Duration) (CTX, context.CancelFunc) {
	newCtx, cFunc := context.WithTimeout(parent, d)
	return CTX{
		Context:     newCtx,
		FieldLogger: parent.FieldLogger,
	}, cFunc
}

// WithDeadline returns a copy of parent with deadline condition
// and cancel function
func WithDeadline(parent CTX, deadline time.Time) (CTX, context.CancelFunc) {
	newCtx, cFunc := context.WithDeadline(parent, deadline)
	return CTX{
		Context:     newCtx,
		FieldLogger: parent.FieldLogger,
	}, cFunc
}

// TODO returns a copy of parent in which the value associated with key is val.
func TODO(parent CTX) CTX {
	return CTX{
		Context:     context.TODO(),
		FieldLogger: parent.FieldLogger,
	}
}

// resolveFormatter returns the configured formatter or a sensible default.
func resolveFormatter(logConfig *HookConfig) logrus.Formatter {
	if logConfig.LogFormat != nil {
		return logConfig.LogFormat
	}
	if logConfig.Environment == "localhost" {
		return &logrus.TextFormatter{ForceColors: true}
	}
	return &logrus.JSONFormatter{}
}

// maxExportLevel returns the most verbose (highest numeric value) level from the slice.
func maxExportLevel(levels []logrus.Level) logrus.Level {
	max := levels[0]
	for _, lvl := range levels[1:] {
		if lvl > max {
			max = lvl
		}
	}
	return max
}

// SetLogger to change ctx logger setup
func (c CTX) SetLogger(logConfig *HookConfig) CTX {
	logger := logrus.New()
	if logConfig != nil {
		logger.SetFormatter(resolveFormatter(logConfig))

		if len(logConfig.ExportLevel) > 0 {
			logger.SetLevel(maxExportLevel(logConfig.ExportLevel))
		}

		if logConfig.SetAsStdOutput {
			logger.SetOutput(os.Stdout)
		}

		h, err := NewHook(logConfig)
		if err != nil {
			logrus.WithField("err", err).Fatal("prepare log hook failed")
		}
		logger.AddHook(h)
	}
	return CTX{
		Context:     c.Context,
		FieldLogger: logger,
	}
}
