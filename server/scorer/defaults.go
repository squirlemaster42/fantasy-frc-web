package scorer

import (
	"sync"

	"server/config"
)

const (
	// Sentinel value indicating a match object has not been loaded from TBA yet.
	unloadedMatchNumber = 0
)

// AllianceSelectionBaseScores maps alliance rank (1-8) to the base points awarded
// to each of the alliance's four picks. The final score is multiplied by
// AlliancePickMultiplier().
var AllianceSelectionBaseScores = map[int][]int{
	1: {32, 31, 9, 8},
	2: {30, 29, 10, 7},
	3: {28, 27, 11, 6},
	4: {26, 25, 12, 5},
	5: {24, 23, 13, 4},
	6: {22, 21, 14, 3},
	7: {20, 19, 15, 2},
	8: {18, 17, 16, 1},
}

// UpperBracketMatchIds and LowerBracketMatchIds identify playoff bracket
// positions for score assignment.
var UpperBracketMatchIds = map[int32]bool{
	1:  true,
	2:  true,
	3:  true,
	4:  true,
	7:  true,
	8:  true,
	11: true,
}

var LowerBracketMatchIds = map[int32]bool{
	5:  true,
	6:  true,
	9:  true,
	10: true,
	12: true,
	13: true,
}

var (
	defaultsOnce sync.Once
	defaults     scorerDefaults
)

type scorerDefaults struct {
	QualWinPoints             int `env:"SCORER_QUAL_WIN_POINTS" default:"3"`
	EnergizedBonusPoints      int `env:"SCORER_ENERGIZED_BONUS_POINTS" default:"1"`
	SuperchargedBonusPoints   int `env:"SCORER_SUPERCHARGED_BONUS_POINTS" default:"1"`
	TraversalBonusPoints      int `env:"SCORER_TRAVERSAL_BONUS_POINTS" default:"2"`
	PlayoffFinalsPoints       int `env:"SCORER_PLAYOFF_FINALS_POINTS" default:"18"`
	PlayoffUpperBracketPoints int `env:"SCORER_PLAYOFF_UPPER_BRACKET_POINTS" default:"15"`
	PlayoffLowerBracketPoints int `env:"SCORER_PLAYOFF_LOWER_BRACKET_POINTS" default:"9"`
	EinsteinMultiplier        int `env:"SCORER_EINSTEIN_MULTIPLIER" default:"2"`
	AlliancePickMultiplier    int `env:"SCORER_ALLIANCE_PICK_MULTIPLIER" default:"2"`
}

func loadDefaults() scorerDefaults {
	var d scorerDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *scorerDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

func QualWinPoints() int             { return getDefaults().QualWinPoints }
func EnergizedBonusPoints() int      { return getDefaults().EnergizedBonusPoints }
func SuperchargedBonusPoints() int   { return getDefaults().SuperchargedBonusPoints }
func TraversalBonusPoints() int      { return getDefaults().TraversalBonusPoints }
func PlayoffFinalsPoints() int       { return getDefaults().PlayoffFinalsPoints }
func PlayoffUpperBracketPoints() int { return getDefaults().PlayoffUpperBracketPoints }
func PlayoffLowerBracketPoints() int { return getDefaults().PlayoffLowerBracketPoints }
func EinsteinMultiplier() int        { return getDefaults().EinsteinMultiplier }
func AlliancePickMultiplier() int    { return getDefaults().AlliancePickMultiplier }
