package ctx

import (
	"bytes"
	"context"
	"net/http"

	"go.opencensus.io/trace"
)

func (s *ctxSuite) TestStartSpan() {
	bg := Background()
	ctx, span := StartSpan(bg, "spanName")
	defer span.End()
	ctx.Info("TestStartSpan")
	foo(ctx, "TestStartSpan")
}

func (s *ctxSuite) TestStartSpanWithRemoteParent() {
	bg := Background()
	cont := context.Background()
	_, remoteSpan := trace.StartSpan(cont, "remoteSpan")
	defer remoteSpan.End()
	sc := remoteSpan.SpanContext()
	ctx, span := StartSpanWithRemoteParent(bg, "newSpan", sc)
	defer span.End()
	ctx.Info("TestStartSpan")
	foo(ctx, "TestStartSpan")
	localSpan := trace.FromContext(ctx)
	s.Equal(sc.TraceID, localSpan.SpanContext().TraceID)
}

func (s *ctxSuite) TestSpanContextToString() {
	cont := context.Background()
	_, span := trace.StartSpan(cont, "span")
	defer span.End()
	sc := span.SpanContext()
	headerValue := SpanContextToString(sc)

	newSc, ok := StringToSpanContext(headerValue)
	s.Equal(true, ok)
	s.Equal(sc.TraceID, newSc.TraceID)
}

func (s *ctxSuite) TestSpanContextToRequest() {
	bg := Background()
	format := HTTPFormat{}
	cont, span := StartSpan(bg, "span")
	defer span.End()
	sc := span.SpanContext()
	req, err := http.NewRequest("GET", "http://test.url", bytes.NewBuffer(nil))
	s.NoError(err)
	AddTraceToRequest(cont, req)

	newSc, ok := format.SpanContextFromRequest(req)
	s.Equal(true, ok)
	s.Equal(sc.TraceID, newSc.TraceID)
}
