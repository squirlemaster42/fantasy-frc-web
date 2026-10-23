package cache

import (
	"context"
	"errors"
	"time"

	"server/log"
	"server/tbaHandler"
)

// EventTeamCache stores the set of teams that are attending the configured TBA
// events for the current season. It is used to validate picks without making a
// per-pick request to The Blue Alliance.
type EventTeamCache interface {
	IsTeamAtEvent(ctx context.Context, teamId string) (bool, error)
	Refresh(ctx context.Context) error
	InvalidateAndRefresh(ctx context.Context) error
	LastRefresh() time.Time
}

// ErrCacheNotReady is returned when the cache has never been successfully
// populated.
var ErrCacheNotReady = errors.New("pick validation cache is not ready")

// fetchEventTeams queries The Blue Alliance for the teams at each configured
// event and returns a deduplicated set of team keys.
func fetchEventTeams(ctx context.Context, handler tbaHandler.TBAInterface, events []string) (map[string]struct{}, error) {
	teams := make(map[string]struct{})
	for _, event := range events {
		log.Debug(ctx, "Fetching teams at event for pick validation cache", "event", event)
		eventTeams, err := handler.MakeTeamsAtEventRequest(ctx, event)
		if err != nil {
			return nil, err
		}
		for _, team := range eventTeams {
			teams[team.Key] = struct{}{}
		}
	}
	return teams, nil
}

// teamSlice returns the unique team keys from a map as a slice of empty-interface
// values suitable for Redis SAdd.
func teamSlice(teams map[string]struct{}) []interface{} {
	members := make([]interface{}, 0, len(teams))
	for team := range teams {
		members = append(members, team)
	}
	return members
}

