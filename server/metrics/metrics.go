package metrics

import "github.com/prometheus/client_golang/prometheus"

func InitAllMetrics() {
	prometheus.MustRegister(httpRequestCount)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(tbaRequestCount)
	prometheus.MustRegister(tbaRequestDuration)
	prometheus.MustRegister(tbaCacheHits)
}
