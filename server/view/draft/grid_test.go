package draft

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"server/model"
)

func TestPickGridClass(t *testing.T) {
	makePlayers := func(count int, pending bool) []model.DraftPlayer {
		players := make([]model.DraftPlayer, count)
		for i := range players {
			players[i] = model.DraftPlayer{Pending: pending}
		}
		return players
	}

	t.Run("single accepted player", func(t *testing.T) {
		assert.Contains(t, PickGridClass(makePlayers(1, false)), "grid-cols-1")
	})

	t.Run("two accepted players", func(t *testing.T) {
		class := PickGridClass(makePlayers(2, false))
		assert.Contains(t, class, "md:grid-cols-2")
	})

	t.Run("four accepted players", func(t *testing.T) {
		class := PickGridClass(makePlayers(4, false))
		assert.Contains(t, class, "lg:grid-cols-4")
	})

	t.Run("eight accepted players", func(t *testing.T) {
		class := PickGridClass(makePlayers(8, false))
		assert.Contains(t, class, "xl:grid-cols-8")
	})

	t.Run("sixteen accepted players wraps at eight columns", func(t *testing.T) {
		class := PickGridClass(makePlayers(16, false))
		assert.Contains(t, class, "xl:grid-cols-8")
	})

	t.Run("pending players are not counted", func(t *testing.T) {
		// 4 pending + 2 accepted should use the 2-player layout.
		players := append(makePlayers(4, true), makePlayers(2, false)...)
		class := PickGridClass(players)
		assert.Contains(t, class, "md:grid-cols-2")
	})
}
