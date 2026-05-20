package metrics

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"sync"
	"time"

	"server/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	dbQueryMeanTime metric.Float64ObservableGauge
	dbQueryCalls    metric.Int64ObservableGauge
	dbQueryRows     metric.Int64ObservableGauge

	queryThresholdMs = getEnvAsInt("DB_QUERY_THRESHOLD_MS", 50)
	pollInterval     = getEnvAsDuration("DB_QUERY_POLL_INTERVAL", 30*time.Second)
	maxQueries       = getEnvAsInt("DB_QUERY_MAX_COUNT", 50)

	dbStatsMu        sync.RWMutex
	latestQueryStats []queryStat
)

type queryStat struct {
	query string
	calls int
	mean  float64
	rows  int64
}

func getEnvAsInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		log.Warn(context.Background(), "Invalid env var, using default", "key", key, "value", val, "error", err)
		return defaultVal
	}
	return intVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		log.Warn(context.Background(), "Invalid env var, using default", "key", key, "value", val, "error", err)
		return defaultVal
	}
	return d
}

func InitDBQueryStats(ctx context.Context, db *sql.DB) {
	meter := otel.Meter("fantasy-frc-web")
	var err error
	dbQueryMeanTime, err = meter.Float64ObservableGauge(
		"db.query.mean.duration",
		metric.WithDescription("Mean execution time of queries in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic("failed to create db.query.mean.duration: " + err.Error())
	}

	dbQueryCalls, err = meter.Int64ObservableGauge(
		"db.query.calls",
		metric.WithDescription("Total number of times each query was called"),
	)
	if err != nil {
		panic("failed to create db.query.calls: " + err.Error())
	}

	dbQueryRows, err = meter.Int64ObservableGauge(
		"db.query.rows",
		metric.WithDescription("Total rows returned by queries"),
	)
	if err != nil {
		panic("failed to create db.query.rows: " + err.Error())
	}

	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			dbStatsMu.RLock()
			stats := latestQueryStats
			dbStatsMu.RUnlock()

			for _, s := range stats {
				queryID := s.query
				if len(queryID) > 100 {
					queryID = queryID[:100] + "..."
				}

				o.ObserveFloat64(dbQueryMeanTime, s.mean/1000.0,
					metric.WithAttributes(attribute.String("query", queryID)))
				o.ObserveInt64(dbQueryCalls, int64(s.calls),
					metric.WithAttributes(attribute.String("query", queryID)))
				o.ObserveInt64(dbQueryRows, s.rows,
					metric.WithAttributes(attribute.String("query", queryID)))
			}
			return nil
		},
		dbQueryMeanTime, dbQueryCalls, dbQueryRows,
	)
	if err != nil {
		log.Warn(ctx, "Failed to register DB query stats callback", "error", err)
	}

	log.Info(ctx, "Starting DB query stats collector", "threshold_ms", queryThresholdMs, "interval", pollInterval, "max_queries", maxQueries)

	go collectQueryStats(ctx, db)
}

func collectQueryStats(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "DB query stats collector shutting down")
			return
		case <-ticker.C:
			collectQueryStatsIteration(ctx, db)
		}
	}
}

func collectQueryStatsIteration(ctx context.Context, db *sql.DB) {
	query := `
		SELECT
			query,
			calls,
			mean_exec_time::numeric,
			total_exec_time::numeric,
			rows
		FROM pg_stat_statements
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, float64(queryThresholdMs)/1000.0, maxQueries)
	if err != nil {
		log.Warn(ctx, "Failed to query pg_stat_statements", "error", err)
		return
	}
	defer rows.Close()

	var stats []queryStat

	for rows.Next() {
		var queryText string
		var calls int
		var meanTime, totalTime float64
		var rowsCount int64

		err := rows.Scan(&queryText, &calls, &meanTime, &totalTime, &rowsCount)
		if err != nil {
			log.Warn(ctx, "Failed to scan pg_stat_statements row", "error", err)
			continue
		}

		queryID := queryText
		if len(queryID) > 100 {
			queryID = queryID[:100] + "..."
		}

		stats = append(stats, queryStat{
			query: queryID,
			calls: calls,
			mean:  meanTime,
			rows:  rowsCount,
		})
	}

	dbStatsMu.Lock()
	latestQueryStats = stats
	dbStatsMu.Unlock()
}
