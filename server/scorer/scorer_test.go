package scorer

import (
	"os"
	"path/filepath"
	"server/logging"
	"server/tbaHandler"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func getTbaTok() string {
    godotenv.Load(filepath.Join("../", ".env"))
    return os.Getenv("TBA_TOKEN")
}

func TestCompareMatches(t *testing.T) {
    s := NewScorer(nil, nil, nil)
    assert.True(t, s.compareMatchOrder("2024cur_f1m1", "2024cur_f1m2"))
    assert.False(t, s.compareMatchOrder("2024cur_f1m1", "2024cur_qm1"))
    assert.True(t, s.compareMatchOrder("2024cur_qm10", "2024cur_qm112"))
    assert.False(t, s.compareMatchOrder("2024cur_qm116", "2024cur_qm11"))
    assert.True(t, s.compareMatchOrder("2024cur_sf2m1", "2024cur_sf9m1"))
    assert.False(t, s.compareMatchOrder("2024cur_f1m2", "2024cur_sf12m1"))
    assert.True(t, s.compareMatchOrder("2024cur_qm90", "2024cur_sf12m1"))
    assert.False(t, s.compareMatchOrder("2024cur_sf12m1", "2024cur_qm72"))
}

func TestGetMatchLevel(t *testing.T) {
    assert.Equal(t, "f", getMatchLevel("2024cur_f1m2"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm1"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm112"))
    assert.Equal(t, "qm", getMatchLevel("2024cur_qm11"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf9m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "sf", getMatchLevel("2024cur_sf12m1"))
    assert.Equal(t, "qm", getMatchLevel("2025cur_qm72"))
}

func TestSortMatchOrder(t *testing.T) {
    unsorted := []string{
        "2024cur_f1m1",
        "2024cur_qf1m1",
        "2024cur_qm1",
        "2024cur_qm100",
        "2024cur_sf1m1",
        "2024cur_sf12m1",
        "2024cur_f1m2",
        "2024cur_qm52",
    }

    s := NewScorer(nil, nil, nil)
    sorted := s.sortMatchesByPlayOrder(unsorted)

    standard := []string{
        "2024cur_qm1",
        "2024cur_qm52",
        "2024cur_qm100",
        "2024cur_qf1m1",
        "2024cur_sf1m1",
        "2024cur_sf12m1",
        "2024cur_f1m1",
        "2024cur_f1m2",
    }

    assert.True(t, len(sorted) == len(standard), "Sorted array is not the correct length")

    for i, match := range standard {
        assert.Equal(t, match, sorted[i])
    }
}

func TestScoreMatches(t *testing.T) {
    //We should not need a tba handler or database
    logger := logging.NewLogger(&logging.TimestampedLogger{})
    logger.Start()
    tbaHandler := tbaHandler.NewHandler(getTbaTok(), logger)
    scorer := NewScorer(tbaHandler, nil, logger)

    match := tbaHandler.MakeMatchReq("2025mawor_qm71")
    scoredMatch, _ := scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 10, scoredMatch.RedScore)
    assert.Equal(t, 2, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm64")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 12, scoredMatch.RedScore)
    assert.Equal(t, 4, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm60")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 2, scoredMatch.RedScore)
    assert.Equal(t, 8, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm52")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 2, scoredMatch.RedScore)
    assert.Equal(t, 12, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm46")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 2, scoredMatch.RedScore)
    assert.Equal(t, 12, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm40")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 10, scoredMatch.RedScore)
    assert.Equal(t, 2, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_qm36")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 2, scoredMatch.RedScore)
    assert.Equal(t, 12, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_sf4m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 15, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_sf6m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 9, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2025mawor_f1m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 18, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    /*
    match = tbaHandler.MakeMatchReq("2024cmptx_sf2m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 30, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cmptx_sf12m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 18, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cmptx_f1m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 36, scoredMatch.BlueScore)
    */
}

func TestGetAllianceSelectionScores (t *testing.T) {
    logger := logging.NewLogger(&logging.TimestampedLogger{})
    logger.Start()
    tbaHandler := tbaHandler.NewHandler(getTbaTok(), logger)
    alliances := tbaHandler.MakeEliminationAllianceRequest("2025mawor")
    scorer := NewScorer(tbaHandler, nil, logger)
    allianceOneScores := scorer.GetAllianceSelectionScore(alliances[0])
    assert.EqualValues(t, 32, allianceOneScores["frc190"])
    assert.EqualValues(t, 31, allianceOneScores["frc1768"])
    assert.EqualValues(t, 9, allianceOneScores["frc3182"])
    allianceTwoScores := scorer.GetAllianceSelectionScore(alliances[1])
    assert.EqualValues(t, 30, allianceTwoScores["frc125"])
    assert.EqualValues(t, 29, allianceTwoScores["frc88"])
    assert.EqualValues(t, 10, allianceTwoScores["frc8626"])
    allianceThreeScores := scorer.GetAllianceSelectionScore(alliances[2])
    assert.EqualValues(t, 28, allianceThreeScores["frc1153"])
    assert.EqualValues(t, 27, allianceThreeScores["frc230"])
    assert.EqualValues(t, 11, allianceThreeScores["frc2079"])
    allianceFourScores := scorer.GetAllianceSelectionScore(alliances[3])
    assert.EqualValues(t, 26, allianceFourScores["frc2370"])
    assert.EqualValues(t, 25, allianceFourScores["frc1100"])
    assert.EqualValues(t, 12, allianceFourScores["frc1757"])
    allianceFiveScores := scorer.GetAllianceSelectionScore(alliances[4])
    assert.EqualValues(t, 24, allianceFiveScores["frc1277"])
    assert.EqualValues(t, 23, allianceFiveScores["frc2067"])
    assert.EqualValues(t, 13, allianceFiveScores["frc126"])
    allianceSixScores := scorer.GetAllianceSelectionScore(alliances[5])
    assert.EqualValues(t, 22, allianceSixScores["frc5459"])
    assert.EqualValues(t, 21, allianceSixScores["frc1699"])
    assert.EqualValues(t, 14, allianceSixScores["frc1740"])
    allianceSevenScores := scorer.GetAllianceSelectionScore(alliances[6])
    assert.EqualValues(t, 20, allianceSevenScores["frc5000"])
    assert.EqualValues(t, 19, allianceSevenScores["frc1735"])
    assert.EqualValues(t, 15, allianceSevenScores["frc1119"])
    allianceEightScores := scorer.GetAllianceSelectionScore(alliances[7])
    assert.EqualValues(t, 18, allianceEightScores["frc7153"])
    assert.EqualValues(t, 17, allianceEightScores["frc5422"])
    assert.EqualValues(t, 16, allianceEightScores["frc9644"])
}
