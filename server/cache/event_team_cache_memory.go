package cache

import (
	"context"
	"sync"
	"time"

	"server/log"
	"server/tbaHandler"
	"server/utils"
)

// memoryEventTeamCache is an in-memory implementation of EventTeamCache.
type memoryEventTeamCache struct {
	mu          sync.RWMutex
	teams       map[string]struct{}
	lastRefresh time.Time
	handler     tbaHandler.TBAInterface
	events      []string
}

func newMemoryEventTeamCache(handler tbaHandler.TBAInterface) *memoryEventTeamCache {
	return &memoryEventTeamCache{
		handler: handler,
		events:  utils.Events(),
		teams:   make(map[string]struct{}),
	}
}

func (m *memoryEventTeamCache) Refresh(ctx context.Context) error {
	teams, err := fetchEventTeams(ctx, m.handler, m.events)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.teams = teams
	m.lastRefresh = time.Now().UTC()
	m.mu.Unlock()

	log.Info(ctx, "Refreshed in-memory pick validation cache", "events", len(m.events), "teams", len(teams))
	return nil
}

func (m *memoryEventTeamCache) InvalidateAndRefresh(ctx context.Context) error {
	m.mu.Lock()
	m.teams = make(map[string]struct{})
	m.lastRefresh = time.Time{}
	m.mu.Unlock()

	return m.Refresh(ctx)
}

func (m *memoryEventTeamCache) IsTeamAtEvent(ctx context.Context, teamId string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.lastRefresh.IsZero() {
		return false, ErrCacheNotReady
	}

	_, ok := m.teams[teamId]
	return ok, nil
}

func (m *memoryEventTeamCache) LastRefresh() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastRefresh
}
