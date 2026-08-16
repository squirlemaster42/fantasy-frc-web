package draft

import (
	"server/utils"
	"sync"
	"time"
)

const (
	draftActorInboxBufferEnvKey = "DRAFT_ACTOR_INBOX_BUFFER"
	defaultDraftActorInboxBuffer = 100

	draftActorRequestTimeoutEnvKey = "DRAFT_ACTOR_REQUEST_TIMEOUT"
	defaultDraftActorRequestTimeout = 5 * time.Second
)

var (
	defaultsOnce sync.Once
	defaults     draftDefaults
)

type draftDefaults struct {
	inboxBuffer    int
	requestTimeout time.Duration
}

func loadDefaults() draftDefaults {
	return draftDefaults{
		inboxBuffer:    utils.MustGetEnvInt(draftActorInboxBufferEnvKey, defaultDraftActorInboxBuffer),
		requestTimeout: utils.MustGetEnvDuration(draftActorRequestTimeoutEnvKey, defaultDraftActorRequestTimeout),
	}
}

func getDefaults() *draftDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// DraftActorInboxBuffer returns the capacity of each draft actor's inbox channel.
func DraftActorInboxBuffer() int { return getDefaults().inboxBuffer }

// DraftActorRequestTimeout returns how long callers wait when posting to or
// receiving a reply from a draft actor.
func DraftActorRequestTimeout() time.Duration { return getDefaults().requestTimeout }
