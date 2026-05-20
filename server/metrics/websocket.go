package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	websocketListenerGauge metric.Int64UpDownCounter
)

func InitWebSocketMetrics() {
	meter := otel.Meter("fantasy-frc-web")
	var err error
	websocketListenerGauge, err = meter.Int64UpDownCounter(
		"websocket.listeners.active",
		metric.WithDescription("Number of active websocket listeners for draft pick events"),
	)
	if err != nil {
		panic("failed to create websocket.listeners.active: " + err.Error())
	}
}

func IncrementWebSocketListener() {
	if websocketListenerGauge == nil {
		return
	}
	websocketListenerGauge.Add(context.Background(), 1)
}

func DecrementWebSocketListener() {
	if websocketListenerGauge == nil {
		return
	}
	websocketListenerGauge.Add(context.Background(), -1)
}
