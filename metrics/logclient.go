package metrics

import (
	log "github.com/sirupsen/logrus"
)

// LogClient export default logclient interface.
type LogClient struct{}

// Gauge measure the value of a particular thing at a particular time,
// like the amount of fuel in a car’s gas tank or the number of users connected to a system
func (lc *LogClient) Gauge(name string, value float64, tags []string, rate float64) error {
	log.WithFields(log.Fields{"key": name, "val": value, "tags": tags}).Debug("metric gauge")
	return nil
}

// Count tracks how many times something happened per second,
// like the number of database requests or page views.
func (lc *LogClient) Count(name string, value int64, tags []string, rate float64) error {
	log.WithFields(log.Fields{"key": name, "val": value, "tags": tags}).Debug("metric count")
	return nil
}

// Histogram tracks the statistical distribution of a set of values,
// like the duration of a number of database queries or the size of files uploaded by users.
// Each histogram will track the average, the minimum, the maximum, the median, the 95th percentile and the count.
func (lc *LogClient) Histogram(name string, value float64, tags []string, rate float64) error {
	log.WithFields(log.Fields{"key": name, "val": value, "tags": tags}).Debug("metric histogram")
	return nil
}

// TimeInMilliseconds is essentially a special case of histograms,
// so it is treated in the same manner by DogStatsD for backwards compatibility.
func (lc *LogClient) TimeInMilliseconds(name string, value float64, tags []string, rate float64) error {
	log.WithFields(log.Fields{"key": name, "time_ms": value, "tags": tags}).Debug("metric time")
	return nil
}
