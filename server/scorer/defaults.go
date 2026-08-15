package scorer

import (
	"server/utils"
	"sync"
)

const (
	scorerQualWinPointsEnvKey = "SCORER_QUAL_WIN_POINTS"
	defaultScorerQualWinPoints = 3

	scorerEnergizedBonusPointsEnvKey = "SCORER_ENERGIZED_BONUS_POINTS"
	defaultScorerEnergizedBonusPoints  = 1

	scorerSuperchargedBonusPointsEnvKey = "SCORER_SUPERCHARGED_BONUS_POINTS"
	defaultScorerSuperchargedBonusPoints  = 1

	scorerTraversalBonusPointsEnvKey = "SCORER_TRAVERSAL_BONUS_POINTS"
	defaultScorerTraversalBonusPoints  = 2

	scorerPlayoffFinalsPointsEnvKey = "SCORER_PLAYOFF_FINALS_POINTS"
	defaultScorerPlayoffFinalsPoints = 18

	scorerPlayoffUpperBracketPointsEnvKey = "SCORER_PLAYOFF_UPPER_BRACKET_POINTS"
	defaultScorerPlayoffUpperBracketPoints = 15

	scorerPlayoffLowerBracketPointsEnvKey = "SCORER_PLAYOFF_LOWER_BRACKET_POINTS"
	defaultScorerPlayoffLowerBracketPoints = 9

	scorerEinsteinMultiplierEnvKey = "SCORER_EINSTEIN_MULTIPLIER"
	defaultScorerEinsteinMultiplier = 2

	scorerAlliancePickMultiplierEnvKey = "SCORER_ALLIANCE_PICK_MULTIPLIER"
	defaultScorerAlliancePickMultiplier = 2

	// Sentinel value indicating a match object has not been loaded from TBA yet.
	unloadedMatchNumber = 0
)

// AllianceSelectionBaseScores maps alliance rank (1-8) to the base points awarded
// to each of the alliance's four picks. The final score is multiplied by
// ScorerAlliancePickMultiplier().
var AllianceSelectionBaseScores = map[int][]int16{
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
	qualWinPoints          int
	energizedBonusPoints   int
	superchargedBonusPoints int
	traversalBonusPoints   int
	playoffFinalsPoints    int
	playoffUpperBracketPoints int
	playoffLowerBracketPoints int
	einsteinMultiplier     int
	alliancePickMultiplier int
}

func loadDefaults() scorerDefaults {
	return scorerDefaults{
		qualWinPoints:             utils.MustGetEnvInt(scorerQualWinPointsEnvKey, defaultScorerQualWinPoints),
		energizedBonusPoints:      utils.MustGetEnvInt(scorerEnergizedBonusPointsEnvKey, defaultScorerEnergizedBonusPoints),
		superchargedBonusPoints:   utils.MustGetEnvInt(scorerSuperchargedBonusPointsEnvKey, defaultScorerSuperchargedBonusPoints),
		traversalBonusPoints:      utils.MustGetEnvInt(scorerTraversalBonusPointsEnvKey, defaultScorerTraversalBonusPoints),
		playoffFinalsPoints:       utils.MustGetEnvInt(scorerPlayoffFinalsPointsEnvKey, defaultScorerPlayoffFinalsPoints),
		playoffUpperBracketPoints: utils.MustGetEnvInt(scorerPlayoffUpperBracketPointsEnvKey, defaultScorerPlayoffUpperBracketPoints),
		playoffLowerBracketPoints: utils.MustGetEnvInt(scorerPlayoffLowerBracketPointsEnvKey, defaultScorerPlayoffLowerBracketPoints),
		einsteinMultiplier:        utils.MustGetEnvInt(scorerEinsteinMultiplierEnvKey, defaultScorerEinsteinMultiplier),
		alliancePickMultiplier:    utils.MustGetEnvInt(scorerAlliancePickMultiplierEnvKey, defaultScorerAlliancePickMultiplier),
	}
}

func getDefaults() *scorerDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

func QualWinPoints() int                { return getDefaults().qualWinPoints }
func EnergizedBonusPoints() int         { return getDefaults().energizedBonusPoints }
func SuperchargedBonusPoints() int      { return getDefaults().superchargedBonusPoints }
func TraversalBonusPoints() int         { return getDefaults().traversalBonusPoints }
func PlayoffFinalsPoints() int          { return getDefaults().playoffFinalsPoints }
func PlayoffUpperBracketPoints() int    { return getDefaults().playoffUpperBracketPoints }
func PlayoffLowerBracketPoints() int    { return getDefaults().playoffLowerBracketPoints }
func EinsteinMultiplier() int           { return getDefaults().einsteinMultiplier }
func AlliancePickMultiplier() int       { return getDefaults().alliancePickMultiplier }
