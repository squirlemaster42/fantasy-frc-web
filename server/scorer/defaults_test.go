package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, 3, d.qualWinPoints)
	assert.Equal(t, 1, d.energizedBonusPoints)
	assert.Equal(t, 1, d.superchargedBonusPoints)
	assert.Equal(t, 2, d.traversalBonusPoints)
	assert.Equal(t, 18, d.playoffFinalsPoints)
	assert.Equal(t, 15, d.playoffUpperBracketPoints)
	assert.Equal(t, 9, d.playoffLowerBracketPoints)
	assert.Equal(t, 2, d.einsteinMultiplier)
	assert.Equal(t, 2, d.alliancePickMultiplier)
}

func TestLoadDefaults_Override(t *testing.T) {
	t.Setenv("SCORER_QUAL_WIN_POINTS", "5")
	t.Setenv("SCORER_ENERGIZED_BONUS_POINTS", "2")
	t.Setenv("SCORER_SUPERCHARGED_BONUS_POINTS", "3")
	t.Setenv("SCORER_TRAVERSAL_BONUS_POINTS", "4")
	t.Setenv("SCORER_PLAYOFF_FINALS_POINTS", "20")
	t.Setenv("SCORER_PLAYOFF_UPPER_BRACKET_POINTS", "16")
	t.Setenv("SCORER_PLAYOFF_LOWER_BRACKET_POINTS", "10")
	t.Setenv("SCORER_EINSTEIN_MULTIPLIER", "3")
	t.Setenv("SCORER_ALLIANCE_PICK_MULTIPLIER", "4")

	d := loadDefaults()

	assert.Equal(t, 5, d.qualWinPoints)
	assert.Equal(t, 2, d.energizedBonusPoints)
	assert.Equal(t, 3, d.superchargedBonusPoints)
	assert.Equal(t, 4, d.traversalBonusPoints)
	assert.Equal(t, 20, d.playoffFinalsPoints)
	assert.Equal(t, 16, d.playoffUpperBracketPoints)
	assert.Equal(t, 10, d.playoffLowerBracketPoints)
	assert.Equal(t, 3, d.einsteinMultiplier)
	assert.Equal(t, 4, d.alliancePickMultiplier)
}
