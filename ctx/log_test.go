package ctx

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var ()

func newLogger() *logrus.Logger {
	return &logrus.Logger{
		Out:       io.Discard,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
}

func BenchmarkLogger(b *testing.B) {
	logger := newLogger()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		logger.Info("hello world")
	}
}

func BenchmarkLevelLogger(b *testing.B) {
	logger := newLogger()
	h, _ := NewHook(&HookConfig{})
	logger.AddHook(h)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		logger.Info("hello world")
	}
}

func BenchmarkLoggerWithField(b *testing.B) {
	logger := newLogger()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		logger.WithField("key", "k").Info("hello world")
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger := newLogger()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		logger.WithFields(logrus.Fields{"key": "k"}).Info("hello world")
	}
}

// Hook Rollbar
func TestLevelLogger(t *testing.T) {
	var b bytes.Buffer
	logger := &logrus.Logger{
		Out:       &b,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	h, _ := NewHook(&HookConfig{Environment: "localhost", LogFormat: &logrus.TextFormatter{ForceColors: true}})
	logger.AddHook(h)
	logger.Info("hi-rollbar")

	m := map[string]interface{}{}
	err := json.Unmarshal(b.Bytes(), &m)
	assert.NoError(t, err)
	assert.Equal(t, "log_test.go", m["file"])
	assert.Equal(t, float64(71), m["line"])
	t.Logf("map %T, num %T", m["line"], 71)
	assert.Equal(t, m["func"], "ctx.TestLevelLogger")
}

func strToPtr(input string) (output *string) {
	return &input
}

// // Test SetLogger
// func TestSetLogger(t *testing.T) {
// 	testCase := []struct {
// 		Case          string
// 		Configs       []*Config
// 		Env           string
// 		ServiceName   string
// 		GoProjectPath string
// 		LogBaseLevel  string
// 		LogExportPath *string
// 		MockFunc      func()
// 		ExpErr        error
// 	}{
// 		{
// 			Case: "production developer",
// 			Configs: []*Config{
// 				{
// 					Environment:   "production",
// 					ServiceName:   "ctx test case 1",
// 					ProjectFolder: "github.com/hsuanshao/golibs/",
// 					ExportLevel:   logrus.AllLevels,
// 					ExportToFile:  strToPtr("./log/ctx_all_log.log"),
// 				},
// 				{
// 					Environment:   "production",
// 					ServiceName:   "ctx test case 1",
// 					ProjectFolder: "github.com/hsuanshao/golibs/",
// 					ExportLevel:   []logrus.Level{logrus.InfoLevel, logrus.DebugLevel, logrus.WarnLevel, logrus.TraceLevel},
// 					ExportToFile:  strToPtr("./log/ctx_access.log"),
// 				},
// 				{
// 					Environment:   "production",
// 					ServiceName:   "ctx test case error log",
// 					ProjectFolder: "github.com/hsuanshao/golibs/",
// 					ExportLevel:   []logrus.Level{logrus.PanicLevel, logrus.FatalLevel, logrus.ErrorLevel},
// 					ExportToFile:  strToPtr("./log/ctx_error.log"),
// 				},
// 				{
// 					Environment:    "production",
// 					ServiceName:    "ctx test case std log",
// 					ProjectFolder:  "github.com/hsuanshao/golibs/",
// 					ExportLevel:    logrus.AllLevels,
// 					ExportToFile:   nil,
// 					SetAsStdOutput: true,
// 				},
// 			},
// 			ExpErr: nil,
// 		},
// 		{
// 			Case:    "without config",
// 			Configs: nil,
// 			ExpErr:  fmt.Errorf(`set logger without config`),
// 		},
// 		{
// 			Case: "standard log",
// 			Configs: []*Config{
// 				{
// 					Environment:    "localhost",
// 					LogFormat:      &logrus.TextFormatter{ForceColors: true},
// 					ExportToFile:   nil,
// 					SetAsStdOutput: false,
// 				},
// 			},
// 			ExpErr: nil,
// 		},
// 	}

// 	for i, c := range testCase {
// 		expErr := SetLogger(c.Configs...)
// 		if expErr != nil && c.ExpErr.Error() != expErr.Error() {
// 			t.Errorf("expected error doesn't match on case %d, %s: %s", i, c.Case, expErr.Error())
// 		}

// 		if c.ExpErr == nil && c.ExpErr == expErr {
// 			tCTX := Background()
// 			testCTX, cancel := WithTimeout(tCTX, 1*time.Second)
// 			defer cancel()
// 			testCTX.WithFields(logrus.Fields{"case number": i, "case name": c.Case}).Info("check logger with info")
// 			testCTX.WithFields(logrus.Fields{"case number": i, "case name": c.Case}).Warn("check logger warning")
// 			testCTX.WithFields(logrus.Fields{"case number": i, "case name": c.Case}).Error("check logger error")
// 		}
// 	}
// }

// MOCK logrus add hook
// Hook is a hook designed for dealing with logs in test scenarios.
type MockHook struct {
	// Entries is an array of all entries that have been received by this hook.
	// For safe access, use the AllEntries() method, rather than reading this
	// value directly.
	Entries []logrus.Entry
	mu      sync.RWMutex
}

// NewGlobal installs a test hook for the global logger.
func NewGlobal() *MockHook {

	hook := new(MockHook)
	logrus.AddHook(hook)

	return hook

}

// NewLocal installs a test hook for a given local logger.
func NewLocal(logger *logrus.Logger) *MockHook {

	hook := new(MockHook)
	logger.Hooks.Add(hook)

	return hook

}

// NewNullLogger creates a discarding logger and installs the test hook.
func NewNullLogger() (*logrus.Logger, *MockHook) {

	logger := logrus.New()
	logger.Out = io.Discard

	return logger, NewLocal(logger)

}

func (t *MockHook) Fire(e *logrus.Entry) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Entries = append(t.Entries, *e)
	return nil
}

func (t *MockHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// LastEntry returns the last entry that was logged or nil.
func (t *MockHook) LastEntry() *logrus.Entry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	i := len(t.Entries) - 1
	if i < 0 {
		return nil
	}
	return &t.Entries[i]
}

// AllEntries returns all entries that were logged.
func (t *MockHook) AllEntries() []*logrus.Entry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	// Make a copy so the returned value won't race with future log requests
	entries := make([]*logrus.Entry, len(t.Entries))
	for i := 0; i < len(t.Entries); i++ {
		// Make a copy, for safety
		entries[i] = &t.Entries[i]
	}
	return entries
}

// Reset removes all Entries from this test hook.
func (t *MockHook) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Entries = make([]logrus.Entry, 0)
}
