package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	websocketListenerGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_listeners_active",
			Help: "Number of active websocket listeners for draft pick events",
		},
	)
)

func InitWebSocketMetrics() {
	prometheus.MustRegister(websocketListenerGauge)
}

func IncrementWebSocketListener() {
	websocketListenerGauge.Inc()
}

func DecrementWebSocketListener() {
	websocketListenerGauge.Dec()
}
