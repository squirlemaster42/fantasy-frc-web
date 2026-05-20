package metrics

import (
	"context"
	"sync"
	"time"

	"server/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	activeUserGauge metric.Float64ObservableGauge
	activeUsers     = make(map[string]time.Time)
	activeUsersMu   sync.RWMutex
)

func InitActiveUserCollector(ctx context.Context) {
	meter := otel.Meter("fantasy-frc-web")
	var err error
	activeUserGauge, err = meter.Float64ObservableGauge(
		"active.users",
		metric.WithDescription("Number of unique authenticated users active within the given time window"),
	)
	if err != nil {
		panic("failed to create active.users: " + err.Error())
	}

	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			activeUsersMu.RLock()
			now := time.Now()
			var count1m, count5m, count15m int

			for _, lastSeen := range activeUsers {
				age := now.Sub(lastSeen)
				if age <= 1*time.Minute {
					count1m++
				}
				if age <= 5*time.Minute {
					count5m++
				}
				if age <= 15*time.Minute {
					count15m++
				}
			}
			activeUsersMu.RUnlock()

			o.ObserveFloat64(activeUserGauge, float64(count1m), metric.WithAttributes(attribute.String("window", "1m")))
			o.ObserveFloat64(activeUserGauge, float64(count5m), metric.WithAttributes(attribute.String("window", "5m")))
			o.ObserveFloat64(activeUserGauge, float64(count15m), metric.WithAttributes(attribute.String("window", "15m")))
			return nil
		},
		activeUserGauge,
	)
	if err != nil {
		log.Warn(ctx, "Failed to register active users callback", "error", err)
	}

	go collectActiveUsers()
}

func RecordUserActivity(userUuid string) {
	activeUsersMu.Lock()
	activeUsers[userUuid] = time.Now()
	activeUsersMu.Unlock()
}

func collectActiveUsers() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		activeUsersMu.Lock()
		pruned := make(map[string]time.Time)
		for uuid, lastSeen := range activeUsers {
			if now.Sub(lastSeen) > 15*time.Minute {
				continue
			}
			pruned[uuid] = lastSeen
		}
		activeUsers = pruned
		activeUsersMu.Unlock()
	}
}
