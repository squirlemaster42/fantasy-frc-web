package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	prometheuscollectors "github.com/prometheus/client_golang/prometheus/collectors"
)

func InitMetrics(database *sql.DB) {
	prometheus.MustRegister(httpRequestCount)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(authenticatedRequestCount)
	prometheus.MustRegister(tbaRequestCount)
	prometheus.MustRegister(tbaRequestDuration)
	prometheus.MustRegister(tbaCacheHits)
	prometheus.MustRegister(prometheuscollectors.NewDBStatsCollector(database, "postgres"))

	InitDBQueryStats(database)
	InitActiveUserCollector()
	InitWebSocketMetrics()
}
