package metrics

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tbaRequestCount    metric.Int64Counter
	tbaRequestDuration metric.Float64Histogram
	tbaCacheHits       metric.Int64Counter
)

func InitTBAMetrics() {
	meter := otel.Meter("fantasy-frc-web")
	var err error
	tbaRequestCount, err = meter.Int64Counter(
		"tba.request.count",
		metric.WithDescription("Total TBA API requests"),
	)
	if err != nil {
		panic("failed to create tba.request.count: " + err.Error())
	}

	tbaRequestDuration, err = meter.Float64Histogram(
		"tba.request.duration",
		metric.WithDescription("TBA request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic("failed to create tba.request.duration: " + err.Error())
	}

	tbaCacheHits, err = meter.Int64Counter(
		"tba.cache.hit.count",
		metric.WithDescription("TBA cache hits/misses"),
	)
	if err != nil {
		panic("failed to create tba.cache.hit.count: " + err.Error())
	}
}

func RecordTbaRequest(endpoint string, status int, duration time.Duration) {
	if tbaRequestCount == nil || tbaRequestDuration == nil {
		return
	}
	if endpoint == "" {
		endpoint = "unknown"
	}

	tbaRequestCount.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("tba.endpoint", endpoint),
			attribute.String("http.status_code", strconv.Itoa(status)),
		),
	)

	tbaRequestDuration.Record(context.Background(), duration.Seconds(),
		metric.WithAttributes(
			attribute.String("tba.endpoint", endpoint),
		),
	)
}

func RecordTbaCacheHit(result string) {
	if tbaCacheHits == nil {
		return
	}
	tbaCacheHits.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("cache.result", result),
		),
	)
}
