package picking

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     pickingDefaults
)

type pickingDefaults struct {
	QueueBuffer int           `env:"PICK_NOTIFIER_QUEUE_BUFFER" default:"10"`
	SendTimeout time.Duration `env:"PICK_NOTIFIER_SEND_TIMEOUT" default:"5s"`
}

func loadDefaults() pickingDefaults {
	var d pickingDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *pickingDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// PickNotifierQueueBuffer returns the per-watcher notification channel capacity.
func PickNotifierQueueBuffer() int { return getDefaults().QueueBuffer }

// PickNotifierSendTimeout returns how long the notifier waits to send to a watcher.
func PickNotifierSendTimeout() time.Duration { return getDefaults().SendTimeout }
