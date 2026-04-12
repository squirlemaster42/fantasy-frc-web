package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// TBA metrics for monitoring The Blue Alliance API requests.
var (
	// tbaRequestCount tracks total TBA API requests by endpoint and status code.
	tbaRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tba_requests_total",
			Help: "Total TBA API requests",
		},
		[]string{"endpoint", "status"},
	)
	// tbaRequestDuration tracks TBA API request duration by endpoint.
	tbaRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tba_request_duration_seconds",
			Help:    "TBA request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint"},
	)
	// tbaCacheHits tracks TBA cache hits and misses.
	tbaCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tba_cache_hits_total",
			Help: "TBA cache hits/misses",
		},
		[]string{"result"},
	)
)

// RecordTbaRequest records a TBA API request's endpoint, status code, and duration.
func RecordTbaRequest(endpoint string, status int, duration time.Duration) {
	if endpoint == "" {
		endpoint = "unknown"
	}

	tbaRequestCount.WithLabelValues(
		endpoint,
		strconv.Itoa(status),
	).Inc()

	tbaRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// RecordTbaCacheHit records a TBA cache hit or miss.
// result should be "hit", "miss", or "not_modified".
func RecordTbaCacheHit(result string) {
	tbaCacheHits.WithLabelValues(result).Inc()
}
