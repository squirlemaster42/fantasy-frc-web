package picking

import (
	"server/utils"
	"sync"
	"time"
)

const (
	pickNotifierQueueBufferEnvKey = "PICK_NOTIFIER_QUEUE_BUFFER"
	defaultPickNotifierQueueBuffer = 10

	pickNotifierSendTimeoutEnvKey = "PICK_NOTIFIER_SEND_TIMEOUT"
	defaultPickNotifierSendTimeout = 5 * time.Second
)

var (
	defaultsOnce sync.Once
	defaults     pickingDefaults
)

type pickingDefaults struct {
	queueBuffer int
	sendTimeout time.Duration
}

func loadDefaults() pickingDefaults {
	return pickingDefaults{
		queueBuffer: utils.MustGetEnvInt(pickNotifierQueueBufferEnvKey, defaultPickNotifierQueueBuffer),
		sendTimeout: utils.MustGetEnvDuration(pickNotifierSendTimeoutEnvKey, defaultPickNotifierSendTimeout),
	}
}

func getDefaults() *pickingDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// PickNotifierQueueBuffer returns the per-watcher notification channel capacity.
func PickNotifierQueueBuffer() int { return getDefaults().queueBuffer }

// PickNotifierSendTimeout returns how long the notifier waits to send to a watcher.
func PickNotifierSendTimeout() time.Duration { return getDefaults().sendTimeout }
