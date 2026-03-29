package ctx

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type ctxSuite struct {
	suite.Suite
}

func foo(ctx CTX, msg string) {
	ctx.WithField("msg", msg).Info("foo")
}

func (s *ctxSuite) SetupSuite() {
}

func TestCtxSuite(t *testing.T) {
	suite.Run(t, new(ctxSuite))
}

func (s *ctxSuite) TestBackground() {
	ctx := Background()
	foo(ctx, "TestBackground")
}

func (s *ctxSuite) TestWithValue() {
	bg := Background()
	ctx := WithValue(bg, "key", "value")
	ctx.Info("TestBasic")
	foo(ctx, "TestWithValue")
	s.Equal("value", ctx.Context.Value("key"))
}

func (s *ctxSuite) TestWithValues() {
	bg := Background()
	ctx := WithValues(bg, map[string]interface{}{"key": "value", "key2": "value2"})
	ctx.Info("TestWithValues")
	foo(ctx, "TestWithValues")

	s.Equal("value", ctx.Context.Value("key"))
	s.Equal("value2", ctx.Context.Value("key2"))
}

func (s *ctxSuite) TestWithCancel() {
	bg := Background()
	ctx, cancel := WithCancel(bg)
	defer cancel()
	testFunc := func(ctx context.Context) bool {
		for {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(100 * time.Millisecond):
				return true
			}
		}
	}
	ctx.Info("TestWithCancel")
	foo(ctx, "TestWithCancel")
	var result bool
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()
	result = testFunc(ctx)
	s.Equal(false, result)
}

func (s *ctxSuite) TestWithTimeout() {
	bg := Background()
	ctx, cancel := WithTimeout(bg, 10*time.Millisecond)
	defer cancel()
	testFunc := func(ctx context.Context) int {
		for {
			select {
			case <-ctx.Done():
				return 1
			case <-time.After(100 * time.Millisecond):
				return 2
			}
		}
	}
	ctx.Info("TestWithTimeout")
	foo(ctx, "TestWithTimeout")
	result := testFunc(ctx)
	s.Equal(1, result)
	s.EqualError(ctx.Err(), "context deadline exceeded")
}

func (s *ctxSuite) TestSetLogger() {
	bg := Background()
	ctx := bg.SetLogger(&HookConfig{LogFormat: &logrus.TextFormatter{DisableColors: false}, SetAsStdOutput: true})
	ctx.Info("TestSetLogger")
	foo(ctx, "TestSetLogger")
}
