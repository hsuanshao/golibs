package entity

import "time"

// HealthStatus describe service connection health status
type HealthStatus struct {
	Cloud   CloudServiceProvider
	Latency time.Duration
}
