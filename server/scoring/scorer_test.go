package scoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func getTbaTok() string {
    godotenv.Load(filepath.Join("../", ".env"))
    return os.Getenv("TBA_TOKEN")
}

func TestScoreSingleMatchWithNoRankingPoints(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatch := tbaHandler.makeMatchReq("2024cur_qm10")
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 0)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)
}

func TestScoreSingleMatchWithOneRankingPoint(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatchId := "2024cur_qm10"
    testMatch := tbaHandler.makeMatchReq(testMatchId)
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.Equal(t, testMatchId, scoredMatch.TbaId)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 0)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)

}

func TestScoreSingleMatchWithOtherRankingPoint(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatchId := "2024cur_qm10"
    testMatch := tbaHandler.makeMatchReq(testMatchId)
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 0)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)

}

func TestScoreSingleMatchWithBothRankingPoints(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatchId := "2024cur_qm10"
    testMatch := tbaHandler.makeMatchReq(testMatchId)
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 0)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)

}

func TestScoringMatchWithSurrogate(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatchId := "2024cur_qm10"
    testMatch := tbaHandler.makeMatchReq(testMatchId)
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 1)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)

}

func TestScoringMatchWithDq(t *testing.T) {
    //TODO Load tba token
    tbaHandler := NewHandler(getTbaTok())
    //TODO Check that this is a good match
    testMatchId := "2024cur_qm10"
    testMatch := tbaHandler.makeMatchReq(testMatchId)
    //We should write these tests so they dont need the database
    scorer := NewScorer(tbaHandler, nil)
    scoredMatch := scorer.scoreMatch(testMatch)
    assert.True(t, scoredMatch.Played)
    assert.Len(t, scoredMatch.Dqed, 1)
    assert.Equal(t, 8, scoredMatch.RedAllianceScore)
    assert.Equal(t, 8, scoredMatch.BlueAllianceScore)
}

func TestScoreSingleTeam(t *testing.T) {

}

func TestScoringTeamWithSurrogateMatch(t *testing.T) {

}

func TestScoringTeamWithDqMatch(t *testing.T) {

}

func TestCompleteTeamAllianceSelectionScore(t *testing.T) {

}

func ScoreSinglePlayer(t *testing.T) {

}
