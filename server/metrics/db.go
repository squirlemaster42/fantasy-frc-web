package metrics

import (
	"context"
	"database/sql"
	"server/database"
	"server/log"
	"server/utils"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	dbQueryMeanTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_query_mean_time_seconds",
			Help: "Mean execution time of queries in seconds",
		},
		[]string{"query"},
	)
	dbQueryCalls = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_query_calls_total",
			Help: "Total number of times each query was called",
		},
		[]string{"query"},
	)
	dbQueryRows = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_query_rows_total",
			Help: "Total rows returned by queries",
		},
		[]string{"query"},
	)
)

var (
	queryThresholdMs = utils.GetEnvInt("DB_QUERY_THRESHOLD_MS", 50)
	pollInterval     = utils.GetEnvDuration("DB_QUERY_POLL_INTERVAL", 30*time.Second)
	maxQueries       = utils.GetEnvInt("DB_QUERY_MAX_COUNT", 50)
)

func InitDBQueryStats(ctx context.Context, db *sql.DB) {
	prometheus.MustRegister(dbQueryMeanTime, dbQueryCalls, dbQueryRows)

	log.Info(ctx, "Starting DB query stats collector", "threshold_ms", queryThresholdMs, "interval", pollInterval, "max_queries", maxQueries)

	go collectQueryStats(ctx, db)
}

func collectQueryStats(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for range ticker.C {
		collectQueryStatsIteration(ctx, db)
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
		log.Error(ctx, "Failed to query pg_stat_statements", "error", err)
		return
	}
	defer database.CloseRows(ctx, rows, "collectQueryStatsIteration")

	dbQueryMeanTime.Reset()
	dbQueryCalls.Reset()
	dbQueryRows.Reset()

	for rows.Next() {
		var queryText string
		var calls int
		var meanTime, totalTime float64
		var rowsCount int64

		err := rows.Scan(&queryText, &calls, &meanTime, &totalTime, &rowsCount)
		if err != nil {
			log.Error(ctx, "Failed to scan pg_stat_statements row", "error", err)
			continue
		}

		queryID := queryText
		if len(queryID) > 100 {
			queryID = queryID[:100] + "..."
		}

		dbQueryMeanTime.WithLabelValues(queryID).Set(meanTime / 1000.0)
		dbQueryCalls.WithLabelValues(queryID).Set(float64(calls))
		dbQueryRows.WithLabelValues(queryID).Set(float64(rowsCount))
	}
}
