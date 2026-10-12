package metrics

import (
	"server/utils"
	"sync"
	"time"
)

const (
	activeUserCollectorTickIntervalEnvKey = "METRICS_ACTIVE_USER_TICK_INTERVAL"
	defaultActiveUserCollectorTickInterval = 10 * time.Second

	metricsQueryIdMaxLengthEnvKey = "METRICS_QUERY_ID_MAX_LENGTH"
	defaultMetricsQueryIdMaxLength = 100

	// Active-user window sizes.
	activeUserWindow1m  = 1 * time.Minute
	activeUserWindow5m  = 5 * time.Minute
	activeUserWindow15m = 15 * time.Minute

	// Fallback label for unknown routes/endpoints.
	unknownRouteLabel = "unknown"
)

var (
	// Histogram buckets are not constants in Go, but centralizing them here
	// makes them easy to find and change consistently.
	httpRequestDurationBuckets = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5}
	tbaRequestDurationBuckets  = []float64{.1, .25, .5, 1, 2.5, 5, 10}
)

var (
	defaultsOnce sync.Once
	defaults     metricsDefaults
)

type metricsDefaults struct {
	activeUserTickInterval time.Duration
	queryIdMaxLength       int
}

func loadDefaults() metricsDefaults {
	return metricsDefaults{
		activeUserTickInterval: utils.MustGetEnvDuration(activeUserCollectorTickIntervalEnvKey, defaultActiveUserCollectorTickInterval),
		queryIdMaxLength:       utils.MustGetEnvInt(metricsQueryIdMaxLengthEnvKey, defaultMetricsQueryIdMaxLength),
	}
}

func getDefaults() *metricsDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// ActiveUserCollectorTickInterval returns how often active-user gauges are updated.
func ActiveUserCollectorTickInterval() time.Duration { return getDefaults().activeUserTickInterval }

// MetricsQueryIdMaxLength returns the maximum length of a query ID label before truncation.
func MetricsQueryIdMaxLength() int { return getDefaults().queryIdMaxLength }
