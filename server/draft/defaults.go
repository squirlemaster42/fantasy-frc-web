package draft

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     draftDefaults
)

type draftDefaults struct {
	InboxBuffer    int           `env:"DRAFT_ACTOR_INBOX_BUFFER" default:"100"`
	RequestTimeout time.Duration `env:"DRAFT_ACTOR_REQUEST_TIMEOUT" default:"5s"`
}

func loadDefaults() draftDefaults {
	var d draftDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *draftDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// DraftActorInboxBuffer returns the capacity of each draft actor's inbox channel.
func DraftActorInboxBuffer() int { return getDefaults().InboxBuffer }

// DraftActorRequestTimeout returns how long callers wait when posting to or
// receiving a reply from a draft actor.
func DraftActorRequestTimeout() time.Duration { return getDefaults().RequestTimeout }
