package metrics

import (
	"sync"
	"time"

	"server/config"
)

const (
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
	ActiveUserTickInterval time.Duration `env:"METRICS_ACTIVE_USER_TICK_INTERVAL" default:"10s"`
	QueryIdMaxLength       int           `env:"METRICS_QUERY_ID_MAX_LENGTH" default:"100"`
	QueryThresholdMs       int           `env:"DB_QUERY_THRESHOLD_MS" default:"50"`
	QueryPollInterval      time.Duration `env:"DB_QUERY_POLL_INTERVAL" default:"30s"`
	QueryMaxCount          int           `env:"DB_QUERY_MAX_COUNT" default:"50"`
}

func loadDefaults() metricsDefaults {
	var d metricsDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *metricsDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// ActiveUserCollectorTickInterval returns how often active-user gauges are updated.
func ActiveUserCollectorTickInterval() time.Duration { return getDefaults().ActiveUserTickInterval }

// MetricsQueryIdMaxLength returns the maximum length of a query ID label before truncation.
func MetricsQueryIdMaxLength() int { return getDefaults().QueryIdMaxLength }

// DBQueryThresholdMs returns the minimum mean query time for stats collection.
func DBQueryThresholdMs() int { return getDefaults().QueryThresholdMs }

// DBQueryPollInterval returns how often database query stats are collected.
func DBQueryPollInterval() time.Duration { return getDefaults().QueryPollInterval }

// DBQueryMaxCount returns the maximum number of queries returned by the stats collector.
func DBQueryMaxCount() int { return getDefaults().QueryMaxCount }
