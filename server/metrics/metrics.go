package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"server/log"
)

var dbStatsReg metric.Registration

func InitMetrics(ctx context.Context, database *sql.DB) error {
	var err error
	dbStatsReg, err = otelsql.RegisterDBStatsMetrics(database, otelsql.WithAttributes(semconv.DBSystemPostgreSQL))
	if err != nil {
		log.Warn(ctx, "Failed to register OTel DB stats metrics", "error", err)
		return fmt.Errorf("failed to register OTel DB stats metrics: %w", err)
	}

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(15 * time.Second)); err != nil {
		log.Warn(ctx, "Failed to start runtime metrics", "error", err)
	}

	InitHTTPMetrics()
	InitTBAMetrics()
	InitWebSocketMetrics()
	InitActiveUserCollector(ctx)
	InitDBQueryStats(ctx, database)
	return nil
}

func ShutdownMetrics() {
	if dbStatsReg != nil {
		_ = dbStatsReg.Unregister()
	}
}
