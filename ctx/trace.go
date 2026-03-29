package ctx

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
)

const (
	// traceHeaderMaxSize is copied and rename from `contrib.go.opencensus.io/exporter/stackdriver/propagation`
	traceHeaderMaxSize = 200
	// TraceHeader is request header for tracing, export so we can use it without http.Request
	TraceHeader = `X-Cloud-Trace-Context`
)

var _ propagation.HTTPFormat = (*HTTPFormat)(nil)

// StartSpan returns a copy of parent with trace span append
func StartSpan(parent CTX, name string, o ...trace.StartOption) (CTX, *trace.Span) {
	newCont, span := trace.StartSpan(parent, name, o...)
	spanContext := span.SpanContext()
	return CTX{
		Context:     newCont,
		FieldLogger: parent.FieldLogger.WithField("traceID", spanContext.TraceID.String()),
	}, span
}

// StartSpanWithRemoteParent returns a copy of parent ctx with parent trace span append
func StartSpanWithRemoteParent(parent CTX, name string, parentSpanContext trace.SpanContext, o ...trace.StartOption) (CTX, *trace.Span) {
	newCont, span := trace.StartSpanWithRemoteParent(parent, name, parentSpanContext, o...)
	return CTX{
		Context:     newCont,
		FieldLogger: parent.FieldLogger.WithField("traceID", parentSpanContext.TraceID.String()),
	}, span
}

// HTTPFormat implements propagation.HTTPFormat to propagate
// traces in HTTP headers for Google Cloud Platform and Stackdriver Trace.
// copied from "contrib.go.opencensus.io/exporter/stackdriver/propagation" and modified.
type HTTPFormat struct{}

// SpanContextFromRequest extracts a Stackdriver Trace span context from incoming requests.
func (f *HTTPFormat) SpanContextFromRequest(req *http.Request) (sc trace.SpanContext, ok bool) {
	h := req.Header.Get(TraceHeader)
	// See https://cloud.google.com/trace/docs/faq for the header HTTPFormat.
	// Return if the header is empty or missing, or if the header is unreasonably
	// large, to avoid making unnecessary copies of a large string.
	if h == "" || len(h) > traceHeaderMaxSize {
		return trace.SpanContext{}, false
	}
	return StringToSpanContext(h)
}

// SpanContextToRequest modifies the given request to include a Stackdriver Trace header.
func (f *HTTPFormat) SpanContextToRequest(sc trace.SpanContext, req *http.Request) {
	header := SpanContextToString(sc)
	req.Header.Set(TraceHeader, header)
}

// SpanContextToString convert SpanContext to `X-Cloud-Trace-Context`
// header propagation format used by Google Cloud products.
func SpanContextToString(sc trace.SpanContext) string {
	sid := binary.BigEndian.Uint64(sc.SpanID[:])
	return fmt.Sprintf("%s/%d;o=%d", hex.EncodeToString(sc.TraceID[:]), sid, int64(sc.TraceOptions))
}

// AddTraceToRequest add `X-Cloud-Trace-Context` header to request
func AddTraceToRequest(context CTX, req *http.Request) {
	span := trace.FromContext(context.Context)
	if span != nil {
		sc := span.SpanContext()
		httpFormat := HTTPFormat{}
		httpFormat.SpanContextToRequest(sc, req)
	}
}

// StringToSpanContext convert `X-Cloud-Trace-Context` header value back to SpanContext.
func StringToSpanContext(h string) (sc trace.SpanContext, ok bool) {
	// Parse the trace id field.
	slash := strings.Index(h, `/`)
	if slash == -1 {
		return trace.SpanContext{}, false
	}
	tid, h := h[:slash], h[slash+1:]

	buf, err := hex.DecodeString(tid)
	if err != nil {
		return trace.SpanContext{}, false
	}
	copy(sc.TraceID[:], buf)

	// Parse the span id field.
	spanstr := h
	semicolon := strings.Index(h, `;`)
	if semicolon != -1 {
		spanstr, h = h[:semicolon], h[semicolon+1:]
	}
	sid, err := strconv.ParseUint(spanstr, 10, 64)
	if err != nil {
		return trace.SpanContext{}, false
	}
	binary.BigEndian.PutUint64(sc.SpanID[:], sid)

	// Parse the options field, options field is optional.
	if !strings.HasPrefix(h, "o=") {
		return sc, true
	}
	o, err := strconv.ParseUint(h[2:], 10, 64)
	if err != nil {
		return trace.SpanContext{}, false
	}
	sc.TraceOptions = trace.TraceOptions(o)
	return sc, true
}
