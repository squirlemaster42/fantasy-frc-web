package metrics

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/prometheus/client_golang/prometheus"
	prometheuscollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"server/log"
)

var dbStatsReg metric.Registration

func InitMetrics(database *sql.DB) error {
	prometheus.MustRegister(httpRequestCount)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(authenticatedRequestCount)
	prometheus.MustRegister(tbaRequestCount)
	prometheus.MustRegister(tbaRequestDuration)
	prometheus.MustRegister(tbaCacheHits)
	prometheus.MustRegister(prometheuscollectors.NewDBStatsCollector(database, "postgres"))

	var err error
	dbStatsReg, err = otelsql.RegisterDBStatsMetrics(database, otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		log.Warn(context.Background(), "Failed to register OTel DB stats metrics", "error", err)
		return fmt.Errorf("failed to register OTel DB stats metrics: %w", err)
	}

	InitDBQueryStats(database)
	InitActiveUserCollector()
	InitWebSocketMetrics()
	return nil
}

func ShutdownMetrics() {
	if dbStatsReg != nil {
		_ = dbStatsReg.Unregister()
	}
}
