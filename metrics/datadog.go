package metrics

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	log "github.com/sirupsen/logrus"
)

// During grace period, "base/metrics" and "service/metrics" will co-exists.
// As flag doesn't allow double register, the flags used by "base/metrics" need to be shared to "service/metrics"

var (
	initOnce = sync.Once{}

	// MetricDest is the public form of MetricDest
	MetricDest = flag.String("metric_destination", "log", "metric destination (datadog | log)")
	// DdHost is the public form of DdHost
	DdHost = flag.String("datadog_host", "", "datadog agent host name")
	// DdPort is the public form of DdPort
	DdPort = flag.String("datadog_port", "", "datadog agent port")

	ddClient = statsCli(nil)
)

const (
	// ddRate is the rate to pass metrics to datadog agent. 1 means always
	ddRate = 1
	// buffer 10 counters before sending to statsd
	bufferMetrics = 10
)

func initDDClient() {
	// In case we don't want to send to datadog, just log
	if *MetricDest == "log" {
		ddClient = &LogClient{}
		return
	}

	// Init datadog client once so the buffer is counted together. Also it's better to
	// maintain one connection toward statsd agent
	addr := fmt.Sprintf("%s:%s", *DdHost, *DdPort)
	log.WithField("addr", addr).Info("connecting to datadog agent")

	var err error
	ddClient, err = statsd.NewBuffered(addr, bufferMetrics)
	if err != nil {
		log.WithFields(log.Fields{"addr": addr, "err": err}).Panic(
			"can't talk to datadog agent")
	}
}

type statsCli interface {
	Gauge(name string, value float64, tags []string, rate float64) error
	Count(name string, value int64, tags []string, rate float64) error
	Histogram(name string, value float64, tags []string, rate float64) error
	TimeInMilliseconds(name string, value float64, tags []string, rate float64) error
}

// DDMetrics wraps datadog statsd metrics implement facebookgo/stats.Client interface.
// See https://godoc.org/github.com/facebookgo/stats#Client for interface details.
type DDMetrics struct{}

// BumpAvg bumps the average for the given key.
func (dm *DDMetrics) BumpAvg(key string, val float64, tags ...string) {
	initOnce.Do(initDDClient)
	// datadog doesn't have a function to compute average only. Work-around by calculating
	// histogram (which is overkill however)
	if err := ddClient.Gauge(key, val, parseTag(tags), ddRate); err != nil {
		log.WithFields(log.Fields{"key": key, "val": val}).Error("BumpAvg fail")
	}
}

// BumpSum bumps the sum for the given key.
func (dm *DDMetrics) BumpSum(key string, val float64, tags ...string) {
	initOnce.Do(initDDClient)
	if err := ddClient.Count(key, int64(val), parseTag(tags), ddRate); err != nil {
		log.WithFields(log.Fields{"key": key, "val": val}).Error("BumpSum fail")
	}
}

// BumpHistogram bumps the histogram for the given key.
func (dm *DDMetrics) BumpHistogram(key string, val float64, tags ...string) {
	initOnce.Do(initDDClient)
	if err := ddClient.Histogram(key, val, parseTag(tags), ddRate); err != nil {
		log.WithFields(log.Fields{"key": key, "val": val}).Error("BumpHistogram fail")
	}
}

// BumpTime is a special version of BumpHistogram which is specialized for
// timers. Calling it starts the timer, and it returns a value on which End()
// can be called to indicate finishing the timer. A convenient way of
// recording the duration of a function is calling it like such at the top of
// the function:
//
//     defer s.BumpTime("my.function").End()
func (dm *DDMetrics) BumpTime(key string, tags ...string) interface {
	End()
} {
	initOnce.Do(initDDClient)
	return &ddTimeTracker{start: time.Now(), key: key, tags: parseTag(tags)}
}

func parseTag(tags []string) []string {
	if tags == nil {
		return nil
	}
	if len(tags)%2 != 0 {
		log.WithField("tags", tags).Panic("tag length needs to be multiple of 2")
	}
	arr := make([]string, len(tags)/2)
	for i := 0; i < len(tags); i += 2 {
		arr[i/2] = tags[i] + ":" + tags[i+1]
	}
	return arr
}

type ddTimeTracker struct {
	start time.Time
	key   string
	tags  []string
}

func (dt *ddTimeTracker) End() {
	d := time.Since(dt.start)
	msec := d / time.Millisecond
	nsec := d % time.Millisecond

	dur := float64(msec) + float64(nsec)*1e-6

	if err := ddClient.TimeInMilliseconds(dt.key, dur, dt.tags, ddRate); err != nil {
		log.WithFields(log.Fields{"key": dt.key, "val": dur}).Error("BumpTime fail")
	}
}
