package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	activeUserGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of unique authenticated users active within the given time window",
		},
		[]string{"window"},
	)

	activeUsers   = make(map[string]time.Time)
	activeUsersMu sync.RWMutex
)

func InitActiveUserCollector() {
	prometheus.MustRegister(activeUserGauge)
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
		var count1m, count5m, count15m int

		for uuid, lastSeen := range activeUsers {
			age := now.Sub(lastSeen)
			if age > 15*time.Minute {
				continue
			}
			pruned[uuid] = lastSeen
			if age <= 1*time.Minute {
				count1m++
			}
			if age <= 5*time.Minute {
				count5m++
			}
			count15m++
		}
		activeUsers = pruned
		activeUsersMu.Unlock()

		activeUserGauge.WithLabelValues("1m").Set(float64(count1m))
		activeUserGauge.WithLabelValues("5m").Set(float64(count5m))
		activeUserGauge.WithLabelValues("15m").Set(float64(count15m))
	}
}
