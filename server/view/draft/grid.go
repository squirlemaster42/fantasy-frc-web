package draft

import "server/model"

// PickGridClass returns the Tailwind grid classes to use for the pick board.
// It counts only accepted (non-pending) players, since pending players are not
// rendered in the pick grid.
func PickGridClass(players []model.DraftPlayer) string {
	numPlayers := 0
	for _, player := range players {
		if !player.Pending {
			numPlayers++
		}
	}

	switch {
	case numPlayers <= 2:
		return "grid-cols-1 md:grid-cols-2"
	case numPlayers <= 4:
		return "grid-cols-1 md:grid-cols-2 lg:grid-cols-4"
	case numPlayers <= 8:
		return "grid-cols-1 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8"
	default:
		// 9-16 players; keep the same 8-column layout and allow wrapping.
		return "grid-cols-1 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8"
	}
}
