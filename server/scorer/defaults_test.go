package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaults_DefaultValues(t *testing.T) {
	d := loadDefaults()

	assert.Equal(t, 3, d.QualWinPoints)
	assert.Equal(t, 1, d.EnergizedBonusPoints)
	assert.Equal(t, 1, d.SuperchargedBonusPoints)
	assert.Equal(t, 2, d.TraversalBonusPoints)
	assert.Equal(t, 18, d.PlayoffFinalsPoints)
	assert.Equal(t, 15, d.PlayoffUpperBracketPoints)
	assert.Equal(t, 9, d.PlayoffLowerBracketPoints)
	assert.Equal(t, 2, d.EinsteinMultiplier)
	assert.Equal(t, 2, d.AlliancePickMultiplier)
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

	assert.Equal(t, 5, d.QualWinPoints)
	assert.Equal(t, 2, d.EnergizedBonusPoints)
	assert.Equal(t, 3, d.SuperchargedBonusPoints)
	assert.Equal(t, 4, d.TraversalBonusPoints)
	assert.Equal(t, 20, d.PlayoffFinalsPoints)
	assert.Equal(t, 16, d.PlayoffUpperBracketPoints)
	assert.Equal(t, 10, d.PlayoffLowerBracketPoints)
	assert.Equal(t, 3, d.EinsteinMultiplier)
	assert.Equal(t, 4, d.AlliancePickMultiplier)
}
