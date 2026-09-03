package handler

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"server/cache"
	"server/model"
)

// collectDraftAvatarColors gathers the unique team numbers from a draft score
// and returns their complementary avatar background colors.
func collectDraftAvatarColors(ctx context.Context, store cache.AvatarStoreInterface, players []model.DraftPlayer) map[string]string {
	teamNums := make([]string, 0)
	for _, player := range players {
		for _, pick := range player.Picks {
			if pick.Pick.Valid {
				teamNums = append(teamNums, strings.TrimPrefix(pick.Pick.String, "frc"))
			}
		}
	}
	return fetchAvatarColors(ctx, store, teamNums)
}

// collectLeaderboardAvatarColors gathers the unique team numbers from a
// leaderboard page and returns their complementary avatar background colors.
func collectLeaderboardAvatarColors(ctx context.Context, store cache.AvatarStoreInterface, entries []model.LeaderboardEntry) map[string]string {
	teamNums := make([]string, 0)
	for _, entry := range entries {
		for _, pick := range entry.Picks {
			if pick.Pick.Valid {
				teamNums = append(teamNums, strings.TrimPrefix(pick.Pick.String, "frc"))
			}
		}
	}
	return fetchAvatarColors(ctx, store, teamNums)
}

// fetchAvatarColors returns a map of team number (without the "frc" prefix) to
// the complementary background color for that team's avatar. Lookups are run in
// parallel with a small bounded worker pool so a page with many teams doesn't
// serialize a large number of Redis/TBA calls.
func fetchAvatarColors(ctx context.Context, store cache.AvatarStoreInterface, teamNums []string) map[string]string {
	if store == nil {
		return map[string]string{}
	}

	unique := make(map[string]struct{}, len(teamNums))
	for _, n := range teamNums {
		unique[n] = struct{}{}
	}

	colors := make(map[string]string, len(unique))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for n := range unique {
		wg.Add(1)
		sem <- struct{}{}
		go func(teamNum string) {
			defer wg.Done()
			defer func() { <-sem }()

			num, err := strconv.Atoi(teamNum)
			if err != nil {
				return
			}

			color := store.GetAvatarColor(ctx, num)
			mu.Lock()
			colors[teamNum] = color
			mu.Unlock()
		}(n)
	}

	wg.Wait()
	return colors
}
