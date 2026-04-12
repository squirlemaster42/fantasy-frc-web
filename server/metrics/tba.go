package metrics

import (
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	tbaRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tba_requests_total",
			Help: "Total TBA API requests",
		},
		[]string{"endpoint", "status"},
	)
	tbaRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tba_request_duration_seconds",
			Help:    "TBA request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint"},
	)
	tbaCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tba_cache_hits_total",
			Help: "TBA cache hits/misses",
		},
		[]string{"result"},
	)
	tbaEndpointPattern = regexp.MustCompile(`/api/v3/(.+)`)
)

func RecordTbaRequest(url string, status int, duration time.Duration) {
	endpoint := "unknown"
	if matches := tbaEndpointPattern.FindStringSubmatch(url); len(matches) > 1 {
		endpoint = matches[1]
	}

	tbaRequestCount.WithLabelValues(
		endpoint,
		strconv.Itoa(status),
	).Inc()

	tbaRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

func RecordTbaCacheHit(result string) {
	tbaCacheHits.WithLabelValues(result).Inc()
}
