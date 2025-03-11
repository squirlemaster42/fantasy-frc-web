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

    match := tbaHandler.MakeMatchReq("2024cur_qm2")
    scoredMatch, _ := scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 8, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cur_qm3")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 8, scoredMatch.RedScore)
    assert.Equal(t, 2, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cur_qm17")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 8, scoredMatch.RedScore)
    assert.Equal(t, 4, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cur_sf2m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 15, scoredMatch.RedScore)
    assert.Equal(t, 0, scoredMatch.BlueScore)

    match = tbaHandler.MakeMatchReq("2024cur_sf12m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 9, scoredMatch.BlueScore)

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

    match = tbaHandler.MakeMatchReq("2024cur_f1m1")
    scoredMatch, _ = scorer.scoreMatch(match, true)
    assert.True(t, scoredMatch.Played)
    assert.Equal(t, 0, scoredMatch.RedScore)
    assert.Equal(t, 18, scoredMatch.BlueScore)
}

func TestScoreTeamRankings(t *testing.T) {
    logger := logging.NewLogger(&logging.TimestampedLogger{})
    logger.Start()
    tbaHandler := tbaHandler.NewHandler(getTbaTok(), logger)
    scorer := NewScorer(tbaHandler, nil, logger)
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc2200"))
    assert.Equal(t, 42, scorer.getTeamRankingScore("frc3847"))
    assert.Equal(t, 16, scorer.getTeamRankingScore("frc624"))
    assert.Equal(t, 26, scorer.getTeamRankingScore("frc503"))
    assert.Equal(t, 34, scorer.getTeamRankingScore("frc2521"))
    assert.Equal(t, 36, scorer.getTeamRankingScore("frc8608"))
    assert.Equal(t, 18, scorer.getTeamRankingScore("frc7226"))
    assert.Equal(t, 2, scorer.getTeamRankingScore("frc5687"))
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc254"))
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc1678"))
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc1690"))
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc1323"))
    assert.Equal(t, 48, scorer.getTeamRankingScore("frc1771"))
}
