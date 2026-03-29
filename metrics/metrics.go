/*Package metrics wraps datadog-go to faciliate metric recording
Following are naming convention of metric:
- Internal process time: *.time
- External latency: *.latency
- Error: *.err
- Warning: *.warn
*/
package metrics

import (
	"strings"
)

const (
	// TagValueNA is used for tags whose values are not available.
	TagValueNA = "n/a"
)

func init() {
	StartMonitor()
}

// Ender provides interface for BumpHistogram
type Ender interface {
	End()
}

// Service provides interface for metrics
type Service interface {
	BumpAvg(key string, val float64, tags ...string)
	BumpSum(key string, val float64, tags ...string)
	BumpHistogram(key string, val float64, tags ...string)

	BumpTime(key string, tags ...string) Ender
}

// New creates a facebook/stats compatible metric client with package name as prefix
func New(pkgName string) Service {
	return &Metrics{pkgName: pkgName}
}

// Metrics wraps datadog-go to be facebookgo/stat.Client interface.
// See https://godoc.org/github.com/facebookgo/stats#Client for interface details.
type Metrics struct {
	pkgName string
	datadog DDMetrics
}

// bumpSumPanic handles panics for all metrics vendor.
// inconsistent tagging.
func (mt *Metrics) bumpSumPanic(key, tag string) {
	mt.datadog.BumpSum(key, 1, "tag", tag)
}

// BumpAvg bumps the average for the given key.
func (mt *Metrics) BumpAvg(key string, val float64, tags ...string) {
	defer func() {
		if err := recover(); err != nil {
			mt.bumpSumPanic("bumpavg.panic", mt.pkgName+`.`+key+"#"+strings.Join(tags, "#"))
		}
	}()

	// push data to datadog.
	mt.datadog.BumpAvg(mt.pkgName+`.`+key, val, tags...)
}

// BumpSum bumps the sum for the given key.
func (mt *Metrics) BumpSum(key string, val float64, tags ...string) {
	defer func() {
		if err := recover(); err != nil {
			mt.bumpSumPanic("bumpsum.panic", mt.pkgName+`.`+key+"#"+strings.Join(tags, "#"))
		}
	}()

	// push data to datadog.
	mt.datadog.BumpSum(mt.pkgName+`.`+key, val, tags...)
}

// BumpHistogram bumps the histogram for the given key.
func (mt *Metrics) BumpHistogram(key string, val float64, tags ...string) {
	defer func() {
		if err := recover(); err != nil {
			mt.bumpSumPanic("bumphistogram.panic", mt.pkgName+`.`+key+"#"+strings.Join(tags, "#"))
		}
	}()

	// push data to datadog.
	mt.datadog.BumpHistogram(mt.pkgName+`.`+key, val, tags...)
}

// BumpTime is a special version of BumpHistogram which is specialized for
// timers. Calling it starts the timer, and it returns a value on which End()
// can be called to indicate finishing the timer. A convenient way of
// recording the duration of a function is calling it like such at the top of
// the function:
//
//     defer s.BumpTime("my.function").End()
func (mt *Metrics) BumpTime(key string, tags ...string) Ender {
	// push data to datadog.
	ddEnd := mt.datadog.BumpTime(mt.pkgName+`.`+key, tags...)

	return &timeTracker{
		ddEnd: ddEnd,
		panicHandler: func() {
			mt.bumpSumPanic("bumptime.panic", mt.pkgName+`.`+key+"#"+strings.Join(tags, "#"))
		},
	}
}

type timeTracker struct {
	ddEnd interface {
		End()
	}
	panicHandler func()
}

func (t *timeTracker) End() {
	defer func() {
		if err := recover(); err != nil {
			t.panicHandler()
		}
	}()

	// end datadog counter.
	t.ddEnd.End()
	return
}
